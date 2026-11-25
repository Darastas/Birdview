package appui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const keyboardInsetThreshold = 120

// applyAppTheme 应用极简中性摄影画布主题（纯白浅色 / OLED 炭黑深色，克制单色交互，让生态摄影成为视觉中心）。
func applyAppTheme(c *ui.Context) *ui.Theme {
	base := ui.LightTheme()
	accent := ui.Hex("#18181B")
	if c.Theme().Dark {
		base = ui.DarkTheme()
		accent = ui.Hex("#FAFAFA")
	}
	t := *base.WithPreferences(c.Preferences()).WithAccent(accent)
	t.Radius = 14
	t.Spacing = 6
	t.FontSize = 15

	if c.Theme().Dark {
		t.Background = ui.Hex("#09090B")
		t.Surface = ui.Hex("#16161A")
		t.SurfaceHover = ui.Hex("#202026")
		t.SurfacePressed = ui.Hex("#2A2A32")
		t.Border = ui.RGBA(255, 255, 255, 0.08)
		t.Text = ui.Hex("#FAFAFA")
		t.TextMuted = ui.Hex("#71717A")
		t.AccentText = ui.Hex("#09090B")
		t.Success = ui.Hex("#22C55E")
		t.Warning = ui.Hex("#FBBF24")
		t.Danger = ui.Hex("#F87171")
	} else {
		t.Background = ui.Hex("#FFFFFF")
		t.Surface = ui.Hex("#F5F5F7")
		t.SurfaceHover = ui.Hex("#ECECEF")
		t.SurfacePressed = ui.Hex("#E4E4E8")
		t.Border = ui.Hex("#EFEFF2")
		t.Text = ui.Hex("#111113")
		t.TextMuted = ui.Hex("#8E8E93")
		t.AccentText = ui.RGB(255, 255, 255)
		t.Success = ui.Hex("#16A34A")
		t.Warning = ui.Hex("#D97706")
		t.Danger = ui.Hex("#EF4444")
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
	return t.Surface
}

func secondaryTextColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#A1A1AA")
	}
	return ui.Hex("#52525B")
}

func subtleHairlineColor(t *ui.Theme) ui.Color {
	return t.Border
}

// flatSurface 创建无阴影、纯平克制的圆角功能块容器（仅用于统计带、设置分组等少量块）。
func flatSurface(c *ui.Context, t *ui.Theme) ui.Element {
	col := ui.Column(c).FillWidth().Background(t.Surface).Radius(16)
	if t.Dark {
		col.Border(1, t.Border)
	}
	return col
}

// minimalSearchBar 绘制无边框软底胶囊搜索条，消除生硬焦点外框。
func minimalSearchBar(c *ui.Context, key string, query *string, placeholder, label string, t *ui.Theme) bool {
	changed := false
	wrap := ui.Row(c.Key(key)).FillWidth().Height(40).PaddingX(14).Radius(20).
		Background(t.Surface).AlignItems(ui.Center).Gap(8)
	if t.Dark {
		wrap.Border(1, t.Border)
	}
	wrap.Children(func() {
		ui.Icon(c, iconSearch).FontSize(15).TextColor(t.TextMuted).Shrink(0)
		in := ui.TextInputBase(c.Key(key+"-in"), query).
			Placeholder(placeholder).Label(label).
			FocusRing(false).FontSize(14).TextColor(t.Text).
			Grow(1).MinWidth(40)
		if in.Changed() {
			changed = true
		}
		if *query != "" {
			clr := ui.Box(c.Key(key+"-clr")).Size(20, 20).Radius(10).
				Background(t.TextMuted.Alpha(0.22)).Center().Shrink(0).
				Role(ui.RoleButton).Label("清空" + label)
			clr.Children(func() {
				ui.Icon(c, iconClose).FontSize(11).TextColor(t.Text)
			})
			if clr.Clicked() {
				*query = ""
				changed = true
			}
		}
	})
	return changed
}

// minimalTextInput 绘制无边框软底表单输入框（无刺眼蓝色聚焦边框），返回内置输入控件。
func minimalTextInput(c *ui.Context, key string, value *string, placeholder, label string, t *ui.Theme) ui.Element {
	var in ui.Element
	box := ui.Row(c.Key(key+"-wrap")).FillWidth().Height(44).PaddingX(14).Radius(12).
		Background(t.Surface).AlignItems(ui.Center)
	if t.Dark {
		box.Border(1, t.Border)
	}
	box.Children(func() {
		in = ui.TextInputBase(c.Key(key), value).
			Placeholder(placeholder).Label(label).
			FocusRing(false).FontSize(14.5).TextColor(t.Text).
			Grow(1).MinWidth(40)
	})
	return in
}

// minimalTextArea 绘制无边框软底多行文本输入区，返回内置输入控件。
func minimalTextArea(c *ui.Context, key string, value *string, placeholder, label string, height float32, t *ui.Theme) ui.Element {
	var in ui.Element
	box := ui.Box(c.Key(key+"-wrap")).FillWidth().Height(height).Padding(12, 14).Radius(12).
		Background(t.Surface)
	if t.Dark {
		box.Border(1, t.Border)
	}
	box.Children(func() {
		in = ui.TextAreaBase(c.Key(key), value).
			Placeholder(placeholder).Label(label).
			FocusRing(false).FontSize(14.5).TextColor(t.Text).Fill()
	})
	return in
}

// filterPill 绘制精简轻盈的胶囊筛选项。
func filterPill(c *ui.Context, key any, label string, active bool, t *ui.Theme) ui.Element {
	bg := t.Surface
	fg := secondaryTextColor(t)
	weight := 500
	if active {
		bg = t.Accent
		fg = t.AccentText
		weight = 600
	}
	el := ui.Row(c.Key(key)).Height(30).PaddingX(13).Radius(15).Background(bg).
		AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
		Role(ui.RoleButton).Label(label).Checked(active).
		Children(func() {
			ui.Text(c, label).FontSize(12.5).FontWeight(weight).TextColor(fg).SingleLine()
		})
	return el
}

