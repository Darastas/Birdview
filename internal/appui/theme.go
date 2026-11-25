package appui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const keyboardInsetThreshold = 120

var (
	opticalBlueLight = ui.Hex("#2563EB")
	opticalCyanLight = ui.Hex("#0EA5E9")
	opticalBlueDark  = ui.Hex("#38BDF8")
	opticalCyanDark  = ui.Hex("#60A5FA")
)

func opticalAccentColors(t *ui.Theme) (ui.Color, ui.Color) {
	if t.Dark {
		return opticalBlueDark, opticalCyanDark
	}
	return opticalBlueLight, opticalCyanLight
}

func opticalAccentColor(t *ui.Theme) ui.Color {
	a, _ := opticalAccentColors(t)
	return a
}

// applyAppTheme 配置「触感空间极简主义」主题令牌：丝柔冷珠光 / 深邃曜石画布，配合 Liquid Glass 与顶部高光质感。
func applyAppTheme(c *ui.Context) *ui.Theme {
	base := ui.LightTheme()
	accent := ui.Hex("#182030")
	if c.Theme().Dark {
		base = ui.DarkTheme()
		accent = ui.Hex("#F8FAFC")
	}
	t := *base.WithPreferences(c.Preferences()).WithAccent(accent)
	t.Radius = 16
	t.Spacing = 6
	t.FontSize = 15
	t.Font = platformFontFamily()

	if c.Theme().Dark {
		t.Background = ui.Hex("#0B0E16")
		t.Surface = ui.Hex("#171B25")
		t.SurfaceHover = ui.Hex("#202634")
		t.SurfacePressed = ui.Hex("#283042")
		t.Border = ui.RGBA(255, 255, 255, 0.09)
		t.Text = ui.Hex("#F8FAFC")
		t.TextMuted = ui.Hex("#6C778A")
		t.AccentText = ui.Hex("#0B0E16")
		t.Success = ui.Hex("#22C55E")
		t.Warning = ui.Hex("#FBBF24")
		t.Danger = ui.Hex("#F87171")
	} else {
		t.Background = ui.Hex("#F3F5F9")
		t.Surface = ui.Hex("#FFFFFF")
		t.SurfaceHover = ui.Hex("#F5F7FB")
		t.SurfacePressed = ui.Hex("#EBEEF5")
		t.Border = ui.RGBA(15, 23, 42, 0.08)
		t.Text = ui.Hex("#0F172A")
		t.TextMuted = ui.Hex("#7C8698")
		t.AccentText = ui.RGB(255, 255, 255)
		t.Success = ui.Hex("#16A34A")
		t.Warning = ui.Hex("#D97706")
		t.Danger = ui.Hex("#EF4444")
	}
	c.SetTheme(&t)
	return &t
}

// platformFontFamily 返回兼顾中英文几何比例与高平滑度渲染的跨平台现代无衬线字体栈。
func platformFontFamily() string {
	switch runtime.GOOS {
	case "windows":
		return "HarmonyOS Sans SC, MiSans VF, Noto Sans SC, PingFang SC, Microsoft YaHei UI, Segoe UI, sans-serif"
	case "ios", "darwin":
		return "-apple-system, PingFang SC, sans-serif"
	case "android":
		return "sans-serif"
	default:
		return "Noto Sans CJK SC, Source Han Sans SC, PingFang SC, sans-serif"
	}
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

func applyCanvasBackground(el ui.Element, t *ui.Theme) ui.Element {
	if t.Dark {
		return el.LinearGradient(ui.LinearGradient{
			From:  ui.Hex("#0E121B"),
			To:    ui.Hex("#07090E"),
			Angle: 180,
		})
	}
	return el.LinearGradient(ui.LinearGradient{
		From:  ui.Hex("#EEF2F7"),
		To:    ui.Hex("#F7F8FB"),
		Angle: 180,
	})
}

func secondaryTextColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#A3B0C2")
	}
	return ui.Hex("#475569")
}

func softPillColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#1F2533")
	}
	return ui.Hex("#E7ECF3")
}

func subtleHairlineColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.07)
	}
	return ui.RGBA(15, 23, 42, 0.07)
}

