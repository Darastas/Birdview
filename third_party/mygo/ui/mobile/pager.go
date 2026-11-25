package mobile

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// PageState is what page of a PageView shows, kept in the app's state.
type PageState struct {
	// Page is the page it rests at, or goes to. Set it to go there.
	Page int

	offset   ui.Point
	width    float32
	last     int
	fromPage int
	dragging bool
}

var pagerSnapAnim = ui.Animation{
	Spring: ui.SpringOptions{
		Duration:  260 * time.Millisecond,
		Bounce:    0,
		Precision: 0.5,
	},
}

// PageView shows one of count pages at a time, which a finger swipes between,
// as UIPageViewController and ViewPager do, the page it lands on following the
// throw. Only the pages next to what shows are built. It fills its parent:
// size it with Height, Grow or Fill.
//
//	mobile.PageView(c, &app.pages, 3, func(i int) { pages[i](c) }).Fill()
func PageView(c *ui.Context, s *PageState, count int, page func(i int)) ui.Element {
	box := ui.Box(c).Clip().Role(ui.RoleGroup)
	width := box.Bounds().W
	if width <= 0 {
		// Laid out for the first time: its width shows next frame.
		c.AnimationFrame()
		return box
	}
	if count <= 0 {
		return box
	}
	s.Page = max(0, min(s.Page, count-1))
	box.Children(func() {
		points := make([]ui.Point, count)
		for i := range points {
			points[i] = ui.Point{X: -float32(i) * width}
		}
		track := ui.Row(c).Absolute().Left(0).Top(0).Bottom(0).Width(width * float32(count))
		if s.width != width {
			// A new width, as the device turned: the page stays.
			s.width = width
			s.offset = points[s.Page]
			s.last = s.Page
		}
		if s.last != s.Page {
			s.last = s.Page
			s.offset = points[s.Page]
		}
		targetPage := -1
		track.Draggable(&s.offset, ui.DraggableOptions{
			Axis:      ui.Horizontal,
			Bounds:    &ui.Rect{X: -float32(count-1) * width, W: float32(count-1) * width},
			Animation: pagerSnapAnim,
			// Crisp one-page-per-swipe snapping anchored to the visual page where the gesture began.
			Snap: func(projected ui.Point) ui.Point {
				base := max(0, min(s.fromPage, count-1))
				if !s.dragging {
					base = max(0, min(s.Page, count-1))
				}
				if track.PressCanceled() {
					targetPage = base
					return points[base]
				}
				baseX := points[base].X
				dragDX := track.DragOffset().X - baseX
				projDX := projected.X - baseX
				threshold := width * 0.16
				next := base
				switch {
				case dragDX > width*0.35 || (projDX > threshold && dragDX > 0):
					next = max(0, base-1)
				case dragDX < -width*0.35 || (projDX < -threshold && dragDX < 0):
					next = min(count-1, base+1)
				}
				targetPage = next
				return points[next]
			},
		})
		shown := track.DragOffset().X
		dragging := track.Dragging()
		if dragging && !s.dragging {
			// Capture actual current slide even if user grabs mid-flight.
			s.fromPage = max(0, min(int(-shown/width+0.5), count-1))
		}
		s.dragging = dragging
		if targetPage >= 0 && targetPage != s.Page {
			// Update page index immediately upon release/snap without waiting for spring settle.
			s.Page, s.last = targetPage, targetPage
			c.AnimationFrame()
		} else if dragging {
			// Update indicator live as soon as slide crosses midpoint.
			midPage := max(0, min(int(-shown/width+0.5), count-1))
			if midPage != s.Page {
				s.Page, s.last = midPage, midPage
				c.AnimationFrame()
			}
		}
		at := max(0, min(int(-shown/width+0.5), count-1))
		lo := max(0, min(at, s.Page)-1)
		hi := min(count-1, max(at, s.Page)+1)
		track.Children(func() {
			for i := range count {
				cell := ui.Box(c.Key(i)).Width(width).Shrink(0).FillHeight()
				if i >= lo && i <= hi {
					cell.Children(func() { page(i) })
				}
			}
		})
		if !track.Transitioning() && !dragging {
			if p := max(0, min(int(-s.offset.X/width+0.5), count-1)); p != s.Page {
				s.Page, s.last = p, p
				c.AnimationFrame()
			}
		}
	})
	return box
}

// PageDots shows which of count pages is chosen, as UIPageControl does.
func PageDots(c *ui.Context, page, count int) ui.Element {
	t := c.Theme()
	return ui.Row(c).Gap(8).Justify(ui.Center).AlignItems(ui.Center).Padding(8).Children(func() {
		for i := range count {
			color := t.TextMuted.Alpha(0.35)
			if i == page {
				color = t.Text
			}
			ui.Box(c.Key(i)).Size(7, 7).Radius(3.5).Background(color)
		}
	})
}
