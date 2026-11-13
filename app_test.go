package main

import (
	"strings"
	"testing"
)

func TestOpenModes(t *testing.T) {
	a := newApp()
	if !a.openMemo("api-notes", openMain) {
		t.Fatal("main window did not open memo")
	}
	if a.activeTabID != "main" || a.tabs[0].MemoID != "api-notes" {
		t.Fatalf("main state = active %q, memo %q", a.activeTabID, a.tabs[0].MemoID)
	}

	if !a.openMemo("capture-flow", openBackground) {
		t.Fatal("background tab did not open memo")
	}
	if a.activeTabID != "main" || a.tabs[0].MemoID != "api-notes" {
		t.Fatalf("background tab changed current reader: active %q, main memo %q", a.activeTabID, a.tabs[0].MemoID)
	}
	if got := a.tabs[len(a.tabs)-1].MemoID; got != "capture-flow" {
		t.Fatalf("background memo = %q, want capture-flow", got)
	}

	if !a.openMemo("reading-list", openForeground) {
		t.Fatal("foreground tab did not open memo")
	}
	active, ok := a.activeMemo()
	if !ok || active.ID != "reading-list" || a.activeTabID == "main" {
		t.Fatalf("foreground state = active %q, memo %+v", a.activeTabID, active)
	}
}

func TestCloseTabReturnsToPreviousTab(t *testing.T) {
	a := newApp()
	a.openMemo("api-notes", openForeground)
	last := a.activeTabID
	if !a.closeActiveTab() {
		t.Fatal("active tab did not close")
	}
	if a.activeTabID != "main" || len(a.tabs) != 1 {
		t.Fatalf("after closing %q: active %q, tabs %d", last, a.activeTabID, len(a.tabs))
	}
	if a.closeActiveTab() {
		t.Fatal("main tab must stay open")
	}
}

func TestParseWikiLinks(t *testing.T) {
	a := newApp()
	parts := parseWikiLinks("Read [[Memos API & authentication]] and [[Page not written yet]].", a.memos)
	if len(parts) != 5 {
		t.Fatalf("parts = %#v", parts)
	}
	if parts[1].Text != "Memos API & authentication" || parts[1].TargetID != "api-notes" || !parts[1].IsLink {
		t.Fatalf("resolved link = %#v", parts[1])
	}
	if parts[3].Text != "Page not written yet" || parts[3].TargetID != "" || !parts[3].IsLink {
		t.Fatalf("unresolved link = %#v", parts[3])
	}
	if parts[4].Text != "." {
		t.Fatalf("trailing text = %#v", parts[4])
	}
	if got := strings.Join([]string{parts[0].Text, parts[2].Text, parts[4].Text}, ""); got != "Read  and ." {
		t.Fatalf("plain text = %q", got)
	}
}

func TestComposerKeepsEditsLocalUntilSave(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	original := a.memos[0]

	a.newMemo()
	if !a.composerOpen || a.editingMemoID != "" || a.composerContent != "" {
		t.Fatalf("new memo composer state = open %t, id %q, content %q", a.composerOpen, a.editingMemoID, a.composerContent)
	}
	a.composerContent = "Draft that is not saved"
	if got, _ := a.memoByID(original.ID); got.Body != original.Body {
		t.Fatal("draft changed an existing memo before save")
	}
	a.cancelMemoEdit()
	if a.composerOpen || a.composerContent != "" {
		t.Fatalf("cancel left composer open with content %q", a.composerContent)
	}

	a.editMemo(original.ID)
	if !a.composerOpen || a.editingMemoID != original.ID || a.composerContent != original.Body {
		t.Fatalf("edit composer state = open %t, id %q", a.composerOpen, a.editingMemoID)
	}
	a.composerContent = "Updated draft"
	if got, _ := a.memoByID(original.ID); got.Body != original.Body {
		t.Fatal("editing the draft changed the memo before save")
	}
}

