package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed tokens.css
var tokenCSS string

func tokenBlock(selector string) map[string]string {
	start := strings.Index(tokenCSS, selector)
	if start < 0 {
		return nil
	}
	open := strings.Index(tokenCSS[start:], "{")
	if open < 0 {
		return nil
	}
	start += open + 1
	close := strings.Index(tokenCSS[start:], "}")
	if close < 0 {
		return nil
	}
	values := make(map[string]string)
	for _, line := range strings.Split(tokenCSS[start:start+close], "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ";"))
		if !strings.HasPrefix(line, "--") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "--"), ":", 2)
		if len(parts) == 2 {
			values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return values
}

func tokenValue(values map[string]string, name string, fallback float32) float32 {
	value := strings.TrimSpace(values[name])
	scale := float64(1)
	switch {
	case strings.HasSuffix(value, "px"):
		value = strings.TrimSuffix(value, "px")
	case strings.HasSuffix(value, "rem"):
		value = strings.TrimSuffix(value, "rem")
		scale = 16
	default:
		return fallback
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil {
		return fallback
	}
	return float32(number * scale)
}

func tokenRatio(values map[string]string, name string, fallback float32) float32 {
	number, err := strconv.ParseFloat(strings.TrimSpace(values[name]), 32)
	if err != nil {
		return fallback
	}
	return float32(number)
}

func themeFor(dark bool) *ui.Theme {
	values := tokenBlock(":root")
	if dark {
		for key, value := range tokenBlock(`[data-theme="dark"]`) {
			values[key] = value
		}
	}
	theme := ui.LightTheme()
	if dark {
		theme = ui.DarkTheme()
	}
	background := ui.Hex(values["bg"])
	surface := ui.Hex(values["surface"])
	foreground := ui.Hex(values["fg"])
	muted := ui.Hex(values["muted"])
	border := ui.Hex(values["border"])
	accent := ui.Hex(values["accent"])
	accentMid := ui.Hex(values["accent-mid"])
	accentSoft := ui.Hex(values["accent-soft"])
	warning := ui.Hex(values["warning"])
	theme.Background = background
	theme.Surface = surface
	theme.SurfaceHover = accentSoft
	theme.SurfacePressed = accentSoft
	theme.Border = border
	theme.Text = foreground
	theme.TextMuted = muted
	theme.Accent = accent
	theme.AccentHover = accentMid
	theme.AccentPressed = accent
	theme.AccentText = ui.Hex(values["on-accent"])
	theme.Selection = accentSoft
	theme.Focus = accent
	theme.Warning = warning
	theme.Scrollbar = ui.Hex(values["border"])
	theme.Inverse = foreground
	theme.InverseText = background
	theme.Radius = tokenValue(values, "radius", theme.Radius)
	theme.Spacing = tokenValue(values, "space-1", theme.Spacing)
	theme.FontSize = tokenValue(values, "step-0", theme.FontSize)
	theme.Font = fontFamily(values["font-sans"])
	return theme
}

func themeConfigPath() string {
	if dir := strings.TrimSpace(os.Getenv("MEMOS_CONFIG_DIR")); dir != "" {
		return filepath.Join(dir, "theme")
	}
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "memos-mygo", "theme")
}

func themeModeName(mode int) string {
	switch mode {
	case themeModeLight:
		return "light"
	case themeModeDark:
		return "dark"
	default:
		return "system"
	}
}

func loadThemeMode() int {
	data, err := os.ReadFile(themeConfigPath())
	if err != nil {
		return themeModeSystem
	}
	switch strings.TrimSpace(string(data)) {
	case "light":
		return themeModeLight
	case "dark":
		return themeModeDark
	default:
		return themeModeSystem
	}
}

func (a *app) saveThemeMode() {
	path := themeConfigPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(themeModeName(a.themeMode)+"\n"), 0o644)
}

func fontFamily(value string) string {
	family := strings.TrimSpace(strings.Split(value, ",")[0])
	return strings.Trim(family, `"'`)
}

func appTokens(dark bool) map[string]string {
	values := tokenBlock(":root")
	if dark {
		for key, value := range tokenBlock(`[data-theme="dark"]`) {
			values[key] = value
		}
	}
	return values
}
