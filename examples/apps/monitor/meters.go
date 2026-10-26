package main

import (
	"math"
	"slices"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/swt"
)

// smooth eases a list of values towards their targets, independent of the frame rate.
type smooth struct {
	cur, tgt []float64
	last     time.Duration
}

func (s *smooth) set(t []float64) {
	if len(s.cur) != len(t) {
		s.cur = append([]float64(nil), t...)
	}
	s.tgt = append(s.tgt[:0], t...)
}

// step moves cur at frame time now and reports whether anything is still moving.
func (s *smooth) step(now time.Duration, tau float64) bool {
	dt := (now - s.last).Seconds()
	if dt <= 0 || dt > 0.1 {
		dt = 1.0 / 60
	}
	s.last = now
	k := 1 - math.Exp(-dt/tau)
	moving := false
	for i := range s.cur {
		d := s.tgt[i] - s.cur[i]
		if math.Abs(d) < 0.002 {
			s.cur[i] = s.tgt[i]
			continue
		}
		s.cur[i] += d * k
		moving = true
	}
	return moving
}

// meterView is a gfx view of a few eased values drawn as bars: the scene is a background node and
// one node per item, whose picture is recorded again while its value moves.
type meterView struct {
	pal    *palette
	scene  *gfx.Scene
	bg     *gfx.Node
	nodes  []*gfx.Node
	v      *view.View
	w, h   float64
	sm     smooth
	on     bool
	nextID int
	// layout and draw are the two things that differ between the core meters and the memory bar:
	// where item i sits, and what its picture is for value cur[i] (all values for a single bar).
	layout func(m *meterView)
	draw   func(m *meterView, i int) *gfx.Picture
	pos    func(m *meterView, i int) gfx.Point
	dirty  bool
	// text is drawn as sprites: they are rendered once and then only placed, where a tile with text
	// is rendered again on every frame of the bar animation. full is set when the layout changed.
	text func(m *meterView, full bool)
}

func newMeterView(pal *palette) *meterView {
	m := &meterView{pal: pal, scene: gfx.NewScene(), bg: gfx.NewNode(1), nextID: 2}
	m.scene.Update(func(tx *gfx.Tx) { tx.Add(nil, m.bg) })
	return m
}

func (m *meterView) option(minW, minH int) g.Option {
	return view.New(m.scene, m.attach, g.Grow(), g.Size(minW, minH))
}

// strip is a view of a fixed height that fills the width.
func (m *meterView) strip(h int) g.Option {
	return view.New(m.scene, m.attach, g.GrowX(), g.Size(0, h))
}

func (m *meterView) attach(v *view.View) {
	m.v = v
	v.SetZoomLimits(1, 1)
	v.OnScroll(func(*view.Scroll) bool { return true })
	v.OnMagnify(func(*view.Magnify) bool { return true })
	v.OnSmartZoom(func(*view.SmartZoom) bool { return true })
	v.Canvas().Unwrap().AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) {
		if w, h := v.Canvas().Size(); w > 0 && h > 0 {
			m.w, m.h = float64(w), float64(h)
			m.relayout()
		}
	}))
}

func (m *meterView) setPalette(p *palette) {
	m.pal = p
	m.relayout()
}

// setValues gives the targets; nodes are added when the count grows.
func (m *meterView) setValues(v []float64, nodes int) {
	for len(m.nodes) < nodes {
		n := gfx.NewNode(m.nextID)
		m.nextID++
		m.nodes = append(m.nodes, n)
		m.scene.Update(func(tx *gfx.Tx) { tx.Add(nil, n) })
		m.dirty = true
	}
	m.sm.set(v)
	if m.dirty {
		m.relayout()
	}
	if m.text != nil && m.v != nil && m.w > 0 {
		m.text(m, false)
	}
	m.wake()
}

// relayout draws the background and every item again, after a resize, a palette change or a new count.
func (m *meterView) relayout() {
	m.dirty = false
	if m.w == 0 || m.layout == nil {
		return
	}
	if m.v != nil {
		m.pal.setCanvasBackground(m.v.Canvas().Unwrap())
	}
	m.layout(m)
	r := gfx.NewRecorder()
	r.FillRect(gfx.R(0, 0, m.w, m.h), m.pal.card)
	bgPic := r.Finish()
	pics := make([]*gfx.Picture, len(m.nodes))
	for i := range m.nodes {
		pics[i] = m.draw(m, i)
	}
	m.scene.Update(func(tx *gfx.Tx) {
		tx.SetPicture(m.bg, bgPic)
		for i, n := range m.nodes {
			tx.SetPicture(n, pics[i])
			p := m.pos(m, i)
			tx.Move(n, p.X, p.Y)
		}
	})
	if m.text != nil && m.v != nil {
		m.text(m, true)
	}
}

func (m *meterView) wake() {
	if m.v == nil || m.on {
		return
	}
	m.on = true
	var placed time.Duration
	m.v.Compositor().OnFrame(func(fi gfx.FrameInfo) {
		if fi.Time-placed < frameGap {
			return
		}
		placed = fi.Time
		moving := m.sm.step(fi.Time, 0.1)
		pics := make([]*gfx.Picture, len(m.nodes))
		for i := range m.nodes {
			pics[i] = m.draw(m, i)
		}
		m.scene.Update(func(tx *gfx.Tx) {
			for i, n := range m.nodes {
				tx.SetPicture(n, pics[i]) // an equal picture is ignored by the scene
			}
		})
		if !moving {
			m.on = false
			m.v.Compositor().OnFrame(nil)
		}
	})
}

