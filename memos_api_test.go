package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemosClientLoadsCurrentAPIAndAllPages(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/memos" {
			t.Errorf("path = %q, want /api/memos", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization header = %q", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "1000" {
			t.Errorf("pageSize = %q, want 1000", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("pageToken") == "next-page" {
			fmt.Fprint(w, `{"memos":[{"name":"memos/second","content":"# Second memo\nDetails","tags":["work"],"createTime":"2026-11-12T10:00:00Z"}]}`)
			return
		}
		fmt.Fprint(w, `{"memos":[{"name":"memos/first","content":"Body","property":{"title":"First memo"},"tags":["notes"],"updateTime":"2026-11-13T10:30:00Z"}],"nextPageToken":"next-page"}`)
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.listMemos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(items) != 2 {
		t.Fatalf("requests = %d, memos = %d; want 2 and 2", requests, len(items))
	}
	if items[0].ID != "memos/first" || items[0].Title != "First memo" || items[0].Body != "Body" || items[0].Tags[0] != "notes" {
		t.Fatalf("first memo mapping = %+v", items[0])
	}
	if items[1].Title != "Second memo" || items[1].Updated == "" {
		t.Fatalf("fallback title/time mapping = %+v", items[1])
	}
}

func TestMemosClientFallsBackToV1API(t *testing.T) {
	var currentCalls, legacyCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/memos":
			currentCalls++
			http.NotFound(w, r)
		case "/api/v1/memos":
			legacyCalls++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"memos":[{"name":"memos/legacy","content":"legacy note"}]}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.listMemos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if currentCalls != 1 || legacyCalls != 1 || len(items) != 1 || items[0].Title != "legacy note" {
		t.Fatalf("calls current=%d legacy=%d; items=%+v", currentCalls, legacyCalls, items)
	}
}

func TestMemosClientCreatesAndUpdatesMemoContent(t *testing.T) {
	var createCalls, updateCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer write-token" {
			t.Errorf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode memo body: %v", err)
		}
		if len(body) != 1 {
			t.Errorf("body fields = %v, want content only", body)
		}
		switch r.Method {
		case http.MethodPost:
			createCalls++
			if r.URL.Path != "/api/memos" || r.URL.Query().Get("updateMask") != "" {
				t.Errorf("create request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			if body["content"] != "# Created memo\nBody" {
				t.Errorf("create content = %v", body["content"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"memos/created","content":"# Created memo\nBody","createTime":"2026-11-13T10:00:00Z"}`)
		case http.MethodPatch:
			updateCalls++
			if r.URL.Path != "/api/memos/existing" || r.URL.Query().Get("updateMask") != "content" {
				t.Errorf("update request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			if body["content"] != "Updated body" {
				t.Errorf("update content = %v", body["content"])
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"memos/existing","content":"Updated body","updateTime":"2026-11-13T11:00:00Z"}`)
		default:
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "write-token")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.createMemo(context.Background(), "# Created memo\nBody")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := client.updateMemo(context.Background(), "memos/existing", "Updated body")
	if err != nil {
		t.Fatal(err)
	}
	if createCalls != 1 || updateCalls != 1 {
		t.Fatalf("create calls = %d, update calls = %d", createCalls, updateCalls)
	}
	if created.ID != "memos/created" || created.Title != "Created memo" || created.Body != "# Created memo\nBody" {
		t.Fatalf("created memo = %+v", created)
	}
	if updated.ID != "memos/existing" || updated.Body != "Updated body" {
		t.Fatalf("updated memo = %+v", updated)
	}
}

func TestMemosClientCreateFallsBackToV1API(t *testing.T) {
	var current, legacy int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		switch r.URL.Path {
		case "/api/memos":
			current++
			http.NotFound(w, r)
		case "/api/v1/memos":
			legacy++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"memos/legacy","content":"Legacy memo"}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.createMemo(context.Background(), "Legacy memo")
	if err != nil {
		t.Fatal(err)
	}
	if current != 1 || legacy != 1 || created.ID != "memos/legacy" {
		t.Fatalf("current=%d legacy=%d created=%+v", current, legacy, created)
	}
}

func TestMemosClientDoesNotExposeTokenInErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "do-not-print-this-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.listMemos(context.Background())
	if err == nil || !strings.Contains(err.Error(), "rejected the credentials") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), client.accessToken) {
		t.Fatal("error contains the access token")
	}
}

