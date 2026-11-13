package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	dark := c.Theme().Dark
	if a.themeMode == themeModeLight {
		dark = false
	} else if a.themeMode == themeModeDark {
		dark = true
	}
	theme := themeFor(dark)
	c.SetTheme(theme)
	values := appTokens(dark)
	if a.loading {
		if a.loadingSince.IsZero() {
			a.loadingSince = c.Now()
		}
	} else {
		a.loadingSince = time.Time{}
	}
	status := a.connectionStatus()
	if a.saveErr != "" {
		status = a.saveErr
	}
	if status != a.announcedStatus {
		c.Announce(status)
		a.announcedStatus = status
	}
	width, _ := c.Size()
	if width < tokenValue(values, "breakpoint-md", 992) {
		if a.compactReader {
			a.readerView(c, theme, values)
		} else {
			a.mobileNotesView(c, theme, values)
		}
		return
	}
	a.desktopNotesView(c, theme, values)
}

func (a *app) desktopNotesView(c *ui.Context, theme *ui.Theme, values map[string]string) {
	if _, ok := a.activeMemo(); !ok && !a.loading && !a.composerOpen {
		if filtered := a.filteredMemos(); len(filtered) > 0 {
			a.openMemo(filtered[0].ID, openMain)
		}
	}
	ui.Row(c.Key("notes-desktop-shell")).Fill().Background(theme.Background).Children(func() {
		a.unifiedLibraryPane(c, theme, values).Width(tokenValue(values, "library-width", 360)).FillHeight()
		ui.Column(c.Key("notes-reader-pane")).Label("Memo workspace").Grow(1).FillHeight().Padding(tokenValue(values, "space-4", 16)).Gap(0).Background(theme.Background).Children(func() {
			a.tabStrip(c, theme, values, false)
			ui.Divider(c).Background(ui.Hex(values["divider"]))
			if a.liveData && !a.authenticated {
				ui.Column(c.Key("desktop-sign-in-panel")).FillWidth().Grow(1).MaxWidth(tokenValue(values, "login-max-width", 384)).AlignSelf(ui.Center).PaddingY(tokenValue(values, "space-6", 32)).Children(func() {
					a.signInPanel(c, theme, values)
				})
			} else {
				a.reader(c, theme, values)
			}
		})
	})
}

func (a *app) mobileNotesView(c *ui.Context, theme *ui.Theme, values map[string]string) {
	ui.Column(c.Key("notes-mobile-shell")).Fill().Padding(tokenValue(values, "space-3", 12)).Gap(tokenValue(values, "space-2", 8)).Background(theme.Background).Children(func() {
		a.appHeader(c, theme, values)
		ui.Divider(c).Background(ui.Hex(values["divider"]))
		a.memoListPane(c, theme, values, false)
	})
}

func (a *app) notesSidebar(c *ui.Context, theme *ui.Theme, values map[string]string) ui.Element {
	sidebar := ui.Column(c.Key("notes-sidebar")).Label("Memo navigation").FillHeight().Padding(tokenValue(values, "space-4", 16)).Gap(tokenValue(values, "space-4", 16)).Background(ui.Hex(values["sidebar"])).BorderWidth(0, tokenValue(values, "stroke", 1), 0, 0).BorderColor(ui.Hex(values["divider"]))
	sidebar.Children(func() {
		ui.Row(c.Key("sidebar-brand")).Gap(tokenValue(values, "space-2", 8)).MinHeight(controlMinHeight(c, values)).Children(func() {
			ui.Icon(c, memoIcons["notebook"]).Size(tokenValue(values, "step-2", 20), tokenValue(values, "step-2", 20))
			ui.Text(c.Key("sidebar-title"), "Memos").Role(ui.RoleHeading).Level(1).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-2", 20)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
		})
		ui.Text(c.Key("sidebar-library-label"), "Library").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
		allMemos := sidebarButton(c.Key("sidebar-all-memos"), theme, values, "All memos", "notes", len(a.memos), a.selectedTag == "")
		if allMemos.Clicked() {
			a.selectTag("")
		}
		ui.Text(c.Key("sidebar-tags-label"), "Tags").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
		ui.Scroll(c.Key("sidebar-tag-scroll")).Grow(1).Children(func() {
			ui.Column(c.Key("sidebar-tags")).Gap(tokenValue(values, "space-1", 4)).Children(func() {
				for _, tag := range availableMemoTags(a.memos) {
					tag := tag
					button := sidebarButton(c.Key("sidebar-tag-"+tag), theme, values, tag, "tag", memoTagCount(a.memos, tag), a.selectedTag == tag)
					if button.Clicked() {
						a.selectTag(tag)
					}
				}
			})
		})
		ui.Divider(c).Background(ui.Hex(values["divider"]))
		ui.Text(c.Key("sidebar-status"), a.connectionStatus()).FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted).MaxLines(2).Ellipsis("…")
	})
	return sidebar
}

func sidebarButton(c *ui.Context, theme *ui.Theme, values map[string]string, label, icon string, count int, selected bool) ui.Element {
	button := ui.ButtonBase(c).Label(label).FillWidth().MinWidth(0).MinHeight(controlMinHeight(c, values)).Padding(tokenValue(values, "space-2", 8)).Gap(tokenValue(values, "space-2", 8)).Radius(theme.Radius).Selected(selected).TextColor(theme.Text)
	background := ui.Hex(values["sidebar"])
	if selected || button.Hovered() {
		background = theme.Selection
	}
	button.Background(background).Children(func() {
		ui.Icon(c, memoIcons[icon]).Size(tokenValue(values, "icon-size", 16), tokenValue(values, "icon-size", 16)).TextColor(theme.TextMuted).Shrink(0)
		weight := int(tokenRatio(values, "weight-medium", 500))
		if selected {
			weight = int(tokenRatio(values, "weight-semibold", 600))
		}
		ui.Text(c, label).Grow(1).MinWidth(0).MaxLines(1).Ellipsis("…").FontWeight(weight)
		ui.Text(c, fmt.Sprint(count)).FontSize(tokenValue(values, "step-minus-1", 12)).FontFeatures("tnum").TextColor(theme.TextMuted).Shrink(0)
	})
	return button
}

