package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWikiLinkMenuOpensInEachMode(t *testing.T) {
	tests := []struct {
		name       string
		choice     string
		activeTab  string
		activeMemo string
		tabCount   int
	}{
		{name: "main", choice: "Open in main window", activeTab: "main", activeMemo: "api-notes", tabCount: 1},
		{name: "foreground", choice: "Open in new tab", activeTab: "reader-1", activeMemo: "api-notes", tabCount: 2},
		{name: "background", choice: "Open in background tab", activeTab: "main", activeMemo: "decision-v1", tabCount: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := newApp()
			tester := ui.NewTester(a.view, 1280, 900)
			if err := tester.Click("Desktop client · first cut"); err != nil {
				t.Fatalf("open initial memo: %v", err)
			}
			if err := tester.Click("Open wiki page: Memos API & authentication"); err != nil {
				t.Fatalf("click wiki link: %v; visible text: %q", err, tester.Texts())
			}
			menu := strings.Join(tester.Menu(), "\n")
			for _, option := range []string{"Open in main window", "Open in new tab", "Open in background tab"} {
				if !strings.Contains(menu, option) {
					t.Fatalf("menu %q does not contain %q", menu, option)
				}
			}
			if err := tester.ChooseMenuItem(test.choice); err != nil {
				t.Fatalf("choose %q: %v", test.choice, err)
			}
			active, ok := a.activeMemo()
			if !ok || active.ID != test.activeMemo {
				t.Fatalf("active memo = %+v, want %q", active, test.activeMemo)
			}
			if a.activeTabID != test.activeTab || len(a.tabs) != test.tabCount {
				t.Fatalf("active tab = %q, tabs = %d; want %q, %d", a.activeTabID, len(a.tabs), test.activeTab, test.tabCount)
			}
		})
	}
}

func TestReaderToolbarMenuIsClickable(t *testing.T) {
	for _, width := range []int{390, 1280} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.openMemo("api-notes", openMain)
			tester := ui.NewTester(a.view, width, 900)
			if err := tester.Click("Open options"); err != nil {
				t.Fatalf("click toolbar action: %v; visible text: %q", err, tester.Texts())
			}
			if !strings.Contains(strings.Join(tester.Menu(), "\n"), "Open in background tab") {
				t.Fatal("toolbar action did not open its menu")
			}
			if err := tester.ChooseMenuItem("Open in background tab"); err != nil {
				t.Fatalf("choose background tab: %v", err)
			}
			if a.activeTabID != "main" || len(a.tabs) != 2 {
				t.Fatalf("toolbar action state = active %q, tabs %d", a.activeTabID, len(a.tabs))
			}
		})
	}
}

func TestMemoRowActionIsClickable(t *testing.T) {
	a := newApp()
	tester := ui.NewTester(a.view, 1280, 900)
	label := "Open actions for Desktop client · first cut"
	if err := tester.Click(label); err != nil {
		t.Fatalf("click memo-row action: %v; visible text: %q", err, tester.Texts())
	}
	if !strings.Contains(strings.Join(tester.Menu(), "\n"), "Open in new tab") {
		t.Fatalf("memo-row action did not open its menu: %q", tester.Menu())
	}
	if err := tester.ChooseMenuItem("Open in new tab"); err != nil {
		t.Fatalf("choose memo-row action: %v", err)
	}
	if len(a.tabs) != 2 || a.activeTabID != "reader-1" {
		t.Fatalf("memo-row action state: active %q, tabs %d", a.activeTabID, len(a.tabs))
	}
}

func TestCloseTabActionIsClickable(t *testing.T) {
	a := newApp()
	a.openMemo("api-notes", openForeground)
	tester := ui.NewTester(a.view, 1280, 900)
	if err := tester.Click("Close current tab"); err != nil {
		t.Fatalf("click close-tab action: %v; visible text: %q", err, tester.Texts())
	}
	if len(a.tabs) != 1 || a.activeTabID != "main" {
		t.Fatalf("close action state = active %q, tabs %d", a.activeTabID, len(a.tabs))
	}
}