// load colour: the accent, turning red towards full load.
func loadColor(p *palette, v float64) gfx.Color {
	t := min(max((v-0.7)/0.3, 0), 1)
	return mix(p.accent, p.red, t*t*(3-2*t))
}

func rrect(x, y, w, h float64) *gfx.Path {
	if w < 0.5 {
		return &gfx.Path{}
	}
	return gfx.RoundRectPath(gfx.R(x, y, w, h), min(h/2, w/2))
}

// newCoreMeters is one row per core: index, bar, percent.
func newCoreMeters(pal *palette) *meterView {
	m := newMeterView(pal)
	const labelW, valueW, gap, barH = 22.0, 34.0, 18.0, 8.0
	type grid struct{ cols, rows int }
	var gr grid
	cellW := func(m *meterView) float64 { return (m.w - gap*float64(gr.cols-1)) / float64(gr.cols) }
	rowH := func(m *meterView) float64 { return min(m.h/float64(gr.rows), 30) }
	m.layout = func(m *meterView) {
		n := max(len(m.nodes), 1)
		gr = grid{1, n}
		for gr.cols < n && m.h/float64(gr.rows) < 22 {
			gr.cols++
			gr.rows = (n + gr.cols - 1) / gr.cols
		}
	}
	m.pos = func(m *meterView, i int) gfx.Point {
		return gfx.Point{X: float64(i/gr.rows) * (cellW(m) + gap), Y: float64(i%gr.rows) * rowH(m)}
	}
	m.draw = func(m *meterView, i int) *gfx.Picture {
		p, w, h := m.pal, cellW(m), rowH(m)
		v := 0.0
		if i < len(m.sm.cur) {
			v = m.sm.cur[i]
		}
		r := gfx.NewRecorder()
		y := (h - barH) / 2
		bx, bw := labelW, w-labelW-valueW
		r.FillPath(rrect(bx, y, bw, barH), p.track())
		if v > 0.004 {
			c := loadColor(p, v)
			fw := max(v*bw, barH)
			r.FillGradient(rrect(bx, y, fw, barH), gfx.NewLinearGradient(gfx.Point{X: bx}, gfx.Point{X: bx + bw},
				gfx.Stop{Offset: 0, Color: c.WithAlpha(0.55)}, gfx.Stop{Offset: 1, Color: c}))
		}
		return r.Finish()
	}
	var idx, vals *gfx.Sprite
	var shown []string
	m.text = func(m *meterView, full bool) {
		if idx == nil {
			comp := m.v.Compositor()
			idx = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
			vals = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
		}
		p, w, h := m.pal, cellW(m), rowH(m)
		base := h/2 + p.small.Size*0.36
		texts := make([]string, len(m.nodes))
		for i := range texts {
			texts[i] = pct(m.sm.tgt[min(i, len(m.sm.tgt)-1)])
		}
		if full {
			r := gfx.NewRecorder()
			for i := range m.nodes {
				at := m.pos(m, i)
				r.Text(itoa(i), at.X, at.Y+base, p.small, p.labelSoft)
			}
			idx.SetPicture(r.Finish())
			shown = nil
		}
		if slices.Equal(texts, shown) {
			return
		}
		shown = texts
		r := gfx.NewRecorder()
		for i, s := range texts {
			at := m.pos(m, i)
			r.Text(s, at.X+w-gfxWidth(s, p.small), at.Y+base, p.small, p.label)
		}
		vals.SetPicture(r.Finish())
	}
	return m
}

func gfxWidth(s string, f gfx.Font) float64 { return gfx.DefaultMeasurer.MeasureText(s, f).Width }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	n := len(b)
	for ; i > 0; i /= 10 {
		n--
		b[n] = byte('0' + i%10)
	}
	return string(b[n:])
}

// memBar is the memory breakdown as one stacked bar: values are the fractions of RAM in the order
// of the segments, and the rest is free.
func newMemBar(pal *palette, colors func(p *palette) []gfx.Color) *meterView {
	m := newMeterView(pal)
	m.layout = func(m *meterView) {}
	m.pos = func(m *meterView, i int) gfx.Point { return gfx.Point{} }
	m.draw = func(m *meterView, i int) *gfx.Picture {
		p := m.pal
		r := gfx.NewRecorder()
		h := min(m.h, 12)
		y := (m.h - h) / 2
		r.FillPath(rrect(0, y, m.w, h), p.track())
		cols := colors(p)
		// clip to the track so that segment ends are square inside and round at the ends
		r.Save()
		r.ClipPath(rrect(0, y, m.w, h))
		x := 0.0
		for k, f := range m.sm.cur {
			w := f * m.w
			if w < 0.5 {
				continue
			}
			r.FillRect(gfx.R(x, y, w, h), cols[k])
			x += w
		}
		r.Restore()
		return r.Finish()
	}
	return m
}
