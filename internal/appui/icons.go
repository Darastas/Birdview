package appui

import "github.com/egoist/mygo/ui"

// 简洁单色线性 SVG 图标集（随当前文本颜色着色，适配浅色/深色主题）。
var (
	iconBack         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>`))
	iconChevronRight = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`))
	iconBook         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>`))
	iconJournal      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>`))
	iconAward        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="6"/><path d="M15.477 12.89 17 22l-5-3-5 3 1.523-9.11"/></svg>`))
	iconGlobe        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`))
	iconMapPin       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>`))
	iconSettings     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`))
	iconPlay         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="6 4 20 12 6 20 6 4"/></svg>`))
	iconPause        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>`))
	iconStop         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>`))
	iconVolume       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/></svg>`))
	iconRefresh      = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>`))
	iconPlus         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>`))
	iconMinus        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 12h14"/></svg>`))
	iconMic          = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>`))
	iconImage        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`))
	iconCamera       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>`))
	iconFilter       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>`))
	iconShare        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" x2="12" y1="2" y2="15"/></svg>`))
	iconCheck        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`))
	iconSearch       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`))
	iconExternal     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>`))
	iconTrash        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>`))
	iconClose        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`))
	iconLocate       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="2" x2="5" y1="12" y2="12"/><line x1="19" x2="22" y1="12" y2="12"/><line x1="12" x2="12" y1="2" y2="5"/><line x1="12" x2="12" y1="19" y2="22"/><circle cx="12" cy="12" r="7"/><circle cx="12" cy="12" r="3"/></svg>`))
)

// 以下为 mobile.BottomBar 与 mobile.FloatingButton 使用的矢量 Painter 图标。

func iconCoord(r ui.Rect, x, y float32) (float32, float32) {
	return r.X + x*r.W/24, r.Y + y*r.H/24
}

func paintIconGuide(p *ui.Painter, r ui.Rect, col ui.Color) {
	var path ui.Path
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	x1, y1 := iconCoord(r, 12, 6)
	x2, y2 := iconCoord(r, 12, 19)
	path.MoveTo(x1, y1).LineTo(x2, y2)

	// 左页
	lx0, ly0 := iconCoord(r, 3.5, 5)
	lx1, ly1 := iconCoord(r, 9, 4.5)
	path.MoveTo(x1, y1).QuadTo(lx1, ly1, lx0, ly0)
	lx2, ly2 := iconCoord(r, 3.5, 17.5)
	lx3, ly3 := iconCoord(r, 9, 17.5)
	path.LineTo(lx2, ly2).QuadTo(lx3, ly3, x2, y2)

	// 右页
	rx0, ry0 := iconCoord(r, 20.5, 5)
	rx1, ry1 := iconCoord(r, 15, 4.5)
	path.MoveTo(x1, y1).QuadTo(rx1, ry1, rx0, ry0)
	rx2, ry2 := iconCoord(r, 20.5, 17.5)
	rx3, ry3 := iconCoord(r, 15, 17.5)
	path.LineTo(rx2, ry2).QuadTo(rx3, ry3, x2, y2)

	p.StrokePath(&path, 1.9, col)
}

func paintIconExplore(p *ui.Painter, r ui.Rect, col ui.Color) {
	var path ui.Path
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	cx, cy := iconCoord(r, 12, 12)
	rad := 8.8 * r.W / 24
	path.Circle(cx, cy, rad)

	// 指南针菱形指针
	p1x, p1y := iconCoord(r, 15.5, 8.5)
	p2x, p2y := iconCoord(r, 13.5, 13.5)
	p3x, p3y := iconCoord(r, 8.5, 15.5)
	p4x, p4y := iconCoord(r, 10.5, 10.5)
	path.MoveTo(p1x, p1y).LineTo(p2x, p2y).LineTo(p3x, p3y).LineTo(p4x, p4y).Close()
	p.StrokePath(&path, 1.9, col)
}

func paintIconJournal(p *ui.Painter, r ui.Rect, col ui.Color) {
	var path ui.Path
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	// 记录本轮廓
	x0, y0 := iconCoord(r, 5, 4)
	x1, y1 := iconCoord(r, 19, 4)
	x2, y2 := iconCoord(r, 19, 20)
	x3, y3 := iconCoord(r, 5, 20)
	path.MoveTo(x0, y0).LineTo(x1, y1).LineTo(x2, y2).LineTo(x3, y3).Close()

	// 三行笔记横线
	l1a, l1y := iconCoord(r, 8.5, 9)
	l1b, _ := iconCoord(r, 15.5, 9)
	l2a, l2y := iconCoord(r, 8.5, 12.5)
	l2b, _ := iconCoord(r, 15.5, 12.5)
	l3a, l3y := iconCoord(r, 8.5, 16)
	l3b, _ := iconCoord(r, 13, 16)
	path.MoveTo(l1a, l1y).LineTo(l1b, l1y)
	path.MoveTo(l2a, l2y).LineTo(l2b, l2y)
	path.MoveTo(l3a, l3y).LineTo(l3b, l3y)
	p.StrokePath(&path, 1.9, col)
}

func paintIconSettings(p *ui.Painter, r ui.Rect, col ui.Color) {
	var path ui.Path
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	// 三行调音滑块式设置图标（极简清晰，在小尺寸底栏下比齿轮更锐利）
	y1 := r.Y + 6.5*r.H/24
	y2 := r.Y + 12*r.H/24
	y3 := r.Y + 17.5*r.H/24
	leftX := r.X + 4*r.W/24
	rightX := r.X + 20*r.W/24

	// 行 1：左侧旋钮在 x=9
	k1x := r.X + 9*r.W/24
	path.MoveTo(leftX, y1).LineTo(k1x-2.2*r.W/24, y1)
	path.MoveTo(k1x+2.2*r.W/24, y1).LineTo(rightX, y1)
	path.Circle(k1x, y1, 2.2*r.W/24)

	// 行 2：右侧旋钮在 x=15
	k2x := r.X + 15*r.W/24
	path.MoveTo(leftX, y2).LineTo(k2x-2.2*r.W/24, y2)
	path.MoveTo(k2x+2.2*r.W/24, y2).LineTo(rightX, y2)
	path.Circle(k2x, y2, 2.2*r.W/24)

	// 行 3：左侧旋钮在 x=10.5
	k3x := r.X + 10.5*r.W/24
	path.MoveTo(leftX, y3).LineTo(k3x-2.2*r.W/24, y3)
	path.MoveTo(k3x+2.2*r.W/24, y3).LineTo(rightX, y3)
	path.Circle(k3x, y3, 2.2*r.W/24)

	p.StrokePath(&path, 1.9, col)
}

func paintIconPlus(p *ui.Painter, r ui.Rect, col ui.Color) {
	var path ui.Path
	path.Cap(ui.RoundCap).Join(ui.RoundJoin)
	cx, cy := iconCoord(r, 12, 12)
	lx, _ := iconCoord(r, 5.5, 12)
	rx, _ := iconCoord(r, 18.5, 12)
	_, ty := iconCoord(r, 12, 5.5)
	_, by := iconCoord(r, 12, 18.5)
	path.MoveTo(lx, cy).LineTo(rx, cy)
	path.MoveTo(cx, ty).LineTo(cx, by)
	p.StrokePath(&path, 2.3, col)
}