func TestTabsSupportKeyboardNavigation(t *testing.T) {
	a := newApp()
	a.openMemo("api-notes", openForeground)
	tester := ui.NewTester(a.view, 1280, 900)
	if err := tester.Click("Main window"); err != nil {
		t.Fatalf("focus main tab: %v", err)
	}
	tester.Key(0, ui.KeyRight)
	if a.activeTabID != "reader-1" {
		t.Fatalf("right arrow selected %q, want reader-1", a.activeTabID)
	}
	tester.Key(0, ui.KeyHome)
	if a.activeTabID != "main" {
		t.Fatalf("Home selected %q, want main", a.activeTabID)
	}
	tester.Key(0, ui.KeyEnd)
	if a.activeTabID != "reader-1" {
		t.Fatalf("End selected %q, want reader-1", a.activeTabID)
	}
}

func TestNativeLayoutAtRequiredWidths(t *testing.T) {
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			tester := ui.NewTester(a.view, width, 900)
			if !tester.HasText("Memos") {
				t.Fatalf("app title missing at %dpx: %q", width, tester.Texts())
			}
			if !tester.HasText("Desktop client · first cut") {
				t.Fatalf("initial memo missing at %dpx", width)
			}
			labels := []string{"Search memos", "All memos"}
			for _, label := range labels {
				bounds, ok := tester.Find(label)
				if !ok {
					t.Errorf("%q is missing at %dpx", label, width)
					continue
				}
				if bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > float32(width)+1 {
					t.Errorf("%q bounds exceed %dpx viewport: %+v", label, width, bounds)
				}
			}
			if width >= 992 {
				for _, label := range []string{"Show memos", "Show tags", "Main window", "Open options"} {
					if _, ok := tester.Find(label); !ok {
						t.Errorf("desktop pane content %q is missing at %dpx: %q", label, width, tester.Texts())
					}
				}
				if err := tester.Click("Show tags"); err != nil {
					t.Fatalf("open tags tab at %dpx: %v", width, err)
				}
				for _, label := range []string{"Library", "Tags", "product"} {
					if !tester.HasText(label) {
						t.Errorf("tags tab content %q is missing at %dpx: %q", label, width, tester.Texts())
					}
				}
				if err := tester.Click("Show memos"); err != nil {
					t.Fatalf("return to memos tab at %dpx: %v", width, err)
				}
			} else {
				if !tester.HasText("All memos") || !tester.HasText("Filter by tag") {
					t.Errorf("mobile list or tag filter is missing at %dpx: %q", width, tester.Texts())
				}
			}
			image := tester.Image()
			if image.Bounds().Dx() != width {
				t.Errorf("render width = %d, want %d", image.Bounds().Dx(), width)
			}
			if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-light-%d.png", width)))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(file, image); err != nil {
					file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestMemoTagFilterIsClickableInDesktopAndMobileLayouts(t *testing.T) {
	for _, width := range []int{390, 1280} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			tester := ui.NewTester(a.view, width, 900)
			if width < 992 {
				if err := tester.Click("Filter memos by tag"); err != nil {
					t.Fatalf("open tag filter at %dpx: %v; visible text: %q", width, err, tester.Texts())
				}
				if err := tester.ChooseMenuItem("product"); err != nil {
					t.Fatalf("choose product tag at %dpx: %v; menu: %q", width, err, tester.Menu())
				}
			} else {
				if err := tester.Click("Show tags"); err != nil {
					t.Fatalf("open tags tab at %dpx: %v", width, err)
				}
				if err := tester.Click("product"); err != nil {
					t.Fatalf("click product tag at %dpx: %v; visible text: %q", width, err, tester.Texts())
				}
			}
			if a.selectedTag != "product" {
				t.Fatalf("selected tag = %q, want product", a.selectedTag)
			}
			filtered := a.filteredMemos()
			if len(filtered) != 3 {
				t.Fatalf("product tag shows %d memos, want 3", len(filtered))
			}
			if width >= 992 && a.libraryTab != libraryTabMemos {
				t.Fatalf("selecting a tag did not return to memos tab, libraryTab=%d", a.libraryTab)
			}
		})
	}
}

