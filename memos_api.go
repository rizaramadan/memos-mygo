package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const memosPageSize = 1000

type memosClient struct {
	baseURL      *url.URL
	accessToken  string
	sessionLogin bool
	http         *http.Client
}

type memosListResponse struct {
	Memos         []memosAPIMemo `json:"memos"`
	NextPageToken string         `json:"nextPageToken"`
}

type memosAPIMemo struct {
	Name       string   `json:"name"`
	UID        string   `json:"uid"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	CreateTime string   `json:"createTime"`
	UpdateTime string   `json:"updateTime"`
	Property   struct {
		Title string `json:"title"`
	} `json:"property"`
}

type memosHTTPError struct {
	status int
}

type memosAuthError struct {
	message string
}

func (e *memosAuthError) Error() string { return e.message }

func (e *memosHTTPError) Error() string {
	switch e.status {
	case http.StatusUnauthorized:
		return "Memos rejected the credentials. Sign in again or check the access token."
	case http.StatusForbidden:
		return "Memos denied this request. Check your account access."
	default:
		return fmt.Sprintf("Memos returned HTTP %d.", e.status)
	}
}

func newMemosClient(serverURL, token string) (*memosClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Set MEMOS_SERVER_URL to an http or https server address, without a token or query string.")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(parsed.Path, "/api/v1") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/api/v1")
	} else if strings.HasSuffix(parsed.Path, "/api") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/api")
	}
	jar, _ := cookiejar.New(nil)
	return &memosClient{
		baseURL:     parsed,
		accessToken: token,
		http:        &http.Client{Timeout: 20 * time.Second, Jar: jar},
	}, nil
}

func (c *memosClient) signIn(ctx context.Context, username, password string) (string, error) {
	c.accessToken = ""
	c.sessionLogin = false
	if c.baseURL.Scheme != "https" && !isLoopbackHost(c.baseURL.Hostname()) {
		return "", &memosAuthError{message: "Use an HTTPS server for password sign-in."}
	}
	credentials := struct {
		PasswordCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
	}{}
	credentials.PasswordCredentials.Username = username
	credentials.PasswordCredentials.Password = password
	body, err := json.Marshal(credentials)
	if err != nil {
		return "", errors.New("Could not prepare the Memos sign-in request.")
	}
	for _, apiPath := range []string{"/api/auth/signin", "/api/v1/auth/signin"} {
		endpoint := c.endpoint(apiPath)
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
		if err != nil {
			return "", errors.New("Could not build the Memos sign-in request.")
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
		response, err := c.http.Do(request)
		if err != nil {
			return "", errors.New("Could not reach the Memos server. Check the address and network, then retry.")
		}
		if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
			response.Body.Close()
			continue
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			status := response.StatusCode
			response.Body.Close()
			return "", signInHTTPError(status)
		}
		var result struct {
			AccessToken string `json:"accessToken"`
			User        struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"username"`
			} `json:"user"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result)
		response.Body.Close()
		if decodeErr != nil || result.AccessToken == "" {
			return "", &memosAuthError{message: "Memos returned an invalid sign-in response."}
		}
		c.accessToken = result.AccessToken
		c.sessionLogin = true
		name := result.User.DisplayName
		if name == "" {
			name = result.User.Username
		}
		return name, nil
	}
	return "", &memosAuthError{message: "This Memos server does not provide a supported password sign-in endpoint."}
}

func signInHTTPError(status int) error {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return &memosAuthError{message: "Sign-in failed. Check your username and password."}
	case http.StatusForbidden:
		return &memosAuthError{message: "Password sign-in is disabled on this Memos server."}
	default:
		return fmt.Errorf("Memos sign-in failed with HTTP %d.", status)
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func (c *memosClient) endpoint(apiPath string) *url.URL {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + apiPath
	endpoint.RawQuery = ""
	return &endpoint
}

func (c *memosClient) refreshAccessToken(ctx context.Context) error {
	for _, apiPath := range []string{"/api/auth/refresh", "/api/v1/auth/refresh"} {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(apiPath).String(), bytes.NewReader([]byte("{}")))
		if err != nil {
			return &memosAuthError{message: "Could not refresh the Memos session. Sign in again."}
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
		response, err := c.http.Do(request)
		if err != nil {
			return errors.New("Could not reach the Memos server to refresh the session.")
		}
		if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
			response.Body.Close()
			continue
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			response.Body.Close()
			return &memosAuthError{message: "Your Memos session expired. Sign in again."}
		}
		var result struct {
			AccessToken string `json:"accessToken"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result)
		response.Body.Close()
		if decodeErr != nil || result.AccessToken == "" {
			return &memosAuthError{message: "Memos could not refresh the session. Sign in again."}
		}
		c.accessToken = result.AccessToken
		return nil
	}
	return &memosAuthError{message: "Your Memos session expired. Sign in again."}
}