func applyCardRimAndShadow(el ui.Element, radius float32, t *ui.Theme) ui.Element {
	el.Radius(radius).BorderWidth(1)
	if t.Dark {
		return el.
			LinearGradient(ui.LinearGradient{From: ui.Hex("#1B202C"), To: ui.Hex("#131720"), Angle: 180}).
			BorderLinearGradient(ui.LinearGradient{From: ui.RGBA(255, 255, 255, 0.14), To: ui.RGBA(255, 255, 255, 0.03), Angle: 180}).
			InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.07)).
			Shadow(0, 10, 26, -4, ui.RGBA(0, 0, 0, 0.52)).
			Shadow(0, 2, 4, 0, ui.RGBA(0, 0, 0, 0.32))
	}
	return el.
		LinearGradient(ui.LinearGradient{From: ui.Hex("#FFFFFF"), To: ui.Hex("#FAFBFD"), Angle: 180}).
		BorderLinearGradient(ui.LinearGradient{From: ui.RGBA(255, 255, 255, 0.95), To: ui.RGBA(15, 23, 42, 0.08), Angle: 180}).
		Shadow(0, 1, 2, 0, ui.RGBA(15, 23, 42, 0.06)).
		Shadow(0, 8, 22, -4, ui.RGBA(15, 23, 42, 0.07))
}

// tactileCard 构建带顶部高光描边与双层阴影纵深感的实体卡片容器。
func tactileCard(c *ui.Context, t *ui.Theme) ui.Element {
	return applyCardRimAndShadow(ui.Column(c).FillWidth(), 18, t)
}

// tactileRecessedStyle 为搜索框、输入框与分段槽赋予细腻内凹阴刻质感。
func tactileRecessedStyle(el ui.Element, t *ui.Theme, radius float32) ui.Element {
	if t.Dark {
		return el.Background(ui.Hex("#090C12")).Radius(radius).
			Border(1, ui.RGBA(255, 255, 255, 0.07)).
			InsetShadow(0, 2, 4, 0, ui.RGBA(0, 0, 0, 0.55))
	}
	return el.Background(ui.Hex("#E7ECF3")).Radius(radius).
		Border(1, ui.RGBA(15, 23, 42, 0.06)).
		InsetShadow(0, 1.5, 3, 0, ui.RGBA(15, 23, 42, 0.08))
}

func applySculptedPrimaryStyle(el ui.Element, t *ui.Theme) ui.Element {
	if t.Dark {
		return el.
			LinearGradient(ui.LinearGradient{From: ui.Hex("#FFFFFF"), To: ui.Hex("#E2E8F0"), Angle: 180, Oklab: true}).
			InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.9)).
			Shadow(0, 6, 18, -3, ui.RGBA(0, 0, 0, 0.5))
	}
	return el.
		LinearGradient(ui.LinearGradient{From: ui.Hex("#2B3548"), To: ui.Hex("#111827"), Angle: 180, Oklab: true}).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.22)).
		Shadow(0, 6, 16, -2, ui.RGBA(15, 23, 42, 0.28)).
		Shadow(0, 1, 2, 0, ui.RGBA(15, 23, 42, 0.16))
}

func applySculptedSecondaryPillStyle(el ui.Element, t *ui.Theme) ui.Element {
	if t.Dark {
		return el.Background(ui.Hex("#1A1F2B")).
			Border(1, ui.RGBA(255, 255, 255, 0.09)).
			InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.06)).
			Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.28))
	}
	return el.Background(ui.Hex("#FFFFFF")).
		Border(1, ui.RGBA(15, 23, 42, 0.07)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.9)).
		Shadow(0, 1, 3, 0, ui.RGBA(15, 23, 42, 0.06))
}

