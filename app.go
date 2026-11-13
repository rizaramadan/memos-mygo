package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

type memo struct {
	ID      string
	Title   string
	Body    string
	Tags    []string
	Updated string
}

type readerTab struct {
	ID     string
	MemoID string
}

type openMode uint8

const (
	openMain openMode = iota
	openForeground
	openBackground
)

const (
	composerModeLive = iota
	composerModeSource
)

const (
	themeModeSystem = iota
	themeModeLight
	themeModeDark
)

const (
	libraryTabMemos = iota
	libraryTabTags
)

type app struct {
	memos                []memo
	tabs                 []readerTab
	activeTabID          string
	activeTabIndex       int
	nextTabID            uint64
	search               string
	selectedTag          string
	libraryTab           int
	compactReader        bool
	liveData             bool
	authenticated        bool
	username             string
	password             string
	loading              bool
	loadingSince         time.Time
	announcedStatus      string
	saving               bool
	composerOpen         bool
	composerReturnToList bool
	composerMode         int
	themeMode            int
	editingMemoID        string
	composerContent      string
	editingBlock         int
	editingDraft         string
	loadErr              string
	saveErr              string
	client               *memosClient
	window               *mygo.Window
}

func (a *app) newMemo() {
	if !a.liveData || !a.authenticated || a.client == nil || a.loading || a.saving {
		return
	}
	a.composerReturnToList = !a.compactReader
	a.composerOpen = true
	a.composerMode = composerModeLive
	a.editingMemoID = ""
	a.composerContent = ""
	a.editingBlock = -1
	a.editingDraft = ""
	a.saveErr = ""
	a.compactReader = true
}

func (a *app) editMemo(memoID string) {
	if !a.liveData || !a.authenticated || a.client == nil || a.loading || a.saving {
		return
	}
	item, ok := a.memoByID(memoID)
	if !ok {
		return
	}
	a.composerReturnToList = !a.compactReader
	a.composerOpen = true
	a.composerMode = composerModeLive
	a.editingMemoID = item.ID
	a.composerContent = item.Body
	a.editingBlock = -1
	a.editingDraft = ""
	a.saveErr = ""
	a.compactReader = true
}

func (a *app) cancelMemoEdit() {
	if a.saving {
		return
	}
	a.composerOpen = false
	a.composerMode = composerModeLive
	a.editingMemoID = ""
	a.composerContent = ""
	a.editingBlock = -1
	a.editingDraft = ""
	a.saveErr = ""
	a.compactReader = !a.composerReturnToList
}

func (a *app) composerBlocks() []string {
	if a.composerContent == "" {
		return []string{""}
	}
	return strings.Split(a.composerContent, "\n\n")
}

func (a *app) startBlockEdit(index int) {
	blocks := a.composerBlocks()
	if index < 0 || index >= len(blocks) {
		return
	}
	if a.editingBlock >= 0 && a.editingBlock < len(blocks) && a.editingBlock != index {
		blocks[a.editingBlock] = a.editingDraft
		a.composerContent = strings.Join(blocks, "\n\n")
		blocks = a.composerBlocks()
	}
	a.editingBlock = index
	a.editingDraft = blocks[index]
}

func (a *app) commitBlockEdit() {
	if a.editingBlock < 0 {
		return
	}
	blocks := a.composerBlocks()
	if a.editingBlock >= len(blocks) {
		a.editingBlock = -1
		a.editingDraft = ""
		return
	}
	blocks[a.editingBlock] = a.editingDraft
	a.composerContent = strings.Join(blocks, "\n\n")
	a.editingBlock = -1
	a.editingDraft = ""
}

func (a *app) cancelBlockEdit() {
	a.editingBlock = -1
	a.editingDraft = ""
}

func newApp() *app {
	memos := []memo{
		{
			ID:      "decision-v1",
			Title:   "Desktop client · first cut",
			Body:    "Keep the first version deliberately small.\n\n## Core loop\n\n- Capture a thought quickly\n- Find it later with search and tags\n- Read and edit without leaving the desktop\n\nRelated: [[Memos API & authentication]] and [[Make capture feel instant]].",
			Tags:    []string{"product", "desktop"},
			Updated: "Nov 13",
		},
		{
			ID:      "api-notes",
			Title:   "Memos API & authentication",
			Body:    "This prototype uses sample content only. It does not connect to a Memos server.\n\nFor a real client, store credentials in the operating system credential store.\n\nRelated: [[Desktop client · first cut]] and [[Read later]].",
			Tags:    []string{"engineering", "memos"},
			Updated: "Nov 12",
		},
		{
			ID:      "capture-flow",
			Title:   "Make capture feel instant",
			Body:    "A memo should take less effort to save than to keep in working memory.\n\nOpen composer → type → save. Keep the title optional, put focus in the body, and make the saved note visible immediately.\n\nSee [[Desktop client · first cut]] for the first-cut decisions.",
			Tags:    []string{"ideas", "product"},
			Updated: "Nov 11",
		},
		{
			ID:      "reading-list",
			Title:   "Read later",
			Body:    "A small collection of links and reading prompts for the next quiet afternoon.\n\nAdd sources only when they are useful; do not turn a reading list into another inbox.\n\nA missing link stays visible: [[Page not written yet]].",
			Tags:    []string{"personal"},
			Updated: "Nov 9",
		},
		{
			ID:      "meeting-recap",
			Title:   "Product check-in · notes",
			Body:    "## Decisions\n\n- Search should work across server results, not just the loaded page.\n- Keep the note list and reader visible together on desktop.\n- Make account status clear without a noisy sync dashboard.\n\nFollow-up: try the prototype at laptop width before adding another panel. See [[Memos API & authentication]].",
			Tags:    []string{"work", "product"},
			Updated: "Nov 8",
		},
		{
			ID:      "edge-long",
			Title:   "A deliberately long title to check wrapping when memo names exceed the reader width",
			Body:    strings.Repeat("Memos desktop sample text checks wrapping without a horizontal scroll. ", 12),
			Tags:    []string{"long note", "edge case"},
			Updated: "Nov 7",
		},
	}
	return &app{
		memos:       memos,
		tabs:        []readerTab{{ID: "main", MemoID: memos[0].ID}},
		activeTabID: "main",
	}
}