func TestLibraryIconTabsSwitchPanesWithoutLosingState(t *testing.T) {
	a := newApp()
	tester := ui.NewTester(a.view, 1280, 900)
	for _, label := range []string{"Show memos", "Show tags"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("library tab %q is missing: %q", label, tester.Texts())
		}
	}
	if err := tester.Click("Show tags"); err != nil {
		t.Fatalf("open tags tab: %v", err)
	}
	if a.libraryTab != libraryTabTags {
		t.Fatalf("libraryTab=%d, want tags", a.libraryTab)
	}
	if !tester.HasText("product") {
		t.Fatalf("tags tab should show tags: %q", tester.Texts())
	}
	if _, ok := tester.Find("Open memo: Desktop client · first cut"); ok {
		t.Fatalf("tags tab should not show memo rows: %q", tester.Texts())
	}
	a.search = "afternoon"
	tester.Frame()
	if err := tester.Click("Show memos"); err != nil {
		t.Fatalf("return to memos tab: %v", err)
	}
	if a.libraryTab != libraryTabMemos || a.search != "afternoon" {
		t.Fatalf("memos tab lost state: libraryTab=%d search=%q", a.libraryTab, a.search)
	}
	if _, ok := tester.Find("Open memo: Read later"); !ok {
		t.Fatalf("memos tab should show filtered row: %q", tester.Texts())
	}
	if _, ok := tester.Find("Open memo: Memos API & authentication"); ok {
		t.Fatalf("memos tab should not show unfiltered rows: %q", tester.Texts())
	}
}

