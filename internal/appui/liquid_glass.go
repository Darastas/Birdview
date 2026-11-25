package appui

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// crystalGlassShader 实现跨平台自研光学液态玻璃着色器：
// 包含圆角 SDF 凸透镜边沿折射、RGB 微色散、高斯背景模糊融合、内部顶部弧面拱光以及 45° 双极菲涅尔镜面高光棱边。
var crystalGlassShader = shader.MustCompile(`package main

var Radius float
var BezelWidth float
var Refract float
var Chromatic float
var Saturation float
var CrownStrength float
var RimStrength float
var RimWidth float
var Tint vec4
var ShadowTint vec4

func boxNormal(q vec2, h vec2, rad float) vec2 {
	d := abs(q) - (h - vec2(rad))
	sx := 1.0
	if q.x < 0.0 {
		sx = -1.0
	}
	sy := 1.0
	if q.y < 0.0 {
		sy = -1.0
	}
	s := vec2(sx, sy)
	if d.x > 0.0 || d.y > 0.0 {
		m := max(d, vec2(0.0))
		l := max(length(m), 0.0001)
		return s * m / l
	}
	if d.x > d.y {
		return vec2(s.x, 0.0)
	}
	return vec2(0.0, s.y)
}

func boxDistInside(q vec2, h vec2, rad float) float {
	d := abs(q) - (h - vec2(rad))
	outside := length(max(d, vec2(0.0)))
	inside := min(max(d.x, d.y), 0.0)
	return -(outside + inside - rad)
}

func Fragment(dst vec4, pos vec2) vec4 {
	sz := rectSize()
	h := sz * 0.5
	q := pos - h
	rad := min(Radius, min(h.x, h.y))
	distIn := boxDistInside(q, h, rad)

	norm := vec2(0.0)
	if distIn < BezelWidth || distIn < RimWidth*3.5 {
		norm = boxNormal(q, h, rad)
	}

	bend := 0.0
	if BezelWidth > 0.0 && distIn < BezelWidth {
		t := clamp(distIn/BezelWidth, 0.0, 1.0)
		u := 1.0 - t
		bend = u * u * u / sqrt(max(1.0-u*u*u*u*0.78, 0.12))
	}

	sp := clamp(pos-norm*(Refract*bend), vec2(0.5), max(sz-vec2(0.5), vec2(0.5)))
	var c vec3
	if Chromatic > 0.0 && bend > 0.01 {
		disp := norm * (Chromatic * bend)
		spR := clamp(sp-disp, vec2(0.5), max(sz-vec2(0.5), vec2(0.5)))
		spB := clamp(sp+disp, vec2(0.5), max(sz-vec2(0.5), vec2(0.5)))
		c = vec3(backdropSample(spR).r, backdropSample(sp).g, backdropSample(spB).b)
	} else {
		c = backdropSample(sp).rgb
	}

	lum := 0.2126*c.r + 0.7152*c.g + 0.0722*c.b
	c = clamp(mix(vec3(lum), c, Saturation), 0.0, 1.0)

	// 底部轻暗反差衬托 + 玻璃通透着色层
	c = mix(c, ShadowTint.rgb, clamp(ShadowTint.a*(1.0-smoothstep(0.0, 0.85, lum)), 0.0, 0.85))
	c = mix(c, Tint.rgb, Tint.a)

	// 内部顶部柔和弧面拱光反射
	yn := clamp(pos.y/max(sz.y, 1.0), 0.0, 1.0)
	crown := smoothstep(0.46, 0.0, yn) * smoothstep(0.0, 0.08, distIn)
	c += vec3(crown * CrownStrength)

	// 45° 双极斜向菲涅尔镜面高光棱边（左上主光弧 + 右下副反光弧，两侧柔和收敛）
	if RimStrength > 0.0 && distIn >= 0.0 && distIn < RimWidth*3.2 {
		r := distIn / max(RimWidth, 0.4)
		rimMask := exp(-r * r)
		diag := dot(norm, vec2(-0.7071, -0.7071))
		spec := pow(clamp(diag, 0.0, 1.0), 1.6)*0.92 + pow(clamp(-diag, 0.0, 1.0), 2.1)*0.48 + 0.10
		edgeLight := clamp(rimMask * RimStrength * spec, 0.0, 1.0)
		c = mix(c, vec3(1.0), edgeLight)
	}

	return vec4(clamp(c, 0.0, 1.0), 1.0)
}
`)

type crystalGlassKey struct{}
type crystalHoverKey struct{}