func TestMemoTagFiltersCombineWithSearch(t *testing.T) {
	a := newApp()
	a.selectTag("product")
	filtered := a.filteredMemos()
	if len(filtered) != 3 {
		t.Fatalf("product tag returned %d memos, want 3: %+v", len(filtered), filtered)
	}
	for _, item := range filtered {
		if !hasMemoTag(item, "product") {
			t.Errorf("memo %q does not have product tag", item.ID)
		}
	}

	a.search = "follow-up"
	filtered = a.filteredMemos()
	if len(filtered) != 1 || filtered[0].ID != "meeting-recap" {
		t.Fatalf("product + follow-up search returned %+v, want meeting-recap", filtered)
	}

	a.search = "engineering"
	if filtered = a.filteredMemos(); len(filtered) != 0 {
		t.Fatalf("product + engineering search returned %+v, want none", filtered)
	}

	a.search = ""
	a.selectTag("")
	if filtered = a.filteredMemos(); len(filtered) != len(a.memos) {
		t.Fatalf("clearing tag filter left %d memos, want %d", len(filtered), len(a.memos))
	}
}

func TestParseInlineLinks(t *testing.T) {
	a := newApp()
	parts := parseInline("Read [the docs](https://example.com/docs) and [[Memos API & authentication]].", a.memos)
	if len(parts) != 5 {
		t.Fatalf("parts = %#v", parts)
	}
	if parts[1].URL != "https://example.com/docs" || parts[1].Text != "the docs" || !parts[1].IsLink {
		t.Fatalf("web link part = %#v", parts[1])
	}
	if parts[3].TargetID != "api-notes" || !parts[3].IsLink {
		t.Fatalf("wiki link part = %#v", parts[3])
	}
	if got := parts[0].Text + parts[2].Text + parts[4].Text; got != "Read  and ." {
		t.Fatalf("plain text = %q", got)
	}
}

func TestParseWikiLinkAliasShowsDisplayText(t *testing.T) {
	a := newApp()
	parts := parseInline("See [[Memos API & authentication|API docs]] and [[Missing page|Shown text]].", a.memos)
	if len(parts) != 5 {
		t.Fatalf("parts = %#v", parts)
	}
	if parts[1].Text != "API docs" || parts[1].TargetID != "api-notes" || !parts[1].IsLink {
		t.Fatalf("aliased link part = %#v", parts[1])
	}
	if parts[3].Text != "Shown text" || parts[3].TargetID != "" || !parts[3].IsLink {
		t.Fatalf("unresolved alias part = %#v", parts[3])
	}
}

func TestAcceptLinkSuggestionPreservesAlias(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	a.newMemo()
	a.composerContent = "See [[api|my view"
	a.startBlockEdit(0)
	if query, ok := a.linkCompletionQuery(); !ok || query != "api" {
		t.Fatalf("alias query = %q, %t", query, ok)
	}
	a.acceptLinkSuggestion("Memos API & authentication")
	if a.editingDraft != "See [[Memos API & authentication|my view" {
		t.Fatalf("draft after accept = %q", a.editingDraft)
	}
}

func TestLinkCompletionQueryAndAccept(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	a.newMemo()
	a.composerContent = "## H\n\nSee [[api"
	a.startBlockEdit(1)
	query, ok := a.linkCompletionQuery()
	if !ok || query != "api" {
		t.Fatalf("live query = %q, %t", query, ok)
	}
	suggestions := a.linkSuggestions(query)
	if len(suggestions) != 1 || suggestions[0].ID != "api-notes" {
		t.Fatalf("suggestions = %+v", suggestions)
	}
	a.acceptLinkSuggestion(suggestions[0].Title)
	if a.editingDraft != "See [[Memos API & authentication]]" {
		t.Fatalf("live draft = %q", a.editingDraft)
	}
	a.commitBlockEdit()
	if a.composerContent != "## H\n\nSee [[Memos API & authentication]]" {
		t.Fatalf("committed content = %q", a.composerContent)
	}
	a.composerMode = composerModeSource
	a.composerContent = "See [[read"
	if query, ok = a.linkCompletionQuery(); !ok || query != "read" {
		t.Fatalf("source query = %q, %t", query, ok)
	}
	a.acceptLinkSuggestion("Read later")
	if a.composerContent != "See [[Read later]]" {
		t.Fatalf("source content = %q", a.composerContent)
	}
}

func TestSaveMemoRejectsEmptyContent(t *testing.T) {
	a := newServerApp("https://memos.example.test", "")
	a.authenticated = true
	a.composerOpen = true
	a.composerContent = " \n\t "
	a.saveMemo()
	if a.saveErr != "Memo content cannot be empty. Add text before you save." {
		t.Fatalf("save error = %q", a.saveErr)
	}
	if a.saving {
		t.Fatal("empty memo entered pending state")
	}
}