func (c *memosClient) clearSession() {
	c.accessToken = ""
	c.sessionLogin = false
	if c.http.Jar != nil {
		c.http.Jar.SetCookies(c.baseURL, nil)
	}
}

func (c *memosClient) listMemos(ctx context.Context) ([]memo, error) {
	items, err := c.listEndpoint(ctx, "/api/memos")
	var statusErr *memosHTTPError
	if errors.As(err, &statusErr) && (statusErr.status == http.StatusNotFound || statusErr.status == http.StatusGone) {
		return c.listEndpoint(ctx, "/api/v1/memos")
	}
	return items, err
}

func (c *memosClient) createMemo(ctx context.Context, content string) (memo, error) {
	var result memosAPIMemo
	err := c.mutateMemo(ctx, http.MethodPost, []string{"/api/memos", "/api/v1/memos"}, "", content, &result)
	if err != nil {
		return memo{}, err
	}
	return convertAPIMemo(result)
}

func (c *memosClient) updateMemo(ctx context.Context, memoName, content string) (memo, error) {
	id := strings.TrimPrefix(strings.TrimSpace(memoName), "memos/")
	if id == "" || strings.Contains(id, "/") {
		return memo{}, errors.New("Memos returned an invalid memo ID.")
	}
	var result memosAPIMemo
	paths := []string{
		"/api/memos/" + url.PathEscape(id),
		"/api/v1/memos/" + url.PathEscape(id),
	}
	err := c.mutateMemo(ctx, http.MethodPatch, paths, "content", content, &result)
	if err != nil {
		return memo{}, err
	}
	return convertAPIMemo(result)
}

func (c *memosClient) mutateMemo(ctx context.Context, method string, paths []string, updateMask, content string, result *memosAPIMemo) error {
	body, err := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: content})
	if err != nil {
		return errors.New("Could not prepare the Memos save request.")
	}
	for _, apiPath := range paths {
		endpoint := c.endpoint(apiPath)
		if updateMask != "" {
			query := endpoint.Query()
			query.Set("updateMask", updateMask)
			endpoint.RawQuery = query.Encode()
		}
		for attempt := 0; attempt < 2; attempt++ {
			request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
			if err != nil {
				return errors.New("Could not build the Memos save request.")
			}
			request.Header.Set("Accept", "application/json")
			request.Header.Set("Content-Type", "application/json")
			if c.accessToken != "" {
				request.Header.Set("Authorization", "Bearer "+c.accessToken)
			}
			response, err := c.http.Do(request)
			if err != nil {
				return errors.New("Could not reach the Memos server. Check the address and network, then retry.")
			}
			if response.StatusCode == http.StatusUnauthorized && attempt == 0 && c.sessionLogin {
				response.Body.Close()
				if err := c.refreshAccessToken(ctx); err != nil {
					return err
				}
				continue
			}
			if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
				response.Body.Close()
				if attempt == 0 {
					break
				}
				return &memosHTTPError{status: response.StatusCode}
			}
			if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
				status := response.StatusCode
				response.Body.Close()
				return &memosHTTPError{status: status}
			}
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(result)
			response.Body.Close()
			if decodeErr != nil {
				return errors.New("Memos saved the request but returned an unreadable response. Refresh the list before trying again.")
			}
			return nil
		}
	}
	return errors.New("This Memos server does not provide a supported save endpoint.")
}

