// Command circledraw is the 7GUIs circle drawer task: click to add a circle, select one to resize it.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
)

const (
	circleDiameter = 30
	minDiameter    = 4
	maxDiameter    = 100
)

type circle struct {
	x, y, d int
}

type circleState struct {
	circles []circle
	sel     int
}

func (s circleState) clone() circleState {
	return circleState{circles: append([]circle(nil), s.circles...), sel: s.sel}
}

type circleHistory struct {
	past, future []circleState
}

func (h *circleHistory) record(s circleState) {
	h.past = append(h.past, s.clone())
	h.future = nil
}

func (h *circleHistory) undoOnce(cur circleState) (circleState, bool) {
	if len(h.past) == 0 {
		return cur, false
	}
	h.future = append(h.future, cur.clone())
	last := len(h.past) - 1
	prev := h.past[last].clone()
	h.past = h.past[:last]
	return prev, true
}

func (h *circleHistory) redoOnce(cur circleState) (circleState, bool) {
	if len(h.future) == 0 {
		return cur, false
	}
	h.past = append(h.past, cur.clone())
	last := len(h.future) - 1
	next := h.future[last].clone()
	h.future = h.future[:last]
	return next, true
}

func topmost(circles []circle, x, y int) int {
	for i := len(circles) - 1; i >= 0; i-- {
		r := circles[i].d / 2
		dx, dy := circles[i].x-x, circles[i].y-y
		if dx*dx+dy*dy <= r*r {
			return i
		}
	}
	return -1
}

func circleDraw() gowt.Option {
	var (
		canvas   *gowt.CanvasView
		undoBtn  *gowt.ButtonView
		redoBtn  *gowt.ButtonView
		adjust   *gowt.Panel
		label    *gowt.LabelView
		diameter *gowt.ScaleView
	)
	var circles []circle
	sel := -1
	hist := circleHistory{}

	snapshot := func() circleState { return circleState{circles: circles, sel: sel} }
	refresh := func() {
		canvas.Redraw()
		undoBtn.SetEnabled(len(hist.past) > 0)
		redoBtn.SetEnabled(len(hist.future) > 0)
		ok := sel >= 0 && sel < len(circles)
		adjust.SetVisible(ok)
		if !ok {
			return
		}
		c := circles[sel]
		label.SetText(fmt.Sprintf("Adjust diameter of circle at (%d, %d).", c.x, c.y))
		diameter.SetValue(c.d)
	}
	restore := func(s circleState) {
		circles = s.circles
		sel = s.sel
		refresh()
	}

	white := gowt.RGB{R: 255, G: 255, B: 255}
	black := gowt.RGB{R: 0, G: 0, B: 0}
	grey := gowt.RGB{R: 190, G: 190, B: 190}
	return gowt.VBox(
		gowt.Canvas(func(g *gowt.GC) {
			w, h := g.Size()
			g.SetFill(white)
			g.FillRect(0, 0, w, h)
			for i, c := range circles {
				r := c.d / 2
				if i == sel {
					g.SetFill(grey)
				} else {
					g.SetFill(white)
				}
				g.FillOval(c.x-r, c.y-r, c.d, c.d)
				g.SetColor(black)
				g.Oval(c.x-r, c.y-r, c.d, c.d)
			}
			g.SetColor(black)
			g.Rect(0, 0, w-1, h-1)
		}, gowt.Size(500, 400), gowt.Ref(&canvas), gowt.Init(func(c *gowt.CanvasView) {
			c.OnMouseDown(func(m gowt.Mouse) {
				if m.Button != 1 {
					return
				}
				if i := topmost(circles, m.X, m.Y); i >= 0 {
					sel = i
					refresh()
					return
				}
				hist.record(snapshot())
				circles = append(circles, circle{x: m.X, y: m.Y, d: circleDiameter})
				sel = len(circles) - 1
				refresh()
			})
		})),
		gowt.HBox(
			gowt.Button("Undo", gowt.Ref(&undoBtn), gowt.OnClick(func() {
				if s, ok := hist.undoOnce(snapshot()); ok {
					restore(s)
				}
			})),
			gowt.Button("Redo", gowt.Ref(&redoBtn), gowt.OnClick(func() {
				if s, ok := hist.redoOnce(snapshot()); ok {
					restore(s)
				}
			}))),
		gowt.HBox(gowt.GrowX(), gowt.Ref(&adjust),
			gowt.Label("", gowt.Ref(&label)),
			gowt.Scale(minDiameter, maxDiameter, gowt.Value(minDiameter), gowt.GrowX(),
				gowt.Ref(&diameter), gowt.OnChange(func(v int) {
					if sel < 0 || sel >= len(circles) || circles[sel].d == v {
						return
					}
					hist.record(snapshot())
					circles[sel].d = v
					refresh()
				}))),
		gowt.Init(func(*gowt.Panel) { refresh() }),
	)
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Circle draw", circleDraw())
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