// minimalSearchBar 绘制内凹软底搜索条（无生硬焦点描边）。
func minimalSearchBar(c *ui.Context, key string, query *string, placeholder, label string, t *ui.Theme) bool {
	changed := false
	wrap := tactileRecessedStyle(ui.Row(c.Key(key)).FillWidth().Height(40).PaddingX(14).AlignItems(ui.Center).Gap(8), t, 20)
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
				Background(t.TextMuted.Alpha(0.24)).Center().Shrink(0).
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

// minimalTextInput 绘制内凹阴刻单行输入框，消除生硬蓝色框线。
func minimalTextInput(c *ui.Context, key string, value *string, placeholder, label string, t *ui.Theme) ui.Element {
	var in ui.Element
	box := tactileRecessedStyle(ui.Row(c.Key(key+"-wrap")).FillWidth().Height(44).PaddingX(14).AlignItems(ui.Center), t, 12)
	box.Children(func() {
		in = ui.TextInputBase(c.Key(key), value).
			Placeholder(placeholder).Label(label).
			FocusRing(false).FontSize(14).TextColor(t.Text).
			Grow(1).MinWidth(40)
	})
	return in
}

// minimalTextArea 绘制内凹阴刻多行文本输入区。
func minimalTextArea(c *ui.Context, key string, value *string, placeholder, label string, height float32, t *ui.Theme) ui.Element {
	var in ui.Element
	box := tactileRecessedStyle(ui.Box(c.Key(key+"-wrap")).FillWidth().Height(height).Padding(12, 14), t, 12)
	box.Children(func() {
		in = ui.TextAreaBase(c.Key(key), value).
			Placeholder(placeholder).Label(label).
			FocusRing(false).FontSize(14).TextColor(t.Text).Fill()
	})
	return in
}

// filterPill 绘制精雕微阴影胶囊筛选按钮。
func filterPill(c *ui.Context, key any, label string, active bool, t *ui.Theme) ui.Element {
	fg := secondaryTextColor(t)
	weight := 400
	el := ui.Row(c.Key(key)).Height(32).PaddingX(14).Radius(16).
		AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
		Role(ui.RoleButton).Label(label).Checked(active)
	if active {
		fg = t.AccentText
		weight = 500
		applySculptedPrimaryStyle(el, t)
	} else {
		applySculptedSecondaryPillStyle(el, t)
	}
	el.Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(weight).TextColor(fg).SingleLine()
	})
	return el
}

// tactilePrimaryBtn 绘制立体雕塑高光主行动按钮。
func tactilePrimaryBtn(c *ui.Context, key any, label string, t *ui.Theme) ui.Element {
	btn := applySculptedPrimaryStyle(
		ui.Row(c.Key(key)).Height(46).PaddingX(20).Radius(23).
			AlignItems(ui.Center).Justify(ui.Center).
			Role(ui.RoleButton).Label(label),
		t,
	)
	btn.Children(func() {
		ui.Text(c, label).FontSize(15).FontWeight(500).TextColor(t.AccentText).SingleLine()
	})
	return btn
}

// monochromeButton 绘制带实体光影质感的胶囊按钮。
func monochromeButton(c *ui.Context, key any, label string, primary bool, t *ui.Theme) ui.Element {
	if primary {
		return tactilePrimaryBtn(c, key, label, t)
	}
	btn := applySculptedSecondaryPillStyle(
		ui.Row(c.Key(key)).Height(44).PaddingX(18).Radius(22).
			AlignItems(ui.Center).Justify(ui.Center).
			Role(ui.RoleButton).Label(label),
		t,
	)
	btn.Children(func() {
		ui.Text(c, label).FontSize(14).FontWeight(500).TextColor(t.Text).SingleLine()
	})
	return btn
}

