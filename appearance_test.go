package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestShadcnLayoutGates(t *testing.T) {
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("width-%d/dark-%t", width, dark), func(t *testing.T) {
				a := newApp()
				tester := ui.NewTester(a.view, width, 900)
				tester.SetDark(dark)
				assertNativeBounds(t, tester, width, "Search memos")
				if width >= 992 {
					for _, label := range []string{"Library and memos", "Memo workspace", "All memos", "Open options", "Show memos", "Show tags"} {
						assertNativeBounds(t, tester, width, label)
					}
					if err := tester.Click("Show tags"); err != nil {
						t.Fatal(err)
					}
					for _, label := range []string{"Library", "Tags", "All memos"} {
						if !tester.HasText(label) {
							t.Fatalf("tags tab content %q missing: %q", label, tester.Texts())
						}
					}
					if err := tester.Click("Show memos"); err != nil {
						t.Fatal(err)
					}
					if tabStripHeight(appTokens(dark)) != 28 {
						t.Errorf("tab strip height = %.1f, want 28", tabStripHeight(appTokens(dark)))
					}
					bounds, ok := tester.Find("Main window")
					if !ok {
						t.Fatal("tab label is missing")
					}
					if bounds.H > 32 {
						t.Errorf("tab label is too tall: %+v", bounds)
					}
				} else {
					assertNativeBounds(t, tester, width, "Filter memos by tag")
				}
				if err := tester.Click("Search memos"); err != nil {
					t.Fatal(err)
				}
				tester.Type("no-matching-note-qa")
				if !tester.HasText("No sample memos match.") {
					t.Fatal("search empty state has no next action")
				}
			})
		}
	}
}

func TestThemeChooserSwitchesModesAndPersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MEMOS_CONFIG_DIR", dir)
	for _, width := range []int{390, 1280} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			tester := ui.NewTester(a.view, width, 900)
			assertNativeBounds(t, tester, width, "Theme")
			if a.themeMode != themeModeSystem {
				t.Fatalf("default theme mode = %d, want system", a.themeMode)
			}
			choose := func(item string, want int) {
				t.Helper()
				if err := tester.Click("Theme"); err != nil {
					t.Fatalf("open theme menu: %v", err)
				}
				if err := tester.ChooseMenuItem(item); err != nil {
					t.Fatalf("choose %s: %v", item, err)
				}
				if a.themeMode != want {
					t.Fatalf("theme mode = %d, want %d", a.themeMode, want)
				}
			}
			choose("Dark", themeModeDark)
			data, err := os.ReadFile(filepath.Join(dir, "theme"))
			if err != nil {
				t.Fatalf("theme choice was not saved: %v", err)
			}
			if strings.TrimSpace(string(data)) != "dark" {
				t.Fatalf("saved theme = %q, want dark", data)
			}
			choose("Light", themeModeLight)
			choose("System", themeModeSystem)
		})
	}
}

func TestForcedThemeOverridesSystemAppearance(t *testing.T) {
	a := newApp()
	a.themeMode = themeModeLight
	lightTester := ui.NewTester(a.view, 1280, 900)
	lightTester.SetDark(true)
	if r, _, _, _ := lightTester.Image().At(8, 8).RGBA(); r < 0x8000 {
		t.Fatalf("forced light rendered dark: %#04x", r)
	}
	b := newApp()
	b.themeMode = themeModeDark
	darkTester := ui.NewTester(b.view, 1280, 900)
	if r, _, _, _ := darkTester.Image().At(8, 8).RGBA(); r >= 0x8000 {
		t.Fatalf("forced dark rendered light: %#04x", r)
	}
}

func TestShadcnEditorControlsAtAllWidths(t *testing.T) {
	for _, width := range []int{320, 375, 390, 600, 768, 992, 1024, 1280, 1440, 2048} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.liveData, a.authenticated = true, true
			a.client = &memosClient{}
			a.newMemo()
			tester := ui.NewTester(a.view, width, 900)
			for _, label := range []string{"Editor mode", "Live", "Add paragraph", "Cancel", "Save memo"} {
				assertNativeBounds(t, tester, width, label)
			}
			if err := tester.Click("Source"); err != nil {
				t.Fatal(err)
			}
			assertNativeBounds(t, tester, width, "Memo content")
			for _, label := range []string{"Cancel", "Save memo"} {
				bounds, _ := tester.Find(label)
				min := float32(36)
				if width < 992 {
					min = 44
				}
				if bounds.H < min || bounds.W < 24 {
					t.Errorf("%s target is too small: %+v", label, bounds)
				}
			}
		})
	}
}

func TestShadcnListHandlesLongAndMissingFields(t *testing.T) {
	for _, width := range []int{320, 390, 992, 1280, 2048} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newApp()
			a.memos = []memo{{ID: "edge", Title: strings.Repeat("Long unbroken title", 40), Body: strings.Repeat("body", 200), Tags: []string{strings.Repeat("longtag", 70)}, Updated: "November 13, 2026"}}
			a.tabs[0].MemoID = "edge"
			tester := ui.NewTester(a.view, width, 900)
			assertNativeBounds(t, tester, width, "Search memos")
			assertNativeBounds(t, tester, width, "Open actions for "+a.memos[0].Title)
			a.memos[0] = memo{ID: "edge"}
			tester.Frame()
			assertNativeBounds(t, tester, width, "Open actions for ")
		})
	}
}

func TestShadcnLoadingHasTimeoutGuidanceAndAnnouncements(t *testing.T) {
	a := newServerApp("https://memos.example.test", "")
	a.authenticated, a.loading = true, true
	tester := ui.NewTester(a.view, 390, 900)
	tester.Advance(16 * time.Second)
	if !tester.HasText("This is taking longer than expected.") {
		t.Fatal("slow loading state has no connection guidance")
	}
	if len(tester.Announcements()) == 0 {
		t.Fatal("loading state is not announced")
	}
}

func assertNativeBounds(t *testing.T, tester *ui.Tester, width int, label string) {
	t.Helper()
	bounds, ok := tester.Find(label)
	if !ok {
		t.Fatalf("labelled surface %q is missing", label)
	}
	if bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > float32(width)+1 || bounds.Y+bounds.H > 901 {
		t.Errorf("%q exceeds %dx900 viewport: %+v", label, width, bounds)
	}
}