func (a *app) themeMenuButton(c *ui.Context, theme *ui.Theme, values map[string]string, height float32) {
	icon := "sun"
	if theme.Dark {
		icon = "moon"
	}
	button := ui.ButtonBase(c.Key("theme-menu")).Label("Theme").Tooltip("Appearance: " + themeModeName(a.themeMode)).Shrink(0).Radius(theme.Radius).MinHeight(height).PaddingY(4).PaddingX(8)
	button.Menu(func(menu *ui.Menu) {
		for _, item := range []struct {
			label string
			mode  int
		}{
			{"System", themeModeSystem},
			{"Light", themeModeLight},
			{"Dark", themeModeDark},
		} {
			if menu.Item(item.label).Radio(a.themeMode == item.mode).Chosen() {
				a.themeMode = item.mode
				a.saveThemeMode()
			}
		}
	})
	button.Children(func() {
		ui.Icon(c, memoIcons[icon]).Size(tokenValue(values, "icon-size", 16), tokenValue(values, "icon-size", 16)).TextColor(theme.TextMuted)
	})
}

func (a *app) libraryTabStrip(c *ui.Context, theme *ui.Theme, values map[string]string) {
	height := tabStripHeight(values)
	ui.Row(c.Key("library-tab-row")).Gap(tokenValue(values, "space-1", 4)).MinHeight(height).Children(func() {
		ui.ScrollHorizontal(c.Key("library-tabs-scroll")).Grow(1).MinHeight(height).Children(func() {
			tabs := ui.TabsBase(c.Key("library-tabs"), &a.libraryTab, 2)
			tabs.List.Gap(tokenValue(values, "space-1", 4)).MinHeight(height).Children(func() {
				items := []struct{ icon, label, tip string }{
					{"notes", "Show memos", "Show memo list"},
					{"tag", "Show tags", "Show library and tags"},
				}
				for i, item := range items {
					item := item
					on := i == a.libraryTab
					tab := tabs.Tab(i).MinHeight(height).PaddingY(4).PaddingX(8).Radius(theme.Radius).Label(item.label).Tooltip(item.tip)
					if on {
						tab.TextColor(theme.Text).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
					} else {
						tab.TextColor(theme.TextMuted)
						if tab.Hovered() {
							tab.TextColor(theme.Text)
						}
					}
					tab.DrawOver(func(p *ui.Painter, r ui.Rect) {
						if on {
							inset := tokenValue(values, "space-1", 4)
							bar := float32(2)
							p.Fill(ui.Rect{X: r.X + inset, Y: r.Y + r.H - bar, W: r.W - 2*inset, H: bar}, theme.Accent, bar/2)
						}
					})
					tab.Children(func() {
						ui.Icon(c, memoIcons[item.icon]).Size(tokenValue(values, "icon-size", 16), tokenValue(values, "icon-size", 16))
					})
				}
			})
			tabs.List.Changed()
		})
	})
}

func (a *app) libraryTagsPane(c *ui.Context, theme *ui.Theme, values map[string]string) {
	ui.Column(c.Key("library-tags-pane")).FillWidth().Grow(1).MinHeight(0).Gap(tokenValue(values, "space-4", 16)).Children(func() {
		ui.Text(c.Key("sidebar-library-label"), "Library").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
		allMemos := sidebarButton(c.Key("sidebar-all-memos"), theme, values, "All memos", "notes", len(a.memos), a.selectedTag == "")
		if allMemos.Clicked() {
			a.selectTag("")
		}
		ui.Text(c.Key("sidebar-tags-label"), "Tags").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
		ui.Scroll(c.Key("sidebar-tag-scroll")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key("sidebar-tags")).Gap(tokenValue(values, "space-1", 4)).Children(func() {
				for _, tag := range availableMemoTags(a.memos) {
					tag := tag
					button := sidebarButton(c.Key("sidebar-tag-"+tag), theme, values, tag, "tag", memoTagCount(a.memos, tag), a.selectedTag == tag)
					if button.Clicked() {
						a.selectTag(tag)
					}
				}
			})
		})
	})
}

func (a *app) libraryMemosPane(c *ui.Context, theme *ui.Theme, values map[string]string, filtered []memo) {
	ui.Column(c.Key("library-memos-pane")).FillWidth().Grow(1).MinHeight(0).Gap(tokenValue(values, "space-4", 16)).Children(func() {
		ui.Row(c.Key("memo-list-heading")).Gap(tokenValue(values, "space-1", 4)).Children(func() {
			ui.Text(c.Key("memo-list-title"), a.memoListTitle()).FontSize(tokenValue(values, "step-1", 17)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).Grow(1).MaxLines(1).Ellipsis("…")
			ui.Text(c.Key("memo-list-count"), fmt.Sprint(len(filtered))).FontSize(tokenValue(values, "step-minus-1", 12)).FontFeatures("tnum").TextColor(theme.TextMuted)
			if a.liveData && a.authenticated {
				button := ui.PrimaryButton(c.Key("new-memo"), "New memo").Disabled(a.loading || a.saving).MinHeight(controlMinHeight(c, values))
				if button.Clicked() {
					a.newMemo()
				}
			}
		})
		if a.liveData && !a.authenticated {
			ui.Text(c.Key("memo-list-sign-in-required"), "Sign in to load your memos.").TextColor(theme.TextMuted)
			return
		}
		if a.liveData && a.loading && len(a.memos) == 0 {
			ui.Progress(c.Key("memo-loading-progress"), -1).FillWidth()
			ui.Text(c.Key("memo-loading"), "Loading memos from your server…").TextColor(theme.TextMuted)
			a.slowLoadingHint(c, theme)
			return
		}
		if a.loadErr != "" {
			ui.Row(c.Key("memo-load-error-row")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key("memo-load-error"), a.loadErr).TextColor(theme.Warning).Grow(1)
				if a.client != nil && ui.Button(c.Key("retry-memos"), "Retry").Disabled(a.loading || a.saving).MinHeight(controlMinHeight(c, values)).Clicked() {
					a.refreshMemos()
				}
			})
			if len(a.memos) == 0 {
				return
			}
		}
		if len(filtered) == 0 {
			message := "No sample memos match. Try another search or tag."
			if a.liveData && (a.search != "" || a.selectedTag != "") {
				message = "No server memos match. Try another search or tag."
			} else if a.liveData {
				message = "No memos are available on this server. Use New memo to add one."
			}
			ui.Text(c.Key("empty-search"), message).TextColor(theme.TextMuted)
			return
		}
		ui.Scroll(c.Key("memo-list-scroll")).Grow(1).MinHeight(0).Children(func() {
			active, _ := a.activeMemo()
			ui.Column(c.Key("memo-list-rows")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				for _, item := range filtered {
					a.memoRow(c, theme, values, item, item.ID == active.ID)
				}
			})
		})
	})
}