// tactileSegmentedControl 绘制精密凹槽 + 立体浮雕滑块分段器（适配「探索」子分页与「设置 -> 外观主题」）。
func tactileSegmentedControl(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	changed := false
	strip := tactileRecessedStyle(ui.Row(c.Key(key)).FillWidth().Padding(3).Gap(2).Role(ui.RoleTabList), t, 13)
	strip.Children(func() {
		for i, lbl := range labels {
			active := *selected == i
			item := ui.Row(c.Key(fmt.Sprintf("%s-%d", key, i))).Grow(1).Basis(0).Height(34).Radius(10).
				AlignItems(ui.Center).Justify(ui.Center).PaddingX(6).
				Role(ui.RoleTab).Label(lbl).Checked(active)
			fg := t.TextMuted
			weight := 400
			if active {
				if t.Dark {
					item.
						LinearGradient(ui.LinearGradient{From: ui.Hex("#283042"), To: ui.Hex("#1E2433"), Angle: 180}).
						Border(1, ui.RGBA(255, 255, 255, 0.14)).
						InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.14)).
						Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.4))
				} else {
					item.
						LinearGradient(ui.LinearGradient{From: ui.Hex("#FFFFFF"), To: ui.Hex("#F7F9FC"), Angle: 180}).
						Border(1, ui.RGBA(15, 23, 42, 0.05)).
						InsetShadow(0, 1, 0, 0, ui.RGB(255, 255, 255)).
						Shadow(0, 1, 3, 0, ui.RGBA(15, 23, 42, 0.10)).
						Shadow(0, 3, 8, -2, ui.RGBA(15, 23, 42, 0.08))
				}
				fg = t.Text
				weight = 500
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

// floatingGlassDock 绘制悬浮在内容之上的真正 GPU Liquid Glass 胶囊底部导航栏（含底部卷动边缘渐变）。
func floatingGlassDock(c *ui.Context, selected *int, t *ui.Theme) bool {
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

	// 底部柔化卷动渐隐边缘（让文字滑入悬空底座前自然淡出）
	ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(92 + bottom).PassThrough().
		Material(glass.ScrollEdge{Bottom: true})

	dockBottom := max(float32(12), bottom+6)
	changed := false
	dock := ui.Row(c.Key("min-bottom-bar")).Absolute().Left(16).Right(16).Bottom(dockBottom).
		Height(62).Padding(5, 6).Radius(31).
		Material(glass.Glass{Style: glass.Regular}).
		Role(ui.RoleTabList)
	if t.Dark {
		dock.Border(1, ui.RGBA(255, 255, 255, 0.16)).
			Shadow(0, 14, 34, -4, ui.RGBA(0, 0, 0, 0.58)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.35))
	} else {
		dock.Border(1, ui.RGBA(255, 255, 255, 0.72)).
			Shadow(0, 12, 30, -4, ui.RGBA(15, 23, 42, 0.16)).
			Shadow(0, 2, 6, 0, ui.RGBA(15, 23, 42, 0.08))
	}

	dock.Children(func() {
		for i, it := range items {
			active := *selected == i
			fg := secondaryTextColor(t)
			weight := 400
			tab := ui.Column(c.Key(fmt.Sprintf("nav-%d", i))).Grow(1).Basis(0).Height(52).Radius(26).
				AlignItems(ui.Center).Justify(ui.Center).Gap(2).
				Role(ui.RoleTab).Label(it.label).Checked(active)
			if active {
				fg = t.Text
				weight = 500
				if t.Dark {
					tab.Background(ui.RGBA(255, 255, 255, 0.14)).
						InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.18))
				} else {
					tab.Background(ui.RGBA(255, 255, 255, 0.85)).
						InsetShadow(0, 1, 0, 0, ui.RGB(255, 255, 255)).
						Shadow(0, 1, 4, 0, ui.RGBA(15, 23, 42, 0.09))
				}
			}
			tab.Children(func() {
				ui.Icon(c, it.icon).FontSize(20).TextColor(fg)
				ui.Text(c, it.label).FontSize(11).FontWeight(weight).TextColor(fg).NoWrap()
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

// tactileIconBtn 绘制具有边框与微立体阴影像差的控制圆钮。
func tactileIconBtn(c *ui.Context, key any, label string, svg *ui.SVG, size float32, t *ui.Theme) ui.Element {
	el := applySculptedSecondaryPillStyle(
		ui.Box(c.Key(key)).Size(size, size).Radius(size/2).Center().Shrink(0).
			Role(ui.RoleButton).Label(label),
		t,
	)
	el.Children(func() {
		ui.Icon(c, svg).FontSize(size * 0.45).TextColor(t.Text)
	})
	return el
}

// liquidGlassCircleBtn 绘制覆盖于全出血照片上的真实 Liquid Glass 圆形悬浮按钮。
func liquidGlassCircleBtn(c *ui.Context, key any, label string, svg *ui.SVG, size float32) ui.Element {
	return ui.Box(c.Key(key)).Size(size, size).Radius(size/2).
		Material(glass.Glass{Style: glass.Clear, Interactive: true, Tint: ui.RGBA(0, 0, 0, 0.22)}).
		Border(1, ui.RGBA(255, 255, 255, 0.28)).
		Shadow(0, 4, 14, 0, ui.RGBA(0, 0, 0, 0.26)).
		Center().Shrink(0).
		Role(ui.RoleButton).Label(label).Children(func() {
		ui.Icon(c, svg).FontSize(size * 0.45).TextColor(ui.RGB(255, 255, 255))
	})
}

func statusPill(c *ui.Context, text string, bg, fg ui.Color) {
	ui.Row(c).Padding(3, 9).Radius(999).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(11).FontWeight(500).TextColor(fg).SingleLine()
	})
}

func minimalDivider(c *ui.Context, t *ui.Theme) {
	ui.Box(c).FillWidth().Height(1).Background(subtleHairlineColor(t))
}

func fallbackText(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