// crystalGlassMaterial 实现 ui.Material 与 ui.MaterialBuilder，用于绘制饱满光学折射玻璃质感。
type crystalGlassMaterial struct {
	blur          float32
	bezel         float32
	refract       float32
	chromatic     float32
	saturation    float32
	crown         float32
	rim           float32
	rimWidth      float32
	tint          ui.Color
	shadowTint    ui.Color
	fallbackColor ui.Color
	interactive   bool
	grow          float32
	hover         float32
}

// photoOverlayGlass 构建叠加在摄影大图上的清透液态玻璃控件材质（饱满折射 + 45° 菲涅尔棱边）。
func photoOverlayGlass(interactive bool) crystalGlassMaterial {
	return crystalGlassMaterial{
		blur:          10,
		bezel:         18,
		refract:       9.5,
		chromatic:     1.6,
		saturation:    1.62,
		crown:         0.16,
		rim:           0.84,
		rimWidth:      1.35,
		tint:          ui.RGBA(255, 255, 255, 0.16),
		shadowTint:    ui.RGBA(12, 16, 24, 0.34),
		fallbackColor: ui.RGBA(18, 22, 32, 0.82),
		interactive:   interactive,
	}
}

// floatingBarGlass 构建悬空底部导航栏与迷你播放器使用的深/浅通透光学玻璃材质。
func floatingBarGlass(dark bool) crystalGlassMaterial {
	if dark {
		return crystalGlassMaterial{
			blur:          16,
			bezel:         22,
			refract:       8.5,
			chromatic:     1.4,
			saturation:    1.72,
			crown:         0.09,
			rim:           0.46,
			rimWidth:      1.25,
			tint:          ui.RGBA(24, 30, 44, 0.68),
			shadowTint:    ui.RGBA(6, 9, 15, 0.25),
			fallbackColor: ui.Hex("#171B25"),
		}
	}
	return crystalGlassMaterial{
		blur:          16,
		bezel:         22,
		refract:       8.5,
		chromatic:     1.2,
		saturation:    1.52,
		crown:         0.20,
		rim:           0.88,
		rimWidth:      1.35,
		tint:          ui.RGBA(255, 255, 255, 0.68),
		shadowTint:    ui.RGBA(230, 236, 246, 0.14),
		fallbackColor: ui.Hex("#FFFFFF"),
	}
}

func (m crystalGlassMaterial) BuildMaterial(e ui.Element) ui.Material {
	if !m.interactive {
		return m
	}
	var target, hover float32
	if e.Pressed() {
		target = 1
	}
	if e.Hovered() {
		hover = 1
	}
	m.grow = e.Animate(crystalGlassKey{}, target, 140*time.Millisecond)
	m.hover = e.Animate(crystalHoverKey{}, hover, 120*time.Millisecond)
	return m
}

func (m crystalGlassMaterial) PaintMaterial(p *ui.Painter, box ui.Rect, radii [4]float32) {
	if box.W <= 0 || box.H <= 0 {
		return
	}
	if m.interactive && m.grow > 0 {
		dx, dy := 1.2*m.grow, 0.6*m.grow
		box = ui.Rect{X: box.X - dx, Y: box.Y - dy, W: box.W + 2*dx, H: box.H + 2*dy}
		for i := range radii {
			if radii[i] > 0 {
				radii[i] += dy
			}
		}
	}
	if p.Preferences().ReduceTransparency {
		p.FillRadii(box, radii, m.fallbackColor)
		return
	}

	rad := min(radii[0], min(box.W/2, box.H/2))
	if radii[1] > rad && radii[1] <= min(box.W/2, box.H/2) {
		rad = radii[1]
	}
	bezel := min(m.bezel, min(box.W/2, box.H/2))
	rim := min(m.rim+0.14*m.hover, 1)
	tint := m.tint
	if m.hover > 0 {
		tint = tint.Alpha(min(float32(tint.A)/255+0.04*m.hover, 0.92))
	}
	crown := float32(math.Min(float64(m.crown+0.04*m.hover), 0.35))

	crystalGlassShader.Draw(p, box, shader.Options{
		Radius:       rad,
		BackdropBlur: m.blur,
		Uniforms: map[string]any{
			"Radius":        rad,
			"BezelWidth":    bezel,
			"Refract":       m.refract,
			"Chromatic":     m.chromatic,
			"Saturation":    m.saturation,
			"CrownStrength": crown,
			"RimStrength":   rim,
			"RimWidth":      m.rimWidth,
			"Tint":          tint,
			"ShadowTint":    m.shadowTint,
		},
	})
}