func (a *app) unifiedLibraryPane(c *ui.Context, theme *ui.Theme, values map[string]string) ui.Element {
	if a.libraryTab != libraryTabTags {
		a.libraryTab = libraryTabMemos
	}
	pane := ui.Column(c.Key("notes-library-pane")).Label("Library and memos").FillHeight().Padding(tokenValue(values, "space-4", 16)).Gap(tokenValue(values, "space-4", 16)).Background(ui.Hex(values["sidebar"])).BorderWidth(0, tokenValue(values, "stroke", 1), 0, 0).BorderColor(ui.Hex(values["divider"]))
	pane.Children(func() {
		ui.Row(c.Key("sidebar-brand")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
			ui.Icon(c, memoIcons["notebook"]).Size(tokenValue(values, "step-2", 20), tokenValue(values, "step-2", 20))
			ui.Text(c.Key("sidebar-title"), "Memos").Role(ui.RoleHeading).Level(1).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-2", 20)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
		})
		ui.SearchField(c.Key("memo-search"), &a.search).Placeholder("Search memos").Label("Search memos").FillWidth().MinHeight(controlMinHeight(c, values))
		a.libraryTabStrip(c, theme, values)
		if a.libraryTab == libraryTabTags {
			a.libraryTagsPane(c, theme, values)
		} else {
			a.libraryMemosPane(c, theme, values, a.filteredMemos())
		}
		ui.Divider(c).Background(ui.Hex(values["divider"]))
		ui.Text(c.Key("sidebar-status"), a.connectionStatus()).FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted).MaxLines(2).Ellipsis("…")
	})
	return pane
}

func (a *app) memoListPane(c *ui.Context, theme *ui.Theme, values map[string]string, desktop bool) ui.Element {
	padding := float32(0)
	if desktop {
		padding = tokenValue(values, "space-4", 16)
	}
	panel := ui.Column(c.Key("memo-list-pane")).Label("Memo list").FillWidth().FillHeight().Padding(padding).Gap(tokenValue(values, "space-4", 16)).Background(theme.Background)
	if desktop {
		panel.BorderWidth(0, tokenValue(values, "stroke", 1), 0, 0).BorderColor(ui.Hex(values["divider"]))
	}
	panel.Children(func() {
		filtered := a.filteredMemos()
		ui.Row(c.Key("memo-list-heading")).Gap(tokenValue(values, "space-1", 4)).Children(func() {
			ui.Text(c.Key("memo-list-title"), a.memoListTitle()).FontSize(tokenValue(values, "step-1", 17)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).Grow(1).MaxLines(1).Ellipsis("…")
			ui.Text(c.Key("memo-list-count"), fmt.Sprint(len(filtered))).FontSize(tokenValue(values, "step-minus-1", 12)).FontFeatures("tnum").TextColor(theme.TextMuted)
			if a.liveData && a.authenticated {
				button := ui.PrimaryButton(c.Key("new-memo"), "New memo").Disabled(a.loading || a.saving).MinHeight(controlMinHeight(c, values))
				if button.Clicked() {
					a.newMemo()
				}
			}
		})
		if !desktop {
			ui.Row(c.Key("mobile-tag-filter-row")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key("mobile-tags-label"), "Tags").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
				label := "Filter by tag"
				if a.selectedTag != "" {
					label = "Tag: " + a.selectedTag
				}
				ui.MenuButton(c.Key("mobile-tag-filter"), label, func(menu *ui.Menu) {
					if menu.Item("All memos").Chosen() {
						a.selectTag("")
					}
					for _, tag := range availableMemoTags(a.memos) {
						tag := tag
						if menu.Item(tag).Chosen() {
							a.selectTag(tag)
						}
					}
				}).Label("Filter memos by tag").MinHeight(controlMinHeight(c, values))
				ui.Spacer(c)
				a.themeMenuButton(c, theme, values, tabStripHeight(values))
			})
		}
		ui.SearchField(c.Key("memo-search"), &a.search).Placeholder("Search memos").Label("Search memos").FillWidth().MinHeight(controlMinHeight(c, values))
		if a.liveData && !a.authenticated {
			if desktop {
				ui.Text(c.Key("memo-list-sign-in-required"), "Sign in to load your memos.").TextColor(theme.TextMuted)
			} else {
				a.signInPanel(c, theme, values)
			}
			return
		}
		if a.liveData && a.loading && len(a.memos) == 0 {
			ui.Progress(c.Key("memo-loading-progress"), -1).FillWidth()
			ui.Text(c.Key("memo-loading"), "Loading memos from your server…").TextColor(theme.TextMuted)
			a.slowLoadingHint(c, theme)
			return
		}
		if a.loadErr != "" {
			ui.Row(c.Key("memo-load-error-row")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key("memo-load-error"), a.loadErr).TextColor(theme.Warning).Grow(1)
				if a.client != nil && ui.Button(c.Key("retry-memos"), "Retry").Disabled(a.loading || a.saving).MinHeight(controlMinHeight(c, values)).Clicked() {
					a.refreshMemos()
				}
			})
			if len(a.memos) == 0 {
				return
			}
		}
		if len(filtered) == 0 {
			message := "No sample memos match. Try another search or tag."
			if a.liveData && (a.search != "" || a.selectedTag != "") {
				message = "No server memos match. Try another search or tag."
			} else if a.liveData {
				message = "No memos are available on this server. Use New memo to add one."
			}
			ui.Text(c.Key("empty-search"), message).TextColor(theme.TextMuted)
			return
		}
		ui.Scroll(c.Key("memo-list-scroll")).Grow(1).Children(func() {
			active, _ := a.activeMemo()
			ui.Column(c.Key("memo-list-rows")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				for _, item := range filtered {
					a.memoRow(c, theme, values, item, item.ID == active.ID)
				}
			})
		})
	})
	return panel
}