// monochromeButton 绘制现代黑白单色胶囊按钮（primary 为实心黑白主按钮，非 primary 为柔和灰底按钮）。
func monochromeButton(c *ui.Context, key any, label string, primary bool, t *ui.Theme) ui.Element {
	bg := t.Surface
	fg := t.Text
	if primary {
		bg = t.Accent
		fg = t.AccentText
	}
	btn := ui.Row(c.Key(key)).Height(44).PaddingX(18).Radius(22).Background(bg).
		AlignItems(ui.Center).Justify(ui.Center).
		Role(ui.RoleButton).Label(label)
	if !primary && t.Dark {
		btn.Border(1, t.Border)
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(14).FontWeight(600).TextColor(fg).SingleLine()
	})
	return btn
}

// underlineTabStrip 绘制杂志社论风极简下划线标签栏（底端 1px 细线 + 2px 单色激活条）。
func underlineTabStrip(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	changed := false
	ui.Column(c.Key(key)).FillWidth().Role(ui.RoleTabList).Children(func() {
		ui.Row(c).FillWidth().Gap(4).Children(func() {
			for i, lbl := range labels {
				active := *selected == i
				fg := t.TextMuted
				weight := 500
				if active {
					fg = t.Text
					weight = 600
				}
				tab := ui.Column(c.Key(fmt.Sprintf("%s-%d", key, i))).Grow(1).Basis(0).Height(38).
					AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(6, 4, 0, 4).
					Role(ui.RoleTab).Label(lbl).Checked(active)
				tab.Children(func() {
					ui.Text(c, lbl).FontSize(13.5).FontWeight(weight).TextColor(fg).SingleLine()
					if active {
						ui.Box(c).Width(24).Height(2).Radius(1).Background(t.Text)
					} else {
						ui.Box(c).Width(24).Height(2).Background(ui.Transparent)
					}
				})
				if tab.Clicked() && !active {
					*selected = i
					changed = true
				}
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(t.Border)
	})
	return changed
}

// minimalSegmentedControl 绘制扁平分段选择器（用于「设置 -> 外观主题」）。
func minimalSegmentedControl(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	changed := false
	trackBg := ui.Hex("#EBEBF0")
	thumbBg := ui.Hex("#FFFFFF")
	if t.Dark {
		trackBg = ui.Hex("#09090B")
		thumbBg = ui.Hex("#27272E")
	}
	strip := ui.Row(c.Key(key)).FillWidth().Padding(3).Radius(11).Background(trackBg).Gap(2).Role(ui.RoleTabList)
	strip.Children(func() {
		for i, lbl := range labels {
			active := *selected == i
			item := ui.Row(c.Key(fmt.Sprintf("%s-%d", key, i))).Grow(1).Basis(0).Height(32).Radius(9).
				AlignItems(ui.Center).Justify(ui.Center).PaddingX(6).
				Role(ui.RoleTab).Label(lbl).Checked(active)
			fg := t.TextMuted
			weight := 500
			if active {
				item.Background(thumbBg)
				if !t.Dark {
					item.Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.08))
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

// minimalBottomBar 绘制极简纯白/炭黑底部导航栏（无厚重胶囊指示块，1px 极细顶线 + 清晰单色线性图标）。
func minimalBottomBar(c *ui.Context, selected *int, t *ui.Theme) bool {
	_, _, bottom, _ := c.SafeArea()
	if bottom > keyboardInsetThreshold {
		return false
	}
	type navDef struct {
		label string
		icon  *ui.SVG
	}
	items := []navDef{
		{label: "图鉴", icon: iconNavGuide},
		{label: "探索", icon: iconNavExplore},
		{label: "日志", icon: iconNavJournal},
		{label: "设置", icon: iconNavSettings},
	}

	changed := false
	ui.Row(c.Key("min-bottom-bar")).FillWidth().Background(t.Background).
		Padding(0, 8, bottom, 8).Margin(0, 0, -bottom, 0).Shrink(0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).
		Role(ui.RoleTabList).Children(func() {
		for i, it := range items {
			active := *selected == i
			fg := t.TextMuted
			weight := 400
			if active {
				fg = t.Text
				weight = 600
			}
			tab := ui.Column(c.Key(fmt.Sprintf("nav-%d", i))).Grow(1).Basis(0).Height(56).
				AlignItems(ui.Center).Justify(ui.Center).Gap(3).
				Role(ui.RoleTab).Label(it.label).Checked(active)
			tab.Children(func() {
				ui.Icon(c, it.icon).FontSize(20).TextColor(fg)
				ui.Text(c, it.label).FontSize(10.5).FontWeight(weight).TextColor(fg).NoWrap()
			})
			if tab.Clicked() && !active {
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
		ui.Icon(c, svg).FontSize(size * 0.46).TextColor(fg)
	})
}

func glassCircleBtn(c *ui.Context, key any, label string, svg *ui.SVG, size float32) ui.Element {
	return circleIconBtn(c, key, label, svg, ui.RGBA(0, 0, 0, 0.38), ui.RGB(255, 255, 255), size)
}

func statusPill(c *ui.Context, text string, bg, fg ui.Color) {
	ui.Row(c).Padding(3, 9).Radius(999).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(11.5).FontWeight(500).TextColor(fg).SingleLine()
	})
}

func minimalDivider(c *ui.Context, t *ui.Theme) {
	ui.Box(c).FillWidth().Height(1).Background(t.Border)
}

func fallbackText(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
