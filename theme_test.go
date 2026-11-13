package main

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestWarningTokenMeetsTextContrastInBothThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		values := appTokens(dark)
		for _, background := range []string{values["bg"], values["surface"]} {
			ratio := contrastRatio(values["warning"], background)
			if ratio < 4.5 {
				t.Errorf("dark=%t warning contrast on %s = %.2f:1, want at least 4.5:1", dark, background, ratio)
			}
		}
	}
}

func TestThemeModePersistsAcrossLaunches(t *testing.T) {
	t.Setenv("MEMOS_CONFIG_DIR", t.TempDir())
	if got := loadThemeMode(); got != themeModeSystem {
		t.Fatalf("missing config loaded %d, want system", got)
	}
	a := newApp()
	a.themeMode = themeModeDark
	a.saveThemeMode()
	if got := loadThemeMode(); got != themeModeDark {
		t.Fatalf("saved dark loaded %d", got)
	}
	path := themeConfigPath()
	if err := os.WriteFile(path, []byte("bogus\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadThemeMode(); got != themeModeSystem {
		t.Fatalf("bogus config loaded %d, want system", got)
	}
}

func TestCompactThemeMetricsInBothThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		theme := themeFor(dark)
		if theme.FontSize != 14 {
			t.Errorf("dark=%t base font size = %.1f, want 14", dark, theme.FontSize)
		}
		if theme.Spacing != 4 {
			t.Errorf("dark=%t spacing unit = %.1f, want 4", dark, theme.Spacing)
		}
		if theme.Radius != 8 {
			t.Errorf("dark=%t radius = %.1f, want 8", dark, theme.Radius)
		}
	}
}

func TestNeutralThemeContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		values := appTokens(dark)
		for _, background := range []string{"bg", "sidebar", "accent-soft"} {
			for _, foreground := range []string{"fg", "muted"} {
				if ratio := contrastRatio(values[foreground], values[background]); ratio < 4.5 {
					t.Errorf("dark=%t %s on %s contrast %.2f, want 4.5", dark, foreground, background, ratio)
				}
			}
		}
		if ratio := contrastRatio(values["on-accent"], values["accent"]); ratio < 4.5 {
			t.Errorf("dark=%t primary button contrast %.2f, want 4.5", dark, ratio)
		}
		if ratio := contrastRatio(values["border"], values["surface"]); ratio < 3 {
			t.Errorf("dark=%t control border contrast %.2f, want 3", dark, ratio)
		}
	}
}

func contrastRatio(foreground, background string) float64 {
	fg := relativeLuminance(foreground)
	bg := relativeLuminance(background)
	if fg < bg {
		fg, bg = bg, fg
	}
	return (fg + 0.05) / (bg + 0.05)
}

func relativeLuminance(value string) float64 {
	value = strings.TrimPrefix(value, "#")
	red, _ := strconv.ParseInt(value[0:2], 16, 64)
	green, _ := strconv.ParseInt(value[2:4], 16, 64)
	blue, _ := strconv.ParseInt(value[4:6], 16, 64)
	channels := []float64{float64(red) / 255, float64(green) / 255, float64(blue) / 255}
	for index, channel := range channels {
		if channel <= 0.04045 {
			channels[index] = channel / 12.92
		} else {
			channels[index] = math.Pow((channel+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}
