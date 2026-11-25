package appui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// applyAppTheme 应用现代高透气移动主题（浅色中性摄影画布 + 天际蓝主色，绝无复古或国风配色）。
func applyAppTheme(c *ui.Context) *ui.Theme {
	base := ui.LightTheme()
	accent := ui.Hex("#0284C7")
	if c.Theme().Dark {
		base = ui.DarkTheme()
		accent = ui.Hex("#38BDF8")
	}
	t := *base.WithPreferences(c.Preferences()).WithAccent(accent)
	t.Radius = 14
	t.Spacing = 6
	t.FontSize = 15

	if c.Theme().Dark {
		t.Background = ui.Hex("#0B0F17")
		t.Surface = ui.Hex("#141B26")
		t.SurfaceHover = ui.Hex("#1C2534")
		t.SurfacePressed = ui.Hex("#253042")
		t.Border = ui.RGBA(255, 255, 255, 0.08)
		t.Text = ui.Hex("#F8FAFC")
		t.TextMuted = ui.Hex("#94A3B8")
		t.AccentText = ui.Hex("#06131E")
		t.Success = ui.Hex("#10B981")
		t.Warning = ui.Hex("#F59E0B")
		t.Danger = ui.Hex("#F43F5E")
	} else {
		t.Background = ui.Hex("#F3F5F8")
		t.Surface = ui.Hex("#FFFFFF")
		t.SurfaceHover = ui.Hex("#EEF2F6")
		t.SurfacePressed = ui.Hex("#E2E8F0")
		t.Border = ui.Hex("#E5E9F0")
		t.Text = ui.Hex("#0F172A")
		t.TextMuted = ui.Hex("#64748B")
		t.AccentText = ui.RGB(255, 255, 255)
		t.Success = ui.Hex("#059669")
		t.Warning = ui.Hex("#D97706")
		t.Danger = ui.Hex("#E11D48")
	}
	c.SetTheme(&t)
	return &t
}

// ApplyThemeSource 将设置中的外观选项同步至 MyGo 主题引擎。
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

func softPillColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#1D2736")
	}
	return ui.Hex("#EBF0F5")
}

func secondaryTextColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#CBD5E1")
	}
	return ui.Hex("#475569")
}

func subtleHairlineColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.07)
	}
	return ui.Hex("#E8EDF3")
}

// elevatedCard 构建无硬边框、带柔和阴影层级的现代圆角卡片容器。
func elevatedCard(c *ui.Context, t *ui.Theme) ui.Element {
	col := ui.Column(c).FillWidth().Background(t.Surface).Radius(16)
	if t.Dark {
		return col.Border(1, ui.RGBA(255, 255, 255, 0.07))
	}
	return col.Shadow(0, 2, 10, 0, ui.RGBA(15, 23, 42, 0.055))
}

func elevatedRowCard(c *ui.Context, t *ui.Theme) ui.Element {
	row := ui.Row(c).FillWidth().Background(t.Surface).Radius(16)
	if t.Dark {
		return row.Border(1, ui.RGBA(255, 255, 255, 0.07))
	}
	return row.Shadow(0, 2, 10, 0, ui.RGBA(15, 23, 42, 0.055))
}

// filterPill 构建圆角柔和胶囊筛选按钮。
func filterPill(c *ui.Context, key any, label string, active bool, t *ui.Theme) ui.Element {
	bg := softPillColor(t)
	fg := secondaryTextColor(t)
	weight := 500
	if active {
		bg = t.Accent
		fg = t.AccentText
		weight = 600
	}
	el := ui.Row(c.Key(key)).Height(34).PaddingX(14).Radius(17).Background(bg).
		AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
		Role(ui.RoleButton).Label(label).Checked(active).
		Children(func() {
			ui.Text(c, label).FontSize(13).FontWeight(weight).TextColor(fg).SingleLine()
		})
	return el
}

// segmentedPillStrip 构建现代滑动风格的分段胶囊切换条。
func segmentedPillStrip(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	changed := false
	strip := ui.Row(c.Key(key)).FillWidth().Padding(3).Radius(13).Background(softPillColor(t)).Gap(2).Role(ui.RoleTabList)
	strip.Children(func() {
		for i, lbl := range labels {
			active := *selected == i
			item := ui.Row(c.Key(fmt.Sprintf("%s-%d", key, i))).Grow(1).Basis(0).Height(34).Radius(10).
				AlignItems(ui.Center).Justify(ui.Center).PaddingX(6).
				Role(ui.RoleTab).Label(lbl).Checked(active)
			fg := t.TextMuted
			weight := 500
			if active {
				item.Background(t.Surface)
				if !t.Dark {
					item.Shadow(0, 1, 4, 0, ui.RGBA(15, 23, 42, 0.08))
				}
				fg = t.Text
				weight = 600
			}
			item.Children(func() {
				ui.Text(c, lbl).FontSize(13).FontWeight(weight).TextColor(fg).SingleLine()
			})
			if item.Clicked() && !active {
				*selected = i
				changed = true
			}
		}
	})
	return changed
}

func circleIconBtn(c *ui.Context, key any, label string, svg *ui.SVG, bg, fg ui.Color, size float32) ui.Element {
	return ui.Box(c.Key(key)).Size(size, size).Radius(size/2).Background(bg).Center().Shrink(0).
		Role(ui.RoleButton).Label(label).Children(func() {
		ui.Icon(c, svg).FontSize(size * 0.48).TextColor(fg)
	})
}

func statusPill(c *ui.Context, text string, bg, fg ui.Color) {
	ui.Row(c).Padding(3, 9).Radius(999).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(11.5).FontWeight(600).TextColor(fg).SingleLine()
	})
}

func fallbackText(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
