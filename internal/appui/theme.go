package appui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// applyAppTheme 应用现代中性蓝灰桌面主题（不使用任何复古或 AI 国风配色）。
func applyAppTheme(c *ui.Context) *ui.Theme {
	base := ui.LightTheme()
	accent := ui.Hex("#0284C7")
	if c.Theme().Dark {
		base = ui.DarkTheme()
		accent = ui.Hex("#38BDF8")
	}
	t := *base.WithAccent(accent)
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13.5
	if c.Theme().Dark {
		t.Background = ui.Hex("#0F172A")
		t.Surface = ui.Hex("#1E293B")
		t.SurfaceHover = ui.Hex("#273548")
		t.SurfacePressed = ui.Hex("#334155")
		t.Border = ui.Hex("#334155")
		t.TextMuted = ui.Hex("#94A3B8")
	} else {
		t.Background = ui.Hex("#F8FAFC")
		t.Surface = ui.Hex("#FFFFFF")
		t.SurfaceHover = ui.Hex("#F1F5F9")
		t.SurfacePressed = ui.Hex("#E2E8F0")
		t.Border = ui.Hex("#E2E8F0")
		t.TextMuted = ui.Hex("#64748B")
	}
	t = *t.WithPreferences(c.Preferences())
	c.SetTheme(&t)
	return &t
}

// ApplyThemeSource 将设置中的外观选项应用到 MyGo 主题引擎。
func ApplyThemeSource(mode string) {
	switch mode {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}