func (a *app) memoRow(c *ui.Context, theme *ui.Theme, values map[string]string, item memo, selected bool) {
	background := theme.Background
	if selected {
		background = theme.Selection
	}
	ui.Row(c.Key("memo-row-" + item.ID)).FillWidth().Background(background).Radius(theme.Radius).Children(func() {
		button := ui.ButtonBase(c.Key("memo-open-" + item.ID)).Grow(1).MinWidth(0).Selected(selected).TextColor(theme.Text).MinHeight(controlMinHeight(c, values)).Padding(tokenValue(values, "space-3", 12)).Background(background).Radius(theme.Radius).Label("Open memo: " + item.Title)
		if button.Hovered() {
			button.Background(theme.Selection)
		}
		button.Children(func() {
			ui.Column(c.Key("memo-row-content-" + item.ID)).FillWidth().MinWidth(0).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key("memo-row-title-text-"+item.ID), item.Title).FontSize(tokenValue(values, "step-0", 14)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).FillWidth().MaxLines(1).Ellipsis("…")
				ui.Text(c.Key("memo-row-excerpt-"+item.ID), excerpt(item.Body)).FontSize(tokenValue(values, "step-minus-1", 12)).LineHeight(tokenRatio(values, "leading-body", 1.5)).TextColor(theme.TextMuted).MaxLines(2).Ellipsis("…")
				ui.Row(c.Key("memo-row-meta-" + item.ID)).Gap(tokenValue(values, "space-2", 8)).Children(func() {
					ui.Text(c.Key("memo-row-date-"+item.ID), item.Updated).FontSize(tokenValue(values, "step-minus-1", 12)).FontFeatures("tnum").TextColor(theme.TextMuted).MaxWidth(tokenValue(values, "note-date-max-width", 72)).MaxLines(1).Ellipsis("…")
					if len(item.Tags) > 0 {
						ui.Text(c.Key("memo-row-tag-"+item.ID), "#"+item.Tags[0]).FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted).Grow(1).MinWidth(0).MaxLines(1).Ellipsis("…")
					}
				})
			})
		})
		if button.Clicked() {
			a.openMemo(item.ID, openMain)
		}
		ui.ButtonBase(c.Key("memo-actions-"+item.ID)).Menu(func(menu *ui.Menu) {
			a.openMenu(menu, item.ID)
		}).Label("Open actions for "+item.Title).Tooltip("Open actions for "+item.Title).Size(controlMinHeight(c, values), controlMinHeight(c, values)).Shrink(0).Radius(theme.Radius).Children(func() {
			ui.Icon(c, memoIcons["more"]).Size(tokenValue(values, "icon-size", 16), tokenValue(values, "icon-size", 16)).TextColor(theme.TextMuted)
		})
	})
}

func availableMemoTags(memos []memo) []string {
	seen := make(map[string]bool)
	var tags []string
	for _, item := range memos {
		for _, tag := range item.Tags {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	sort.Strings(tags)
	return tags
}

func memoTagCount(memos []memo, tag string) int {
	count := 0
	for _, item := range memos {
		if hasMemoTag(item, tag) {
			count++
		}
	}
	return count
}

func (a *app) memoListTitle() string {
	if a.selectedTag != "" {
		return a.selectedTag
	}
	return "All memos"
}

func controlMinHeight(c *ui.Context, values map[string]string) float32 {
	width, _ := c.Size()
	if width < tokenValue(values, "breakpoint-md", 992) {
		return tokenValue(values, "target", 44)
	}
	return tokenValue(values, "control-height", 36)
}

func tabStripHeight(values map[string]string) float32 {
	return tokenValue(values, "tab-height", 28)
}

func (a *app) slowLoadingHint(c *ui.Context, theme *ui.Theme) {
	remaining := 15*time.Second - c.Now().Sub(a.loadingSince)
	if remaining > 0 {
		c.After(remaining)
		return
	}
	ui.Text(c.Key("slow-loading-hint"), "This is taking longer than expected. Check your connection; the request will time out if it cannot finish.").TextColor(theme.TextMuted)
}

func (a *app) connectionStatus() string {
	status := "Sample data · not connected"
	if a.liveData {
		switch {
		case a.saving:
			status = "Saving memo…"
		case a.loading:
			status = "Updating Memos…"
		case a.loadErr != "":
			status = "Memos server unavailable"
		case !a.authenticated:
			status = "Sign in required"
		default:
			status = fmt.Sprintf("Live · %d memos", len(a.memos))
		}
	}
	return status
}

func (a *app) appHeader(c *ui.Context, theme *ui.Theme, values map[string]string) {
	ui.Row(c.Key("app-header")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
		ui.Text(c.Key("app-title"), "Memos").Role(ui.RoleHeading).Level(1).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-2", 20)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
		ui.Spacer(c)
		ui.Text(c.Key("connection-status"), a.connectionStatus()).FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted).Grow(1).MaxLines(1).Ellipsis("…").TextAlign(ui.End)
	})
}