func TestReaderLayoutAtRequiredWidths(t *testing.T) {
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.openMemo("api-notes", openForeground)
			tester := ui.NewTester(a.view, width, 900)
			if !tester.HasText("Memos API & authentication") {
				t.Fatalf("reader memo missing at %dpx: %q", width, tester.Texts())
			}
			labels := []string{"Main window", "Open options", "All memos"}
			for _, label := range labels {
				bounds, ok := tester.Find(label)
				if !ok {
					t.Errorf("%q is missing at %dpx", label, width)
					continue
				}
				if bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > float32(width)+1 {
					t.Errorf("%q bounds exceed %dpx viewport: %+v", label, width, bounds)
				}
			}
			if image := tester.Image(); image.Bounds().Dx() != width {
				t.Errorf("render width = %d, want %d", image.Bounds().Dx(), width)
			}
			if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-reader-%d.png", width)))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(file, tester.Image()); err != nil {
					file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestMyGoTesterSearchOpensFilteredMemoAndReturnsToList(t *testing.T) {
	a := newApp()
	tester := ui.NewTester(a.view, 390, 900)
	if err := tester.Click("Search memos"); err != nil {
		t.Fatalf("focus search: %v", err)
	}
	tester.Type("afternoon")
	if a.search != "afternoon" {
		t.Fatalf("search input = %q, want afternoon", a.search)
	}
	filtered := a.filteredMemos()
	if len(filtered) != 1 || filtered[0].ID != "reading-list" {
		t.Fatalf("search results = %+v, want only reading-list", filtered)
	}
	if !tester.HasText("Read later") || tester.HasText("Memos API & authentication") {
		t.Fatalf("search view does not show only the matching note: %q", tester.Texts())
	}
	if err := tester.Click("Read later"); err != nil {
		t.Fatalf("open filtered note: %v", err)
	}
	active, ok := a.activeMemo()
	if !ok || active.ID != "reading-list" || !a.compactReader {
		t.Fatalf("opened note = %+v, compact reader = %t", active, a.compactReader)
	}
	if !tester.HasText("A small collection of links and reading prompts") {
		t.Fatalf("reader body did not render: %q", tester.Texts())
	}
	if err := tester.Click("All memos"); err != nil {
		t.Fatalf("return to filtered list: %v", err)
	}
	if a.compactReader || !tester.HasText("Read later") {
		t.Fatalf("back action did not return to the filtered list: %q", tester.Texts())
	}
}

func TestNativeViewDarkTheme(t *testing.T) {
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			tester := ui.NewTester(a.view, width, 900)
			tester.SetDark(true)
			if !tester.HasText("Desktop client · first cut") {
				t.Fatalf("dark theme render missing memo at %dpx: %q", width, tester.Texts())
			}
			if image := tester.Image(); image.Bounds().Dx() != width {
				t.Fatalf("dark render width = %d, want %d", image.Bounds().Dx(), width)
			}
			if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-dark-%d.png", width)))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(file, tester.Image()); err != nil {
					file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLiveDataStatesAtRequiredWidths(t *testing.T) {
	states := []struct {
		name    string
		prepare func() *app
		want    string
	}{
		{
			name: "new-memo",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.newMemo()
				return a
			},
			want: "New memo",
		},
		{
			name: "edit-memo",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.editMemo("decision-v1")
				return a
			},
			want: "Edit memo",
		},
		{
			name: "dirty-memo",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.newMemo()
				a.composerContent = "Unsaved local draft"
				return a
			},
			want: "New memo",
		},
		{
			name: "pending-memo",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.newMemo()
				a.saving = true
				return a
			},
			want: "Saving…",
		},
		{
			name: "error-memo",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.newMemo()
				a.saveErr = "Could not save memo. Check the connection and try again."
				return a
			},
			want: "Could not save memo. Check the connection and try again.",
		},
		{
			name: "sign-in",
			prepare: func() *app {
				return newServerApp("https://memos.example.test", "")
			},
			want: "Connect to Memos",
		},
		{
			name: "loading",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.memos = nil
				a.tabs[0].MemoID = ""
				a.loading = true
				return a
			},
			want: "Loading memos from your server…",
		},
		{
			name: "error",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.memos = nil
				a.tabs[0].MemoID = ""
				a.loadErr = "Memos rejected the credentials. Sign in again or check the access token."
				return a
			},
			want: "Memos rejected the credentials",
		},
		{
			name: "empty",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.memos = nil
				a.tabs[0].MemoID = ""
				return a
			},
			want: "No memos are available on this server.",
		},
		{
			name: "populated",
			prepare: func() *app {
				a := newApp()
				a.liveData = true
				a.authenticated = true
				return a
			},
			want: "Live · 6 memos",
		},
	}
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		for _, state := range states {
			t.Run(fmt.Sprintf("%s/%d", state.name, width), func(t *testing.T) {
				a := state.prepare()
				tester := ui.NewTester(a.view, width, 900)
				if !tester.HasText(state.want) {
					t.Fatalf("state text %q missing at %dpx: %q", state.want, width, tester.Texts())
				}
				if tester.Image().Bounds().Dx() != width {
					t.Fatalf("render width = %d, want %d", tester.Image().Bounds().Dx(), width)
				}
				if state.name == "sign-in" {
					for _, label := range []string{"Username", "Password", "Sign in"} {
						if !tester.HasText(label) {
							t.Errorf("sign-in control %q missing at %dpx", label, width)
						}
					}
					if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
						if err := os.MkdirAll(dir, 0o755); err != nil {
							t.Fatal(err)
						}
						file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-login-%d.png", width)))
						if err != nil {
							t.Fatal(err)
						}
						if err := png.Encode(file, tester.Image()); err != nil {
							file.Close()
							t.Fatal(err)
						}
						if err := file.Close(); err != nil {
							t.Fatal(err)
						}
					}
				}
				if strings.HasSuffix(state.name, "-memo") {
					saveLabel := "Save memo"
					if state.name == "pending-memo" {
						saveLabel = "Saving…"
					}
					for _, label := range []string{"Live", "Cancel", saveLabel} {
						if !tester.HasText(label) {
							t.Errorf("editor control %q missing at %dpx", label, width)
						}
						if bounds, ok := tester.Find(label); ok && (bounds.X < 0 || bounds.X+bounds.W > float32(width)+1) {
							t.Errorf("editor control %q exceeds %dpx viewport: %+v", label, width, bounds)
						}
					}
					if state.name != "pending-memo" {
						if err := tester.Click("Source"); err != nil {
							t.Errorf("switch to source at %dpx: %v", width, err)
						} else if !tester.HasText("Memo content") {
							t.Errorf("source textarea missing at %dpx", width)
						}
					}
				}
				if state.name == "dirty-memo" && a.composerContent != "Unsaved local draft" {
					t.Fatalf("dirty composer content changed to %q", a.composerContent)
				}
				if state.name == "new-memo" || state.name == "edit-memo" || ((state.name == "dirty-memo" || state.name == "pending-memo" || state.name == "error-memo") && (width == 390 || width == 1280)) {
					if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
						if err := os.MkdirAll(dir, 0o755); err != nil {
							t.Fatal(err)
						}
						file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-%s-%d.png", state.name, width)))
						if err != nil {
							t.Fatal(err)
						}
						if err := png.Encode(file, tester.Image()); err != nil {
							file.Close()
							t.Fatal(err)
						}
						if err := file.Close(); err != nil {
							t.Fatal(err)
						}
					}
				}
			})
		}
	}
}