func (c *memosClient) listEndpoint(ctx context.Context, apiPath string) ([]memo, error) {
	items := make([]memo, 0)
	pageToken := ""
	seenTokens := make(map[string]struct{})
	for page := 0; page < 100; page++ {
		response, err := c.fetchPage(ctx, apiPath, pageToken)
		if err != nil {
			return nil, err
		}
		for _, item := range response.Memos {
			converted, err := convertAPIMemo(item)
			if err != nil {
				return nil, err
			}
			items = append(items, converted)
		}
		if response.NextPageToken == "" {
			return items, nil
		}
		if _, seen := seenTokens[response.NextPageToken]; seen {
			return nil, errors.New("Memos returned a repeated page token.")
		}
		seenTokens[response.NextPageToken] = struct{}{}
		pageToken = response.NextPageToken
	}
	return nil, errors.New("Memos returned more than 100 pages. Narrow the server query and try again.")
}

func (c *memosClient) fetchPage(ctx context.Context, apiPath, pageToken string) (memosListResponse, error) {
	return c.fetchPageAttempt(ctx, apiPath, pageToken, true)
}

func (c *memosClient) fetchPageAttempt(ctx context.Context, apiPath, pageToken string, mayRefresh bool) (memosListResponse, error) {
	endpoint := c.endpoint(apiPath)
	query := endpoint.Query()
	query.Set("pageSize", fmt.Sprint(memosPageSize))
	if pageToken != "" {
		query.Set("pageToken", pageToken)
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return memosListResponse{}, errors.New("Could not build the Memos request.")
	}
	request.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return memosListResponse{}, errors.New("Could not reach the Memos server. Check the address and network, then retry.")
	}
	if response.StatusCode == http.StatusUnauthorized && mayRefresh && c.sessionLogin {
		response.Body.Close()
		if err := c.refreshAccessToken(ctx); err != nil {
			return memosListResponse{}, err
		}
		return c.fetchPageAttempt(ctx, apiPath, pageToken, false)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return memosListResponse{}, &memosHTTPError{status: response.StatusCode}
	}
	var result memosListResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<20)).Decode(&result); err != nil {
		return memosListResponse{}, errors.New("Memos returned an unreadable response.")
	}
	return result, nil
}

func convertAPIMemo(item memosAPIMemo) (memo, error) {
	id := strings.TrimSpace(item.Name)
	if id == "" {
		id = strings.TrimSpace(item.UID)
	}
	if id == "" {
		return memo{}, errors.New("Memos returned a note without an ID.")
	}
	title := strings.TrimSpace(item.Property.Title)
	if title == "" {
		title = titleFromContent(item.Content)
	}
	if title == "" {
		title = "Untitled memo"
	}
	updated := item.UpdateTime
	if updated == "" {
		updated = item.CreateTime
	}
	return memo{
		ID:      id,
		Title:   title,
		Body:    item.Content,
		Tags:    append([]string(nil), item.Tags...),
		Updated: formatMemosTime(updated),
	}, nil
}

func titleFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.TrimSpace(strings.TrimLeft(line, "# "))
	}
	return ""
}

func formatMemosTime(value string) string {
	if value == "" {
		return "Date unavailable"
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Local().Format("Jan 2, 2006 · 3:04 PM")
		}
	}
	return value
}

func (a *app) refreshMemos() {
	if a.client == nil || a.window == nil || a.loading || a.saving || !a.authenticated {
		return
	}
	a.loading = true
	a.loadErr = ""
	window := a.window
	client := a.client
	window.Update(func() {})
	go func() {
		items, err := client.listMemos(context.Background())
		window.Update(func() {
			a.loading = false
			if err != nil {
				a.loadErr = err.Error()
				var authErr *memosAuthError
				var statusErr *memosHTTPError
				if errors.As(err, &authErr) || (errors.As(err, &statusErr) && statusErr.status == http.StatusUnauthorized) {
					a.authenticated = false
					a.client.clearSession()
				}
				return
			}
			a.loadErr = ""
			a.applyServerMemos(items)
		})
	}()
}