func (a *app) readerView(c *ui.Context, theme *ui.Theme, values map[string]string) {
	gap := tokenValue(values, "space-2", 8)
	width, _ := c.Size()
	ui.Column(c.Key("reader-shell")).Fill().Padding(tokenValue(values, "space-3", 16)).Gap(gap).Background(theme.Background).Children(func() {
		a.appHeader(c, theme, values)
		ui.Divider(c).Background(ui.Hex(values["divider"]))
		ui.Row(c.Key("reader-navigation")).Gap(gap).Children(func() {
			back := ui.Button(c.Key("all-memos"), "All memos").Label("Back to all memos").MinHeight(controlMinHeight(c, values))
			if back.Clicked() {
				a.compactReader = false
			}
		})
		a.tabStrip(c, theme, values, width < tokenValue(values, "breakpoint-sm", 600))
		ui.Divider(c).Background(ui.Hex(values["divider"]))
		a.reader(c, theme, values)
	})
}

func (a *app) signInPanel(c *ui.Context, theme *ui.Theme, values map[string]string) {
	ui.Column(c.Key("memos-sign-in")).FillWidth().Gap(tokenValue(values, "space-4", 16)).Children(func() {
		ui.Text(c.Key("sign-in-heading"), "Connect to Memos").Role(ui.RoleHeading).Level(2).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-2", 20)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
		if a.client == nil {
			ui.Text(c.Key("sign-in-config-error"), a.loadErr).TextColor(theme.Warning)
			return
		}
		ui.Text(c.Key("sign-in-note"), "Your credentials go to your Memos server. The app does not save them.").TextColor(theme.TextMuted)
		ui.Field(c.Key("login-username-field"), "Username", func() {
			ui.TextInput(c.Key("login-username"), &a.username).FillWidth().Disabled(a.loading).MinHeight(controlMinHeight(c, values))
		})
		ui.Field(c.Key("login-password-field"), "Password", func() {
			ui.TextInput(c.Key("login-password"), &a.password).Password().FillWidth().Disabled(a.loading).MinHeight(controlMinHeight(c, values))
		})
		if a.loadErr != "" {
			ui.Text(c.Key("login-error"), a.loadErr).TextColor(theme.Warning)
		}
		label := "Sign in"
		if a.loading {
			label = "Signing in…"
			ui.Progress(c.Key("login-progress"), -1).FillWidth()
			a.slowLoadingHint(c, theme)
		}
		if ui.PrimaryButton(c.Key("login-submit"), label).Disabled(a.loading).MinHeight(controlMinHeight(c, values)).Clicked() {
			a.signIn()
		}
	})
}

func (a *app) tabStrip(c *ui.Context, theme *ui.Theme, values map[string]string, compact bool) {
	labels := make([]string, len(a.tabs))
	for i, tab := range a.tabs {
		labels[i] = "Main window"
		if tab.ID != "main" {
			if item, ok := a.memoByID(tab.MemoID); ok {
				labels[i] = item.Title
			}
		}
	}
	if a.activeTabIndex < 0 || a.activeTabIndex >= len(a.tabs) {
		a.activeTabIndex = 0
		a.activeTabID = a.tabs[0].ID
	}
	height := tabStripHeight(values)
	ui.Row(c.Key("reader-tabs-row")).Gap(tokenValue(values, "space-1", 4)).MinHeight(height).Children(func() {
		ui.ScrollHorizontal(c.Key("reader-tabs-scroll")).Grow(1).MinHeight(height).Children(func() {
			tabs := ui.TabsBase(c.Key("reader-tabs"), &a.activeTabIndex, len(labels))
			tabs.List.Gap(tokenValue(values, "space-1", 4)).MinHeight(height).Children(func() {
				for i, label := range labels {
					label := label
					on := i == a.activeTabIndex
					tab := tabs.Tab(i).MinHeight(height).PaddingY(4).PaddingX(8).Radius(theme.Radius).Label(label)
					if on {
						tab.TextColor(theme.Text).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
					} else {
						tab.TextColor(theme.TextMuted)
						if tab.Hovered() {
							tab.TextColor(theme.Text)
						}
					}
					tab.DrawOver(func(p *ui.Painter, r ui.Rect) {
						if on {
							inset := tokenValue(values, "space-1", 4)
							bar := float32(2)
							p.Fill(ui.Rect{X: r.X + inset, Y: r.Y + r.H - bar, W: r.W - 2*inset, H: bar}, theme.Accent, bar/2)
						}
					})
					tab.Children(func() {
						ui.Text(c, label).MaxLines(1).Ellipsis("…").SingleLine()
					})
				}
			})
			if tabs.List.Changed() && a.activeTabIndex >= 0 && a.activeTabIndex < len(a.tabs) {
				a.activeTabID = a.tabs[a.activeTabIndex].ID
				a.compactReader = true
			}
		})
		if a.activeTabID != "main" {
			closeLabel := "Close tab"
			if compact {
				closeLabel = "×"
			}
			button := ui.Button(c.Key("close-active-tab"), closeLabel).Label("Close current tab").Tooltip("Close current tab").MinHeight(height).PaddingY(4).PaddingX(8)
			if button.Clicked() {
				a.closeActiveTab()
			}
		}
		if item, ok := a.activeMemo(); ok && !a.composerOpen {
			if a.liveData && a.authenticated {
				if ui.Button(c.Key("edit-memo"), "Edit").Disabled(a.loading || a.saving).MinHeight(height).PaddingY(4).PaddingX(8).Clicked() {
					a.editMemo(item.ID)
				}
			}
			ui.MenuButton(c, "Open options", func(menu *ui.Menu) {
				a.openMenu(menu, item.ID)
			}).Tooltip("Choose where to open this memo").MinHeight(height).PaddingY(4).PaddingX(8)
		}
		a.themeMenuButton(c, theme, values, height)
	})
}