func TestLiveCreateAndEditActionsOpenEditorWithoutSaving(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	tester := ui.NewTester(a.view, 1280, 900)

	if err := tester.Click("New memo"); err != nil {
		t.Fatalf("open new memo editor: %v; visible text: %q", err, tester.Texts())
	}
	if !a.composerOpen || a.composerContent != "" || !tester.HasText("Save memo") {
		t.Fatal("new memo editor did not open in an untouched state")
	}
	if err := tester.Click("Cancel"); err != nil {
		t.Fatalf("cancel new memo: %v", err)
	}
	if a.composerOpen {
		t.Fatal("cancel left the new memo editor open")
	}
	if err := tester.Click("Desktop client · first cut"); err != nil {
		t.Fatalf("open memo reader: %v", err)
	}
	if err := tester.Click("Edit"); err != nil {
		t.Fatalf("open edit memo editor: %v; visible text: %q", err, tester.Texts())
	}
	if !a.composerOpen || a.editingMemoID != "decision-v1" || a.composerContent != a.memos[0].Body {
		t.Fatal("edit action did not load the selected memo into the editor")
	}
	if err := tester.Click("Cancel"); err != nil {
		t.Fatalf("cancel edit: %v", err)
	}
	if a.memos[0].Body != "Keep the first version deliberately small.\n\n## Core loop\n\n- Capture a thought quickly\n- Find it later with search and tags\n- Read and edit without leaving the desktop\n\nRelated: [[Memos API & authentication]] and [[Make capture feel instant]]." {
		t.Fatal("cancel changed the saved memo content")
	}
}

