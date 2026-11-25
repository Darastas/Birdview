package appui

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

func shimmerPhase() float32 {
	const periodNs = int64(1350 * time.Millisecond)
	return float32(time.Now().UnixNano()%periodNs) / float32(periodNs)
}

func birdHopPhase() float32 {
	const periodNs = int64(1050 * time.Millisecond)
	return float32(time.Now().UnixNano()%periodNs) / float32(periodNs)
}

// drawShimmerRect 绘制 108° 斜向流畅冷灰渐变扫光骨架底板。
func drawShimmerRect(p *ui.Painter, r ui.Rect, radius float32, phase float32, dark bool) {
	base := ui.Hex("#E3E8F0")
	peak := ui.Hex("#F7FAFF")
	if dark {
		base = ui.Hex("#1A202D")
		peak = ui.Hex("#293347")
	}

	center := -0.30 + phase*1.60
	hi := func(at float32) float32 {
		d := float32(math.Abs(float64(at - center)))
		const halfW = float32(0.32)
		if d >= halfW {
			return 0
		}
		u := 1 - d/halfW
		return u * u * (3 - 2*u)
	}

	var stops [7]ui.GradientStop
	positions := [7]float32{0, 0.16, 0.33, 0.50, 0.67, 0.84, 1}
	for i, at := range positions {
		stops[i] = ui.GradientStop{Color: base.Mix(peak, hi(at)), At: at}
	}

	p.FillGradient(r, ui.LinearGradient{
		Angle: 108,
		Stops: stops[:],
	}, radius)
}

// skeletonShimmerBox 构建带斜向灰色流光渐变动画的占位骨架矩形。
func skeletonShimmerBox(c *ui.Context, radius float32, t *ui.Theme) ui.Element {
	c.AnimationFrame()
	phase := shimmerPhase()
	return ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
		drawShimmerRect(p, r, radius, phase, t.Dark)
	})
}

// skeletonBirdCard 构建带灰色渐变流光底板与活泼弹性跳跃小鸟图标的加载占位卡片。
func skeletonBirdCard(c *ui.Context, radius float32, iconSize float32, t *ui.Theme) ui.Element {
	c.AnimationFrame()
	phase := shimmerPhase()
	hop := birdHopPhase()
	bounce := float32(math.Abs(math.Sin(float64(hop) * math.Pi)))
	liftY := -bounce * 9
	shadowW := max(float32(16), iconSize*0.64) - bounce*7
	shadowAlpha := float32(0.16) - bounce*0.08
	if t.Dark {
		shadowAlpha = float32(0.36) - bounce*0.14
	}
	shadowCol := ui.RGBA(15, 23, 42, shadowAlpha)
	iconCol := t.TextMuted.Alpha(0.55)
	if t.Dark {
		iconCol = t.TextMuted.Alpha(0.65)
	}

	return ui.Box(c).Fill().Center().Draw(func(p *ui.Painter, r ui.Rect) {
		drawShimmerRect(p, r, radius, phase, t.Dark)
	}).Children(func() {
		ui.Column(c).Center().Gap(3).Children(func() {
			ui.Box(c).Translate(0, liftY).Children(func() {
				ui.Icon(c, iconBird).FontSize(iconSize).TextColor(iconCol)
			})
			ui.Box(c).Size(shadowW, 4).Radius(2).Background(shadowCol)
		})
	})
}

// skeletonAudioRows 为物种详情页鸟鸣参考渲染渐变扫光骨架加载行。
func skeletonAudioRows(c *ui.Context, count int, t *ui.Theme) {
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		for range count {
			ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
				skeletonShimmerBox(c, 19, t).Size(38, 38).Shrink(0)
				ui.Column(c).Grow(1).Gap(7).Children(func() {
					skeletonShimmerBox(c, 6, t).WidthPercent(62).Height(13)
					skeletonShimmerBox(c, 5, t).WidthPercent(44).Height(11)
				})
			})
		}
	})
}

// skeletonTextLines 为物种详情页物种简介渲染多行扫光文本骨架。
func skeletonTextLines(c *ui.Context, t *ui.Theme) {
	widths := []float32{96, 88, 92, 64}
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		for _, w := range widths {
			skeletonShimmerBox(c, 6, t).WidthPercent(w).Height(13)
		}
	})
}
