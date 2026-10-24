// Package synth builds a synthetic UI-mockup / diagram document for gfx benchmarks:
// rounded boxes with border, shadow and two text lines, joined by cubic connectors with
// arrowheads, every third one dashed.
package synth

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/haiodo/gowt/gfx"
)

const (
	CellW = 200.0 // grid pitch of the box layout
	CellH = 130.0

	lodBelow = 24 // device pixels
)

type Box struct {
	Node  *gfx.Node
	W, H  float64
	Conns []*Conn
}

type Conn struct {
	Node   *gfx.Node
	A, B   *Box
	Vert   bool // bottom-to-top instead of right-to-left
	Dashed bool
}

type Doc struct {
	Scene *gfx.Scene
	Boxes []*Box
	Conns []*Conn
	Cols  int
	Rows  int
}

var (
	ink    = gfx.Color{R: 0.1, G: 0.1, B: 0.12, A: 1}
	muted  = gfx.Color{R: 0.45, G: 0.45, B: 0.5, A: 1}
	border = gfx.Color{R: 0.72, G: 0.72, B: 0.76, A: 1}
	fill   = gfx.Color{R: 0.98, G: 0.98, B: 1, A: 1}
	link   = gfx.Color{R: 0.35, G: 0.38, B: 0.45, A: 1}
	accent = gfx.Color{R: 0.2, G: 0.5, B: 0.95, A: 1}
	shadow = gfx.Shadow{DX: 0, DY: 3, Blur: 8, Color: gfx.Color{A: 0.3}}
)

// New lays out n boxes in a wide grid with right and down connectors.
func New(n int, seed int64) *Doc {
	rng := rand.New(rand.NewSource(seed))
	cols := int(math.Ceil(math.Sqrt(float64(n) * 1.6)))
	rows := (n + cols - 1) / cols
	d := &Doc{Cols: cols, Rows: rows}
	d.Scene = gfx.NewScene(gfx.R(-100, -100, float64(cols)*CellW+200, float64(rows)*CellH+200), 256)
	for i := 0; i < n; i++ {
		w, h := 120+rng.Float64()*40, 56+rng.Float64()*24
		x := float64(i%cols)*CellW + rng.Float64()*20
		y := float64(i/cols)*CellH + rng.Float64()*20
		b := &Box{W: w, H: h}
		b.Node = &gfx.Node{ID: i, X: x, Y: y, Z: 1, Pic: boxPicture(i, w, h), Shape: roundRect(w, h), Hit: gfx.HitFill}
		d.Boxes = append(d.Boxes, b)
		d.Scene.Add(b.Node)
	}
	for i, b := range d.Boxes {
		if i%cols+1 < cols && i+1 < n {
			d.Link(b, d.Boxes[i+1], false)
		}
		if i+cols < n {
			d.Link(b, d.Boxes[i+cols], true)
		}
	}
	return d
}

// Link adds a connector from a to b.
func (d *Doc) Link(a, b *Box, vert bool) *Conn {
	c := &Conn{A: a, B: b, Vert: vert, Dashed: len(d.Conns)%3 == 2}
	pic, shape := c.build()
	c.Node = &gfx.Node{ID: 1<<20 + len(d.Conns), Pic: pic, Shape: shape, Hit: gfx.HitStroke}
	a.Conns = append(a.Conns, c)
	b.Conns = append(b.Conns, c)
	d.Conns = append(d.Conns, c)
	d.Scene.Add(c.Node)
	return c
}

// MoveBox moves b to (x, y), rebuilds its connectors and returns the area that went stale.
func (d *Doc) MoveBox(b *Box, x, y float64) gfx.Rect {
	dirty := d.Scene.Update(b.Node, x, y, nil)
	for _, c := range b.Conns {
		pic, shape := c.build()
		c.Node.Shape = shape
		dirty = dirty.Union(d.Scene.Update(c.Node, 0, 0, pic))
	}
	return dirty
}

func roundRect(w, h float64) *gfx.Path {
	p := &gfx.Path{}
	p.RoundRect(0.5, 0.5, w-1, h-1, 8)
	return p
}

func boxPicture(i int, w, h float64) *gfx.Picture {
	r := gfx.NewRecorder()
	r.FillShadowRRect(0, 0, w, h, 8, fill, shadow)
	r.StrokePath(roundRect(w, h), gfx.StrokeStyle{Width: 1, MiterLimit: 10}, border)
	dot := &gfx.Path{}
	dot.Ellipse(w-14, 14, 4, 4)
	r.FillPath(dot, accent)
	r.Text(fmt.Sprintf("Button %d", i), 10, 22, gfx.Font{Name: "Helvetica Neue", Size: 13}, ink)
	r.Text("Primary action label", 10, 42, gfx.Font{Name: "Helvetica Neue", Size: 11}, muted)
	pic := r.Finish()
	body := &gfx.Path{}
	body.Rect(0, 0, w, h)
	lr := gfx.NewRecorder()
	lr.FillPath(body, border)
	pic.Lod, pic.LodBelow = lr.Finish(), lodBelow
	return pic
}

// Curve returns the connector path between the anchor points of two boxes.
func Curve(a, b *Box, vert bool) *gfx.Path {
	var p0, p3, c1, c2 gfx.Point
	if vert {
		p0 = gfx.Point{X: a.Node.X + a.W/2, Y: a.Node.Y + a.H}
		p3 = gfx.Point{X: b.Node.X + b.W/2, Y: b.Node.Y}
		k := math.Max(40, math.Abs(p3.Y-p0.Y)/2)
		c1, c2 = gfx.Point{X: p0.X, Y: p0.Y + k}, gfx.Point{X: p3.X, Y: p3.Y - k}
	} else {
		p0 = gfx.Point{X: a.Node.X + a.W, Y: a.Node.Y + a.H/2}
		p3 = gfx.Point{X: b.Node.X, Y: b.Node.Y + b.H/2}
		k := math.Max(40, math.Abs(p3.X-p0.X)/2)
		c1, c2 = gfx.Point{X: p0.X + k, Y: p0.Y}, gfx.Point{X: p3.X - k, Y: p3.Y}
	}
	p := &gfx.Path{}
	p.MoveTo(p0.X, p0.Y)
	p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, p3.X, p3.Y)
	return p
}

// build records the connector in scene coordinates, so its node sits at (0, 0).
func (c *Conn) build() (*gfx.Picture, *gfx.Path) {
	curve := Curve(c.A, c.B, c.Vert)
	m := curve.Measure(0.05)
	tip, dir := m.At(m.Length())
	// the line stops at the arrow base so a dash does not poke through the head
	body, _ := curve.Split(0.98)
	head := &gfx.Path{}
	nx, ny := -dir.Y, dir.X
	bx, by := tip.X-dir.X*10, tip.Y-dir.Y*10
	head.MoveTo(tip.X, tip.Y)
	head.LineTo(bx+nx*4.5, by+ny*4.5)
	head.LineTo(bx-nx*4.5, by-ny*4.5)
	head.Close()
	st := gfx.StrokeStyle{Width: 1.5, Cap: gfx.CapRound, MiterLimit: 10}
	if c.Dashed {
		st.Dash = []float64{6, 4}
	}
	r := gfx.NewRecorder()
	r.StrokePath(body, st, link)
	r.FillPath(head, link)
	pic := r.Finish()
	lr := gfx.NewRecorder()
	lr.StrokePath(curve, gfx.StrokeStyle{Width: 1}, link)
	pic.Lod, pic.LodBelow = lr.Finish(), lodBelow
	return pic, curve
}