func TestMemoEditorModesKeepDraftAndRenderPreviewAtRequiredWidths(t *testing.T) {
	widths := []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048}
	base := "## Preview heading\n\nPreview paragraph.\n\n- First list item\n- Second list item"
	for _, width := range widths {
		for _, dark := range []bool{false, true} {
			theme := "light"
			if dark {
				theme = "dark"
			}
			t.Run(fmt.Sprintf("%s/%d", theme, width), func(t *testing.T) {
				draft := base
				a := newApp()
				a.liveData = true
				a.authenticated = true
				a.client = &memosClient{}
				a.newMemo()
				a.composerContent = draft
				tester := ui.NewTester(a.view, width, 900)
				if dark {
					tester.SetDark(true)
				}
				if a.composerMode != composerModeLive {
					t.Fatalf("new memo editor mode = %d, want live", a.composerMode)
				}
				for _, text := range []string{"Preview heading", "Preview paragraph.", "First list item", "Edit block 1"} {
					if !tester.HasText(text) {
						t.Errorf("live text %q missing at %dpx: %q", text, width, tester.Texts())
					}
				}
				if err := tester.Click("Edit block 2"); err != nil {
					t.Fatalf("edit block at %dpx: %v; visible text: %q", width, err, tester.Texts())
				}
				if a.editingBlock != 1 || a.editingDraft != "Preview paragraph." {
					t.Fatalf("block edit state = %d %q, want 1 paragraph", a.editingBlock, a.editingDraft)
				}
				tester.Type(" + typed")
				if a.editingDraft != "Preview paragraph. + typed" {
					t.Fatalf("typing did not go to the block draft: %q", a.editingDraft)
				}
				if a.composerContent != draft {
					t.Fatalf("typing leaked into the saved draft: %q", a.composerContent)
				}
				draft = "## Preview heading\n\nPreview paragraph. + typed\n\n- First list item\n- Second list item"
				if err := tester.Click("Edit block 3"); err != nil {
					t.Fatalf("move to another block at %dpx: %v", width, err)
				}
				if a.editingBlock != 2 || a.composerContent != draft {
					t.Fatalf("moving blocks did not commit cleanly: editing=%d content=%q", a.editingBlock, a.composerContent)
				}
				if err := tester.Click("Edit block 2"); err != nil {
					t.Fatalf("return to block at %dpx: %v", width, err)
				}
				a.cancelBlockEdit()
				tester.Frame()
				if err := tester.Click("Source"); err != nil {
					t.Fatalf("switch to source at %dpx: %v; visible text: %q", width, err, tester.Texts())
				}
				if a.composerMode != composerModeSource {
					t.Fatalf("editor mode = %d, want source", a.composerMode)
				}
				if !tester.HasText("Memo content") {
					t.Errorf("source textarea missing at %dpx", width)
				}
				if a.composerContent != draft {
					t.Fatalf("switching to source changed the draft")
				}
				if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0o755); err != nil {
						t.Fatal(err)
					}
					file, err := os.Create(filepath.Join(dir, fmt.Sprintf("memos-preview-%s-%d.png", theme, width)))
					if err != nil {
						t.Fatal(err)
					}
					if err := png.Encode(file, tester.Image()); err != nil {
						file.Close()
						t.Fatal(err)
					}
					if err := file.Close(); err != nil {
						t.Fatal(err)
					}
				}
				if err := tester.Click("Live"); err != nil {
					t.Fatalf("switch back to live at %dpx: %v", width, err)
				}
				if a.composerMode != composerModeLive {
					t.Fatalf("editor did not return to live mode at %dpx", width)
				}
				for _, text := range []string{"Preview heading", "Edit block 3"} {
					if !tester.HasText(text) {
						t.Errorf("live text %q missing after return at %dpx", text, width)
					}
				}
				if a.composerContent != draft {
					t.Fatalf("switching back to live changed the draft")
				}
			})
		}
	}
}

func TestWebLinkOpensInBrowser(t *testing.T) {
	a := newApp()
	a.memos = append([]memo{{
		ID:      "web-link-note",
		Title:   "Web link note",
		Body:    "Read [the docs](https://example.com/docs) for details.",
		Tags:    nil,
		Updated: "Nov 14",
	}}, a.memos...)
	a.openMemo("web-link-note", openMain)
	tester := ui.NewTester(a.view, 1280, 900)
	if err := tester.Click("Open web link: https://example.com/docs"); err != nil {
		t.Fatalf("click web link: %v; visible text: %q", err, tester.Texts())
	}
	for _, opened := range tester.OpenedURLs() {
		if opened == "https://example.com/docs" {
			return
		}
	}
	t.Fatalf("web link did not open browser: %q", tester.OpenedURLs())
}

func TestDesktopReaderAutoOpensFirstMemoWhenNoneOpen(t *testing.T) {
	a := newServerApp("https://memos.example.test", "")
	a.client = &memosClient{}
	a.authenticated = true
	a.applyServerMemos([]memo{
		{ID: "first", Title: "First", Body: "First body.", Updated: "Nov 13"},
		{ID: "second", Title: "Second", Body: "Second body.", Updated: "Nov 12"},
	})
	a.tabs[0].MemoID = ""
	a.activeTabID = "main"
	tester := ui.NewTester(a.view, 1280, 900)
	active, ok := a.activeMemo()
	if !ok || active.ID != "first" {
		t.Fatalf("reader did not auto-open first memo: %+v, %t", active, ok)
	}
	if !tester.HasText("First body.") {
		t.Fatalf("auto-opened memo body missing: %q", tester.Texts())
	}
}