func newAppFromEnv() *app {
	serverURL := strings.TrimSpace(os.Getenv("MEMOS_SERVER_URL"))
	var a *app
	if serverURL == "" {
		a = newApp()
	} else {
		a = newServerApp(serverURL, strings.TrimSpace(os.Getenv("MEMOS_ACCESS_TOKEN")))
	}
	a.themeMode = loadThemeMode()
	return a
}

func newServerApp(serverURL, accessToken string) *app {
	a := newApp()
	a.liveData = true
	a.memos = nil
	a.tabs = []readerTab{{ID: "main"}}
	a.activeTabID = "main"
	client, err := newMemosClient(serverURL, accessToken)
	if err != nil {
		a.loadErr = err.Error()
		return a
	}
	a.client = client
	a.authenticated = accessToken != ""
	return a
}

func (a *app) memoByID(id string) (memo, bool) {
	for _, item := range a.memos {
		if item.ID == id {
			return item, true
		}
	}
	return memo{}, false
}

func (a *app) memoByTitle(title string) (memo, bool) {
	want := normalizeTitle(title)
	for _, item := range a.memos {
		if normalizeTitle(item.Title) == want {
			return item, true
		}
	}
	return memo{}, false
}

func normalizeTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(title)), " "))
}

func (a *app) activeTab() (readerTab, bool) {
	for _, tab := range a.tabs {
		if tab.ID == a.activeTabID {
			return tab, true
		}
	}
	return readerTab{}, false
}

func (a *app) activeMemo() (memo, bool) {
	tab, ok := a.activeTab()
	if !ok {
		return memo{}, false
	}
	return a.memoByID(tab.MemoID)
}

func (a *app) openMemo(memoID string, mode openMode) bool {
	if _, ok := a.memoByID(memoID); !ok {
		return false
	}
	if len(a.tabs) == 0 || a.tabs[0].ID != "main" {
		return false
	}

	switch mode {
	case openMain:
		a.tabs[0].MemoID = memoID
		a.activeTabID = "main"
		a.activeTabIndex = 0
		a.compactReader = true
	case openForeground:
		a.nextTabID++
		tabID := fmt.Sprintf("reader-%d", a.nextTabID)
		a.tabs = append(a.tabs, readerTab{ID: tabID, MemoID: memoID})
		a.activeTabID = tabID
		a.activeTabIndex = len(a.tabs) - 1
		a.compactReader = true
	case openBackground:
		a.nextTabID++
		tabID := fmt.Sprintf("reader-%d", a.nextTabID)
		a.tabs = append(a.tabs, readerTab{ID: tabID, MemoID: memoID})
	default:
		return false
	}
	return true
}

func (a *app) activateTab(tabID string) bool {
	for i, tab := range a.tabs {
		if tab.ID == tabID {
			a.activeTabID = tabID
			a.activeTabIndex = i
			a.compactReader = true
			return true
		}
	}
	return false
}

func (a *app) closeActiveTab() bool {
	index := -1
	for i, tab := range a.tabs {
		if tab.ID == a.activeTabID {
			index = i
			break
		}
	}
	if index <= 0 {
		return false
	}
	a.tabs = append(a.tabs[:index], a.tabs[index+1:]...)
	fallback := index - 1
	if fallback >= len(a.tabs) {
		fallback = len(a.tabs) - 1
	}
	a.activeTabID = a.tabs[fallback].ID
	a.activeTabIndex = fallback
	return true
}

func (a *app) selectTag(tag string) {
	a.libraryTab = libraryTabMemos
	a.selectedTag = tag
	filtered := a.filteredMemos()
	if len(filtered) > 0 {
		active, ok := a.activeMemo()
		found := false
		if ok {
			for _, item := range filtered {
				if item.ID == active.ID {
					found = true
					break
				}
			}
		}
		if !found {
			a.openMemo(filtered[0].ID, openMain)
		}
	}
	a.compactReader = false
}

func (a *app) filteredMemos() []memo {
	query := strings.ToLower(strings.TrimSpace(a.search))
	if query == "" && a.selectedTag == "" {
		return append([]memo(nil), a.memos...)
	}
	filtered := make([]memo, 0, len(a.memos))
	for _, item := range a.memos {
		if a.selectedTag != "" && !hasMemoTag(item, a.selectedTag) {
			continue
		}
		matchesQuery := query == "" || strings.Contains(strings.ToLower(item.Title), query) || strings.Contains(strings.ToLower(item.Body), query)
		if !matchesQuery {
			for _, tag := range item.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					matchesQuery = true
					break
				}
			}
		}
		if matchesQuery {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func hasMemoTag(item memo, tag string) bool {
	for _, itemTag := range item.Tags {
		if strings.EqualFold(itemTag, tag) {
			return true
		}
	}
	return false
}