func (a *app) reader(c *ui.Context, theme *ui.Theme, values map[string]string) {
	if a.composerOpen {
		a.memoEditor(c, theme, values)
		return
	}
	item, ok := a.activeMemo()
	if !ok {
		message := "No memo is open."
		if a.loading {
			message = "Loading memo…"
		} else if a.liveData {
			message = "No memo is available. Check the server status in the list."
		}
		ui.Text(c.Key("reader-empty"), message).TextColor(theme.TextMuted)
		return
	}
	ui.Scroll(c.Key("reader-scroll")).Grow(1).FillWidth().Children(func() {
		ui.Column(c.Key("reader-article")).FillWidth().MaxWidth(tokenValue(values, "reader-max-width", 640)).Padding(0, 0, tokenValue(values, "space-1", 4), 0).Gap(tokenValue(values, "space-4", 16)).AlignSelf(ui.Start).Children(func() {
			ui.Column(c.Key("reader-heading")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key("reader-updated"), "Updated "+item.Updated).FontSize(tokenValue(values, "step-minus-1", 12)).FontFeatures("tnum").TextColor(theme.TextMuted)
				ui.Text(c.Key("reader-title"), item.Title).Role(ui.RoleHeading).Level(2).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-3", 24)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).LineHeight(tokenRatio(values, "leading-display", 1.2))
				ui.Row(c.Key("reader-tags")).Gap(tokenValue(values, "space-2", 8)).Wrap().Children(func() {
					for i, tag := range item.Tags {
						ui.Badge(c.Key(fmt.Sprintf("reader-tag-%d", i)), tag).FontSize(tokenValue(values, "step-minus-1", 12)).Background(theme.Selection).TextColor(theme.TextMuted).Radius(theme.Radius).MaxWidth(tokenValue(values, "tag-max-width", 160)).MaxLines(1).Ellipsis("…")
					}
				})
			})
			ui.Divider(c).Background(ui.Hex(values["divider"]))
			a.renderBody(c, theme, values, item)
		})
	})
}

func (a *app) memoEditor(c *ui.Context, theme *ui.Theme, values map[string]string) {
	title := "New memo"
	if a.editingMemoID != "" {
		title = "Edit memo"
	}
	if a.composerMode != composerModeSource {
		a.composerMode = composerModeLive
	}
	if a.composerMode == composerModeSource && a.editingBlock >= 0 {
		a.commitBlockEdit()
	}
	sideCompletion := a.completionSidebarOpen(c, values)
	editor := ui.Column(c.Key("memo-editor")).FillWidth().Grow(1).AlignSelf(ui.Start).Gap(tokenValue(values, "space-4", 16))
	if !sideCompletion {
		editor.MaxWidth(tokenValue(values, "reader-max-width", 640))
	}
	editor.Children(func() {
		ui.Row(c.Key("memo-editor-header")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
			ui.Text(c.Key("memo-editor-title"), title).Role(ui.RoleHeading).Level(2).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-2", 20)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).Grow(1)
			mode := ui.SegmentedBase(c.Key("memo-editor-mode"), &a.composerMode, 2)
			mode.Track.Label("Editor mode").Padding(tokenValue(values, "space-1", 4)).Gap(tokenValue(values, "space-1", 4)).Background(theme.Selection).Radius(theme.Radius).Children(func() {
				for index, label := range []string{"Live", "Source"} {
					segment := mode.Segment(index).MinHeight(controlMinHeight(c, values)).PaddingX(tokenValue(values, "space-2", 8)).Radius(theme.Radius).TextColor(theme.TextMuted)
					if a.composerMode == index {
						segment.Background(theme.Background).TextColor(theme.Text)
					}
					segment.Children(func() { ui.Text(c, label).FontWeight(int(tokenRatio(values, "weight-medium", 500))) })
				}
			})
			mode.Track.Changed()
		})
		if a.composerMode == composerModeSource {
			ui.Text(c.Key("memo-editor-help"), "Source · raw Markdown. Changes stay local until you save.").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
			ui.Text(c.Key("memo-content-label"), "Memo content").FontWeight(int(tokenRatio(values, "weight-medium", 500)))
			sourceField := func() {
				ui.TextArea(c.Key("memo-content"), &a.composerContent).Label("Memo content").Font(fontFamily(values["font-mono"])).LineHeight(tokenRatio(values, "leading-body", 1.5)).Padding(tokenValue(values, "space-4", 16)).Placeholder("Write your memo in Markdown…").FillWidth().Grow(1).MinWidth(0).MinHeight(tokenValue(values, "editor-min-height", 320)).Disabled(a.saving)
			}
			if sideCompletion {
				ui.Row(c.Key("memo-source-main")).Grow(1).MinHeight(0).Gap(tokenValue(values, "space-2", 8)).AlignItems(ui.Stretch).Children(func() {
					sourceField()
					a.linkCompletionSidebar(c, theme, values)
				})
			} else {
				sourceField()
				a.linkCompletionBox(c, theme, values)
			}
		} else {
			ui.Text(c.Key("memo-editor-help"), "Live Preview · click any text to edit it. Changes stay local until you save.").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
			if strings.TrimSpace(a.composerContent) == "" && a.editingBlock != 0 {
				a.startBlockEdit(0)
			}
			blocks := a.composerBlocks()
			if strings.TrimSpace(a.composerContent) == "" {
				ui.Text(c.Key("memo-preview-empty"), "Empty note · write the first paragraph below.").TextColor(theme.TextMuted)
			}
			if sideCompletion {
				ui.Row(c.Key("memo-live-main")).Grow(1).MinHeight(0).Gap(tokenValue(values, "space-2", 8)).AlignItems(ui.Stretch).Children(func() {
					ui.Scroll(c.Key("memo-live-scroll")).FillWidth().Grow(1).MinHeight(0).MinWidth(0).Children(func() {
						a.memoLiveBlocks(c, theme, values, blocks, false)
					})
					a.linkCompletionSidebar(c, theme, values)
				})
			} else {
				ui.Scroll(c.Key("memo-live-scroll")).FillWidth().Grow(1).Children(func() {
					a.memoLiveBlocks(c, theme, values, blocks, true)
				})
			}
		}
		if a.saveErr != "" {
			ui.Text(c.Key("memo-save-error"), a.saveErr).TextColor(theme.Warning)
		}
		ui.Row(c.Key("memo-editor-actions")).Gap(tokenValue(values, "space-2", 8)).Children(func() {
			ui.Spacer(c)
			cancel := ui.Button(c.Key("cancel-memo-edit"), "Cancel").Label("Cancel").Disabled(a.saving).MinHeight(controlMinHeight(c, values))
			if cancel.Clicked() {
				a.cancelMemoEdit()
			}
			label := "Save memo"
			if a.saving {
				label = "Saving…"
			}
			save := ui.PrimaryButton(c.Key("save-memo"), label).Label(label).Disabled(a.saving || a.loading).MinHeight(controlMinHeight(c, values))
			if save.Clicked() {
				a.saveMemo()
			}
		})
	})
}

