package main

import "github.com/egoist/mygo/ui"

// A small, consistent set of 24-unit outline icons; no font glyph dependencies.
var memoIcons = map[string]*ui.SVG{
	"notebook": memoIcon(`<rect x="4" y="3" width="16" height="18" rx="2"/><path d="M8 3v18M2 7h4M2 12h4M2 17h4M12 8h4M12 12h4"/>`),
	"notes":    memoIcon(`<rect x="4" y="3" width="16" height="18" rx="2"/><path d="M8 8h8M8 12h8M8 16h5"/>`),
	"tag":      memoIcon(`<path d="M20 13l-7 7a2 2 0 0 1-3 0l-7-7V3h10l7 7a2 2 0 0 1 0 3Z"/><circle cx="7.5" cy="7.5" r="1"/>`),
	"more":     memoIcon(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`),
	"pencil":   memoIcon(`<path d="M17 3a2.8 2.8 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>`),
	"plus":     memoIcon(`<path d="M12 5v14M5 12h14"/>`),
	"check":    memoIcon(`<path d="M20 6 9 17l-5-5"/>`),
	"sun":      memoIcon(`<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>`),
	"moon":     memoIcon(`<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z"/>`),
	"monitor":  memoIcon(`<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>`),
}

func memoIcon(paths string) *ui.SVG {
	icon, err := ui.ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="black" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
	if err != nil {
		panic(err)
	}
	return icon
}
