package appui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const keyboardInsetThreshold = 120

// dockToggleAnim 是底部导航栏自动收起/展开以及联动悬浮控件的统一弹簧动画曲线。
var dockToggleAnim = ui.Animation{
	Spring: ui.SpringOptions{
		Duration:  300 * time.Millisecond,
		Bounce:    0,
		Precision: 0.5,
	},
}

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
	t.Scrollbar = ui.Transparent
	t.ScrollbarStyle = ui.ScrollbarStyle{
		Track:  ui.Transparent,
		Thumb:  ui.Transparent,
		Hover:  ui.Transparent,
		Active: ui.Transparent,
	}

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
	ui.SetImmersive(c, true)
	c.Root().Background(t.Background)
	return &t
}

// immersiveTopPad 返回沉浸式全面屏模式下避让顶部状态栏的上边距。
func immersiveTopPad(c *ui.Context, base float32) float32 {
	if top, _, _, _ := c.SafeArea(); top > 0 {
		return top + 6
	}
	return base
}

// dockClearancePad 返回列表或悬浮按钮避让底部悬浮导航底座的底边距。
func dockClearancePad(c *ui.Context, extra float32) float32 {
	_, _, bottom, _ := c.SafeArea()
	return max(float32(10), bottom+4) + 62 + extra
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
				triggerHapticTap()
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

// tactileSegmentedControl 绘制全宽圆角凹槽 + 立体浮雕滑块分段器（曲率与搜索框保持一致）。
func tactileSegmentedControl(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	strip := tactileRecessedStyle(ui.Row(c.Key(key)).FillWidth().Height(40).Padding(3).Gap(2).Role(ui.RoleTabList), t, 20)
	return renderSegmentItems(c, strip, key, selected, 34, 17, 6, true, t, labels...)
}

// tactileCompactSegment 绘制自适应宽度的紧凑阴刻分段切换器（适配已解锁开关与排序切换）。
func tactileCompactSegment(c *ui.Context, key string, selected *int, t *ui.Theme, labels ...string) bool {
	strip := tactileRecessedStyle(ui.Row(c.Key(key)).Height(34).Padding(3).Gap(2).Shrink(0).Role(ui.RoleTabList), t, 17)
	return renderSegmentItems(c, strip, key, selected, 28, 14, 12, false, t, labels...)
}

func renderSegmentItems(c *ui.Context, strip ui.Element, key string, selected *int, itemH, itemR, padX float32, grow bool, t *ui.Theme, labels ...string) bool {
	changed := false
	strip.Children(func() {
		for i, lbl := range labels {
			active := *selected == i
			item := ui.Row(c.Key(fmt.Sprintf("%s-%d", key, i))).Height(itemH).Radius(itemR).
				AlignItems(ui.Center).Justify(ui.Center).PaddingX(padX).
				Role(ui.RoleTab).Label(lbl).Checked(active)
			if grow {
				item.Grow(1).Basis(0)
			} else {
				item.Shrink(0)
			}
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
				triggerHapticTap()
			}
		}
	})
	return changed
}

// floatingGlassDock 绘制悬浮在内容之上的 5 标签光学液态玻璃底部导航栏（指南居中，支持随滚动同曲线收起/展开）。
func floatingGlassDock(c *ui.Context, selected *int, t *ui.Theme, obscured, hidden bool) (bool, float32) {
	_, _, bottom, _ := c.SafeArea()
	if bottom > keyboardInsetThreshold {
		return false, 1
	}
	type navDef struct {
		label string
		icon  *ui.SVG
	}
	items := []navDef{
		{label: "图鉴", icon: iconNavGuide},
		{label: "探索", icon: iconNavExplore},
		{label: "指南", icon: iconNavSeasonal},
		{label: "日志", icon: iconNavJournal},
		{label: "设置", icon: iconNavSettings},
	}

	dockBottom := max(float32(10), bottom+4)
	hideDist := float32(62) + dockBottom + 28
	targetShift := float32(0)
	if hidden {
		targetShift = hideDist
	}

	changed := false
	dock := ui.Row(c.Key("min-bottom-bar")).Absolute().Left(14).Right(14).Bottom(dockBottom).
		Height(62).Padding(5, 5).Radius(31).
		Role(ui.RoleTabList)
	shift := dock.AnimateTo("dock-slide", targetShift, dockToggleAnim)
	hideFrac := max(float32(0), min(float32(1), shift/max(hideDist, 1)))
	dock.Translate(0, shift)
	if hidden && shift > 24 {
		dock.PassThrough()
	}
	if obscured || hideFrac > 0.05 {
		if t.Dark {
			dock.Background(ui.Hex("#161B26")).Border(1, ui.RGBA(255, 255, 255, 0.10))
		} else {
			dock.Background(ui.Hex("#FFFFFF")).Border(1, ui.RGBA(15, 23, 42, 0.08))
		}
	} else {
		dock.Material(floatingBarGlass(t.Dark))
	}
	if t.Dark {
		dock.
			Shadow(0, 14, 34, -4, ui.RGBA(0, 0, 0, 0.58)).
			Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.35))
	} else {
		dock.
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
				ui.Icon(c, it.icon).FontSize(19).TextColor(fg)
				ui.Text(c, it.label).FontSize(10.5).FontWeight(weight).TextColor(fg).NoWrap()
			})
			if tab.Clicked() && !active {
				*selected = i
				changed = true
			}
		}
	})
	return changed, hideFrac
}