func (a *app) renderBody(c *ui.Context, theme *ui.Theme, values map[string]string, item memo) {
	blocks := strings.Split(item.Body, "\n\n")
	for blockIndex, block := range blocks {
		if strings.HasPrefix(block, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(block, "## "))
			ui.Text(c.Key(fmt.Sprintf("body-heading-%d", blockIndex)), title).Role(ui.RoleHeading).Level(3).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-1", 17)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
			continue
		}
		if strings.HasPrefix(block, "- ") {
			for lineIndex, line := range strings.Split(block, "\n") {
				text := strings.TrimSpace(strings.TrimPrefix(line, "- "))
				ui.Row(c.Key(fmt.Sprintf("body-list-%d-%d", blockIndex, lineIndex))).Gap(tokenValue(values, "space-2", 8)).Children(func() {
					ui.Text(c.Key(fmt.Sprintf("bullet-%d-%d", blockIndex, lineIndex)), "•").TextColor(theme.TextMuted)
					a.inlineText(c, theme, values, text, fmt.Sprintf("body-list-text-%d-%d", blockIndex, lineIndex))
				})
			}
			continue
		}
		a.inlineText(c, theme, values, block, fmt.Sprintf("body-paragraph-%d", blockIndex))
	}
}

func (a *app) completionSuggestions() []memo {
	query, ok := a.linkCompletionQuery()
	if !ok {
		return nil
	}
	return a.linkSuggestions(query)
}

func (a *app) linkSuggestionList(c *ui.Context, theme *ui.Theme, values map[string]string, suggestions []memo) {
	ui.Text(c.Key("link-complete-label"), "Link to memo").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
	for _, suggestion := range suggestions {
		suggestion := suggestion
		button := ui.ButtonBase(c.Key("link-suggest-" + suggestion.ID)).Label("Insert link to " + suggestion.Title).Tooltip("Insert link to " + suggestion.Title).FillWidth().MinHeight(tabStripHeight(values)).PaddingY(4).PaddingX(8).Radius(theme.Radius)
		button.Children(func() {
			ui.Icon(c, memoIcons["tag"]).Size(tokenValue(values, "icon-size", 16), tokenValue(values, "icon-size", 16)).TextColor(theme.TextMuted).Shrink(0)
			ui.Text(c, suggestion.Title).Grow(1).MinWidth(0).MaxLines(1).Ellipsis("…")
		})
		if button.Clicked() {
			a.acceptLinkSuggestion(suggestion.Title)
		}
	}
}

func (a *app) linkCompletionBox(c *ui.Context, theme *ui.Theme, values map[string]string) {
	suggestions := a.completionSuggestions()
	if len(suggestions) == 0 {
		return
	}
	ui.Column(c.Key("link-complete-box")).Gap(tokenValue(values, "space-1", 4)).Children(func() {
		a.linkSuggestionList(c, theme, values, suggestions)
	})
}

func (a *app) linkCompletionSidebar(c *ui.Context, theme *ui.Theme, values map[string]string) {
	suggestions := a.completionSuggestions()
	if len(suggestions) == 0 {
		return
	}
	ui.Column(c.Key("link-complete-sidebar")).Label("Link suggestions").Width(tokenValue(values, "link-sidebar-width", 264)).FillHeight().Padding(tokenValue(values, "space-3", 12)).Gap(tokenValue(values, "space-2", 8)).BorderWidth(0, 0, 0, tokenValue(values, "stroke", 1)).BorderColor(ui.Hex(values["divider"])).Children(func() {
		a.linkSuggestionList(c, theme, values, suggestions)
	})
}

func (a *app) completionSidebarOpen(c *ui.Context, values map[string]string) bool {
	if len(a.completionSuggestions()) == 0 {
		return false
	}
	width, _ := c.Size()
	return width >= tokenValue(values, "breakpoint-md", 992)
}