func TestAliasedWikiLinkShowsDisplayAndOpensTarget(t *testing.T) {
	a := newApp()
	a.memos = append([]memo{{
		ID:      "alias-note",
		Title:   "Alias note",
		Body:    "Open [[Memos API & authentication|the API docs]] now.",
		Updated: "Nov 14",
	}}, a.memos...)
	a.openMemo("alias-note", openMain)
	tester := ui.NewTester(a.view, 1280, 900)
	if !tester.HasText("the API docs") {
		t.Fatalf("alias display text missing: %q", tester.Texts())
	}
	if err := tester.Click("Open wiki page: the API docs"); err != nil {
		t.Fatalf("click alias link: %v; visible text: %q", err, tester.Texts())
	}
	if err := tester.ChooseMenuItem("Open in main window"); err != nil {
		t.Fatalf("open alias target: %v", err)
	}
	active, ok := a.activeMemo()
	if !ok || active.ID != "api-notes" {
		t.Fatalf("alias target = %+v", active)
	}
}

func TestLinkAutocompleteInsertsMemoLink(t *testing.T) {
	for _, width := range []int{390, 1280} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.liveData = true
			a.authenticated = true
			a.client = &memosClient{}
			a.newMemo()
			tester := ui.NewTester(a.view, width, 900)
			a.editingDraft = "See [[api"
			tester.Frame()
			if _, ok := tester.Find("Insert link to Memos API & authentication"); !ok {
				t.Fatalf("completion suggestion missing: %q", tester.Texts())
			}
			if err := tester.Click("Insert link to Memos API & authentication"); err != nil {
				t.Fatalf("accept suggestion: %v", err)
			}
			if a.editingDraft != "See [[Memos API & authentication]]" {
				t.Fatalf("draft after accept = %q", a.editingDraft)
			}
		})
	}
}

func TestLinkCompletionSidebarPlacement(t *testing.T) {
	for _, width := range []int{390, 1280, 1440} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.liveData = true
			a.authenticated = true
			a.client = &memosClient{}
			a.newMemo()
			tester := ui.NewTester(a.view, width, 900)
			a.editingDraft = "See [[api"
			tester.Frame()
			if _, ok := tester.Find("Insert link to Memos API & authentication"); !ok {
				t.Fatalf("completion suggestion missing at %dpx: %q", width, tester.Texts())
			}
			_, sidebar := tester.Find("Link suggestions")
			if width >= 992 && !sidebar {
				t.Fatalf("completion should sit in the right sidebar at %dpx", width)
			}
			if width < 992 && sidebar {
				t.Fatalf("completion should stay inline below the field at %dpx", width)
			}
			if width >= 992 {
				bounds, ok := tester.Find("Edit block 1")
				if !ok {
					t.Fatalf("block editor missing at %dpx", width)
				}
				if bounds.Y > 400 {
					t.Fatalf("editor content sinks below the fold at %dpx: %+v", width, bounds)
				}
			}
		})
	}
}

func TestMemoEditorPreviewShowsEmptyState(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	a.newMemo()
	tester := ui.NewTester(a.view, 390, 900)
	if !tester.HasText("Empty note") {
		t.Fatalf("empty live guidance missing: %q", tester.Texts())
	}
	if _, ok := tester.Find("Edit block 1"); !ok {
		t.Fatalf("empty note has no block editor: %q", tester.Texts())
	}
}

func TestNativeEditorDarkTheme(t *testing.T) {
	a := newApp()
	a.liveData = true
	a.authenticated = true
	a.client = &memosClient{}
	a.newMemo()
	tester := ui.NewTester(a.view, 1280, 900)
	tester.SetDark(true)
	if !tester.HasText("New memo") || !tester.HasText("Save memo") {
		t.Fatalf("dark editor render missing controls: %q", tester.Texts())
	}
	if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(dir, "memos-editor-dark-1280.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, tester.Image()); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoginPasswordFieldDoesNotExposeInputText(t *testing.T) {
	a := newServerApp("https://memos.example.test", "")
	a.password = "unit-test-password"
	tester := ui.NewTester(a.view, 390, 900)
	if tester.HasText(a.password) {
		t.Fatal("password appeared as visible UI text")
	}
	if !tester.HasText("Password") {
		t.Fatal("password field has no accessible label")
	}
}