// applyPhotoOverlayGlass 在静态展示时应用光学液态玻璃，路由滑动期间切换为轻量磨砂半透明以保证 120fps 零卡顿。
func applyPhotoOverlayGlass(el ui.Element, transitioning, interactive bool) ui.Element {
	if transitioning {
		return el.Background(ui.RGBA(18, 24, 36, 0.56)).Border(1, ui.RGBA(255, 255, 255, 0.24))
	}
	return el.Material(photoOverlayGlass(interactive))
}

func circleIconBtn(c *ui.Context, key any, label string, svg *ui.SVG, bg, fg ui.Color, size float32) ui.Element {
	return ui.Box(c.Key(key)).Size(size, size).Radius(size / 2).Background(bg).Center().Shrink(0).
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

// liquidGlassCircleBtn 绘制覆盖于全出血照片上的自研光学液态玻璃圆形悬浮按钮。
func liquidGlassCircleBtn(c *ui.Context, key any, label string, svg *ui.SVG, size float32) ui.Element {
	return liquidGlassCircleBtnFast(c, key, label, svg, size, false)
}

// liquidGlassCircleBtnFast 支持在页面转场期间自动避开离屏模糊回读。
func liquidGlassCircleBtnFast(c *ui.Context, key any, label string, svg *ui.SVG, size float32, transitioning bool) ui.Element {
	return applyPhotoOverlayGlass(
		ui.Box(c.Key(key)).Size(size, size).Radius(size/2),
		transitioning, true,
	).
		Shadow(0, 4, 14, 0, ui.RGBA(0, 0, 0, 0.28)).
		Center().Shrink(0).
		Role(ui.RoleButton).Label(label).Children(func() {
		ui.Icon(c, svg).FontSize(size * 0.45).TextColor(ui.RGB(255, 255, 255))
	})
}

// statusPill 绘制固定高度与光学居中的轻量信息徽章。
func statusPill(c *ui.Context, text string, bg, fg ui.Color) {
	ui.Row(c).Height(24).PaddingX(10).Radius(12).Background(bg).
		AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(11).FontWeight(500).TextColor(fg).SingleLine()
	})
}

type metaItem struct {
	Text   string
	Color  ui.Color // 零值默认 secondaryTextColor(t)
	Weight int      // 零值默认 400
	Tnum   bool
}

// metaAnnotationRow 以无左内边距的纯排版行展示标注类元信息，使首字与上方标题、正文严格对齐在同一垂直视觉轴线（Optical Alignment）。
func metaAnnotationRow(c *ui.Context, t *ui.Theme, items ...metaItem) {
	var valid []metaItem
	for _, it := range items {
		if strings.TrimSpace(it.Text) != "" {
			valid = append(valid, it)
		}
	}
	if len(valid) == 0 {
		return
	}
	ui.Row(c).FillWidth().Gap(8).Wrap().AlignItems(ui.Center).Padding(2, 0, 0, 0).Children(func() {
		for i, it := range valid {
			col := it.Color
			if col == (ui.Color{}) {
				col = secondaryTextColor(t)
			}
			w := it.Weight
			if w == 0 {
				w = 400
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				el := ui.Text(c, it.Text).FontSize(13).FontWeight(w).TextColor(col)
				if it.Tnum {
					el.FontFeatures("tnum")
				}
				if i < len(valid)-1 {
					ui.Box(c).Width(1).Height(10).Radius(0.5).Background(t.TextMuted.Alpha(0.28)).Shrink(0)
				}
			})
		}
	})
}

// externalLinkChip 绘制带品牌图标与微立体触感的外部资料跳转胶囊按钮。
func externalLinkChip(c *ui.Context, key, label string, icon *ui.SVG, url string, t *ui.Theme) {
	chip := applySculptedSecondaryPillStyle(
		ui.Row(c.Key(key)).Height(34).PaddingX(12).Radius(17).
			Gap(6).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
			Role(ui.RoleButton).Label(label),
		t,
	)
	chip.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(opticalAccentColor(t))
		ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
		ui.Icon(c, iconExternalLink).FontSize(12).TextColor(t.TextMuted)
	})
	if chip.Clicked() {
		triggerHapticTap()
		_ = mygo.Shell.OpenExternal(url)
	}
}

// scrollTopFadeScrim 在有图列表离开首行后渐显顶部柔顺渐隐过渡遮罩，静止在首行时保持完全通透。
func scrollTopFadeScrim(c *ui.Context, t *ui.Theme, scrollY float32) {
	const startY, fullY float32 = 8, 28
	if scrollY <= startY {
		return
	}
	k := min(float32(1), (scrollY-startY)/(fullY-startY))
	topCol := ui.Hex("#EEF2F7")
	if t.Dark {
		topCol = ui.Hex("#0E121B")
	}
	ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(20).PassThrough().
		LinearGradient(ui.LinearGradient{
			Angle: 180,
			Stops: []ui.GradientStop{
				{Color: topCol.Alpha(0.98 * k), At: 0},
				{Color: topCol.Alpha(0.68 * k), At: 0.48},
				{Color: topCol.Alpha(0), At: 1.0},
			},
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