func (a *app) signIn() {
	if a.client == nil || a.window == nil || a.loading {
		return
	}
	username := strings.TrimSpace(a.username)
	password := a.password
	a.password = ""
	if username == "" || password == "" {
		a.loadErr = "Enter both your Memos username and password."
		return
	}
	a.loading = true
	a.loadErr = ""
	window := a.window
	client := a.client
	window.Update(func() {})
	go func() {
		displayName, err := client.signIn(context.Background(), username, password)
		signedIn := err == nil
		var items []memo
		if err == nil {
			items, err = client.listMemos(context.Background())
		}
		window.Update(func() {
			a.loading = false
			a.authenticated = signedIn
			if err != nil {
				a.loadErr = err.Error()
				var authErr *memosAuthError
				var statusErr *memosHTTPError
				if errors.As(err, &authErr) || (errors.As(err, &statusErr) && statusErr.status == http.StatusUnauthorized) {
					a.authenticated = false
					client.clearSession()
				}
				return
			}
			a.authenticated = true
			a.username = displayName
			a.loadErr = ""
			a.applyServerMemos(items)
		})
	}()
}

func (a *app) saveMemo() {
	if !a.composerOpen || !a.authenticated || a.client == nil || a.loading || a.saving {
		return
	}
	a.commitBlockEdit()
	content := a.composerContent
	if strings.TrimSpace(content) == "" {
		a.saveErr = "Memo content cannot be empty. Add text before you save."
		return
	}
	if a.window == nil {
		return
	}
	editingID := a.editingMemoID
	a.saving = true
	a.saveErr = ""
	a.loadErr = ""
	window := a.window
	client := a.client
	window.Update(func() {})
	go func() {
		var saved memo
		var err error
		if editingID == "" {
			saved, err = client.createMemo(context.Background(), content)
		} else {
			saved, err = client.updateMemo(context.Background(), editingID, content)
		}
		var items []memo
		if err == nil {
			items, err = client.listMemos(context.Background())
		}
		window.Update(func() {
			a.saving = false
			if err != nil {
				var authErr *memosAuthError
				var statusErr *memosHTTPError
				if errors.As(err, &authErr) || (errors.As(err, &statusErr) && statusErr.status == http.StatusUnauthorized) {
					a.authenticated = false
					a.client.clearSession()
					a.loadErr = err.Error()
				}
				if saved.ID != "" {
					a.upsertMemo(saved, editingID == "")
					a.openMemo(saved.ID, openMain)
					a.composerOpen = false
					a.editingMemoID = ""
					a.composerContent = ""
					a.editingBlock = -1
					a.editingDraft = ""
					a.loadErr = "Memo saved, but the list could not refresh. Select Retry to sync."
					a.saveErr = ""
					return
				}
				a.saveErr = err.Error()
				return
			}
			a.applyServerMemos(items)
			if _, ok := a.memoByID(saved.ID); !ok {
				a.upsertMemo(saved, editingID == "")
			}
			a.openMemo(saved.ID, openMain)
			a.composerOpen = false
			a.editingMemoID = ""
			a.composerContent = ""
			a.editingBlock = -1
			a.editingDraft = ""
			a.saveErr = ""
			a.loadErr = ""
		})
	}()
}

func (a *app) upsertMemo(item memo, first bool) {
	for i := range a.memos {
		if a.memos[i].ID == item.ID {
			if first && i != 0 {
				updated := make([]memo, 0, len(a.memos))
				updated = append(updated, item)
				for index, existing := range a.memos {
					if index != i {
						updated = append(updated, existing)
					}
				}
				a.memos = updated
			} else {
				a.memos[i] = item
			}
			return
		}
	}
	if first {
		a.memos = append([]memo{item}, a.memos...)
		return
	}
	a.memos = append(a.memos, item)
}

func (a *app) applyServerMemos(items []memo) {
	activeID := ""
	if active, ok := a.activeMemo(); ok {
		activeID = active.ID
	}
	a.memos = items
	if _, ok := a.memoByID(activeID); ok {
		return
	}
	if len(items) == 0 {
		a.tabs[0].MemoID = ""
	} else {
		a.tabs[0].MemoID = items[0].ID
	}
	a.activeTabID = "main"
	a.activeTabIndex = 0
}