func (a *app) memoLiveBlocks(c *ui.Context, theme *ui.Theme, values map[string]string, blocks []string, inlineCompletion bool) {
	ui.Column(c.Key("memo-live-blocks")).FillWidth().MaxWidth(tokenValue(values, "reader-max-width", 640)).AlignSelf(ui.Start).Gap(tokenValue(values, "space-2", 8)).Children(func() {
		for i := range blocks {
			i := i
			if i == a.editingBlock {
				field := ui.TextAreaBase(c.Key(fmt.Sprintf("memo-live-edit-%d", i)), &a.editingDraft).Label(fmt.Sprintf("Edit block %d", i+1)).FillWidth().Lines(1, 32).Disabled(a.saving).AutoFocus().Placeholder("Write Markdown for this block…")
				if strings.HasPrefix(blocks[i], "## ") {
					field.Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-1", 17)).FontWeight(int(tokenRatio(values, "weight-semibold", 600))).LineHeight(tokenRatio(values, "leading-display", 1.2))
				} else {
					field.Font(fontFamily(values["font-sans"])).FontSize(tokenValue(values, "step-0", 14)).LineHeight(tokenRatio(values, "leading-body", 1.5))
				}
				if inlineCompletion {
					a.linkCompletionBox(c, theme, values)
				}
				continue
			}
			block := ui.ButtonBase(c.Key(fmt.Sprintf("memo-live-blockrow-%d", i))).Label(fmt.Sprintf("Edit block %d", i+1)).Tooltip("Click to edit this block").FillWidth().Radius(theme.Radius).PaddingY(tokenValue(values, "space-1", 4)).PaddingX(tokenValue(values, "space-2", 8))
			if block.Hovered() {
				block.Background(theme.Selection)
			}
			block.Children(func() {
				ui.Column(c.Key(fmt.Sprintf("memo-live-block-%d", i))).FillWidth().MinWidth(0).Children(func() {
					a.renderLiveBlock(c, theme, values, blocks[i], i)
				})
			})
			if block.Clicked() {
				a.startBlockEdit(i)
			}
		}
		add := ui.ButtonBase(c.Key("memo-add-block")).Label("Add paragraph").Tooltip("Add paragraph").FillWidth().MinHeight(tabStripHeight(values)).PaddingY(4).Radius(theme.Radius)
		if add.Hovered() {
			add.Background(theme.Selection)
		}
		add.Children(func() {
			ui.Text(c, "+  Add paragraph").TextColor(theme.TextMuted).FontSize(tokenValue(values, "step-minus-1", 12))
		})
		if add.Clicked() {
			if strings.TrimSpace(a.composerContent) == "" {
				a.composerContent = ""
				a.startBlockEdit(0)
			} else {
				a.composerContent = strings.TrimRight(a.composerContent, "\n") + "\n\n"
				a.startBlockEdit(len(a.composerBlocks()) - 1)
			}
		}
	})
}

func (a *app) renderLiveBlock(c *ui.Context, theme *ui.Theme, values map[string]string, block string, blockIndex int) {
	if strings.TrimSpace(block) == "" {
		ui.Text(c.Key(fmt.Sprintf("live-empty-%d", blockIndex)), "Empty block — edit to add text.").FontSize(tokenValue(values, "step-minus-1", 12)).TextColor(theme.TextMuted)
		return
	}
	if strings.HasPrefix(block, "## ") {
		title := strings.TrimSpace(strings.TrimPrefix(block, "## "))
		ui.Text(c.Key(fmt.Sprintf("live-heading-%d", blockIndex)), title).Role(ui.RoleHeading).Level(3).Font(fontFamily(values["font-display"])).FontSize(tokenValue(values, "step-1", 17)).FontWeight(int(tokenRatio(values, "weight-semibold", 600)))
		return
	}
	if strings.HasPrefix(block, "- ") {
		for lineIndex, line := range strings.Split(block, "\n") {
			text := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			ui.Row(c.Key(fmt.Sprintf("live-list-%d-%d", blockIndex, lineIndex))).Gap(tokenValue(values, "space-2", 8)).Children(func() {
				ui.Text(c.Key(fmt.Sprintf("live-bullet-%d-%d", blockIndex, lineIndex)), "•").TextColor(theme.TextMuted)
				a.inlineText(c, theme, values, text, fmt.Sprintf("live-list-text-%d-%d", blockIndex, lineIndex))
			})
		}
		return
	}
	a.inlineText(c, theme, values, block, fmt.Sprintf("live-paragraph-%d", blockIndex))
}

func (a *app) inlineText(c *ui.Context, theme *ui.Theme, values map[string]string, text, key string) {
	ui.RichText(c.Key(key)).LineHeight(tokenRatio(values, "leading-body", 1.5)).Children(func() {
		for index, part := range parseInline(text, a.memos) {
			part := part
			childKey := fmt.Sprintf("%s-part-%d", key, index)
			if part.URL != "" {
				link := ui.Link(c.Key(childKey), "", part.URL).Label("Open web link: " + part.URL).Tooltip(part.URL)
				link.Children(func() {
					ui.Text(c, part.Text).TextColor(theme.Accent).Underline()
				})
				continue
			}
			if part.Tag != "" {
				tag := ui.Text(c.Key(childKey), "#"+part.Tag).Label("Tag #" + part.Tag).Tooltip("Filter by #" + part.Tag).TextColor(theme.Accent)
				tag.Menu(func(menu *ui.Menu) {
					if menu.Item("Filter by #" + part.Tag).Chosen() {
						a.selectTag(part.Tag)
					}
				})
				continue
			}
			if !part.IsLink {
				ui.Text(c.Key(childKey), part.Text)
				continue
			}
			if part.TargetID == "" {
				ui.Text(c.Key(childKey), part.Text).TextColor(theme.TextMuted).DottedUnderline().Tooltip("Page not found in sample memos")
				continue
			}
			link := ui.Text(c.Key(childKey), part.Text).TextColor(theme.Accent).Underline().Label("Open wiki page: " + part.Text).Tooltip("Choose how to open " + part.Text)
			link.Menu(func(menu *ui.Menu) {
				a.openMenu(menu, part.TargetID)
			})
		}
	})
}

func (a *app) openMenu(menu *ui.Menu, memoID string) {
	if menu.Item("Edit memo").Disabled(!a.liveData || !a.authenticated || a.client == nil).Chosen() {
		a.editMemo(memoID)
	}
	if menu.Item("Open in main window").Chosen() {
		a.openMemo(memoID, openMain)
	}
	if menu.Item("Open in new tab").Chosen() {
		a.openMemo(memoID, openForeground)
	}
	if menu.Item("Open in background tab").Chosen() {
		a.openMemo(memoID, openBackground)
	}
}

func excerpt(body string) string {
	body = strings.ReplaceAll(body, "## ", "")
	body = strings.ReplaceAll(body, "- ", "• ")
	body = strings.Join(strings.Fields(body), " ")
	body = strings.ReplaceAll(body, "[[", "")
	body = strings.ReplaceAll(body, "]]", "")
	runes := []rune(body)
	if len(runes) > 112 {
		return string(runes[:109]) + "…"
	}
	return body
}