func TestPasswordSignInReadsMemosAndRefreshesSession(t *testing.T) {
	var signInCalls, refreshCalls, listCalls, createCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/auth/signin":
			signInCalls++
			var request struct {
				PasswordCredentials struct {
					Username string `json:"username"`
					Password string `json:"password"`
				} `json:"passwordCredentials"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode sign-in request: %v", err)
			}
			if request.PasswordCredentials.Username != "test-user" || request.PasswordCredentials.Password != "test-password" {
				t.Errorf("unexpected sign-in request credentials")
			}
			http.SetCookie(w, &http.Cookie{Name: "memos_refresh_token", Value: "refresh-test", Path: "/api/auth", HttpOnly: true})
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"accessToken":"expired-token","user":{"username":"test-user","displayName":"Test User"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/auth/refresh":
			refreshCalls++
			cookie, err := r.Cookie("memos_refresh_token")
			if err != nil || cookie.Value != "refresh-test" {
				t.Errorf("refresh cookie = %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"accessToken":"refreshed-token"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/memos":
			createCalls++
			if r.Header.Get("Authorization") == "Bearer expired-token" {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer refreshed-token" {
				t.Errorf("create authorization = %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"memos/created","content":"Created memo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/memos":
			listCalls++
			if r.Header.Get("Authorization") == "Bearer expired-token" {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer refreshed-token" {
				t.Errorf("list authorization = %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"memos":[{"name":"memos/live","content":"Live memo","property":{"title":"Live memo"}}]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newMemosClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	displayName, err := client.signIn(context.Background(), "test-user", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.createMemo(context.Background(), "Created memo")
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.listMemos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if displayName != "Test User" || signInCalls != 1 || refreshCalls != 1 || createCalls != 2 || listCalls != 1 || created.ID != "memos/created" || len(items) != 1 || items[0].ID != "memos/live" {
		t.Fatalf("user=%q sign-ins=%d refreshes=%d creates=%d lists=%d created=%+v memos=%+v", displayName, signInCalls, refreshCalls, createCalls, listCalls, created, items)
	}
}

func TestPasswordSignInRequiresTLSForRemoteHosts(t *testing.T) {
	client, err := newMemosClient("http://memos.example.test", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.signIn(context.Background(), "test-user", "test-password")
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("sign-in error = %v, want HTTPS requirement", err)
	}
}

func TestNewAppFromEnvShowsSignInWhenConfiguredWithoutToken(t *testing.T) {
	t.Setenv("MEMOS_SERVER_URL", "https://memos.example.test")
	t.Setenv("MEMOS_ACCESS_TOKEN", "")
	a := newAppFromEnv()
	if !a.liveData || len(a.memos) != 0 || a.loadErr != "" || a.client == nil || a.authenticated {
		t.Fatalf("configured app state = live %t, memos %d, error %q, client %t", a.liveData, len(a.memos), a.loadErr, a.client != nil)
	}
}

func TestNewAppFromEnvUsesSamplesWithoutServerURL(t *testing.T) {
	t.Setenv("MEMOS_SERVER_URL", "")
	t.Setenv("MEMOS_ACCESS_TOKEN", "ignored")
	a := newAppFromEnv()
	if a.liveData || len(a.memos) == 0 {
		t.Fatalf("unconfigured app state = live %t, memos %d", a.liveData, len(a.memos))
	}
}
