package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestScratchFreshLogin(t *testing.T) {
	a := newServerApp("https://memos.example.test", "")
	a.client = &memosClient{}
	a.authenticated = true
	a.applyServerMemos([]memo{
		{ID: "qa", Title: "OpenCode QA — live CRUD", Body: "OpenCode QA — live CRUD 20261113\n\nVersion 3: second edit.", Updated: "Nov 13"},
		{ID: "local", Title: "test memo from local", Body: "test memo from local", Updated: "Nov 13"},
	})
	t.Logf("tabs=%+v active=%q", a.tabs, a.activeTabID)
	active, ok := a.activeMemo()
	t.Logf("activeMemo ok=%v id=%q", ok, active.ID)
	tester := ui.NewTester(a.view, 1280, 900)
	texts := tester.Texts()
	t.Logf("texts=%q", texts)
	if dir := os.Getenv("MEMOS_PREVIEW_DIR"); dir != "" {
		file, err := os.Create(filepath.Join(dir, "freshlogin-1280.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, tester.Image()); err != nil {
			file.Close()
			t.Fatal(err)
		}
		file.Close()
	}
}
