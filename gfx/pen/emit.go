package pen

import (
	"fmt"
	"math"

	"github.com/haiodo/gowt/gfx"
)

// lodBelow is the device size under which a frame is drawn as a flat fill.
const lodBelow = 24

// Screen is one top level design built into the scene.
type Screen struct {
	Name string
	W, H float64
	Root *gfx.Node
	// Nodes counts the scene nodes made; Texts, Paths and Frames those with text, vector paths and
	// plain frames; Source counts the .pen nodes after refs were expanded.
	Nodes, Texts, Paths, Frames, Source int
	// Measures is the number of text measurements the layout took.
	Measures int
}

// Build resolves the top level node i, lays it out and adds the scene nodes to tx under parent
// (the scene root when nil). The screen's root node sits at the origin; move it with tx.Move.
func (d *Doc) Build(tx *gfx.Tx, parent *gfx.Node, i int) (*Screen, error) {
	if i < 0 || i >= len(d.top) {
		return nil, fmt.Errorf("pen: no top level node %d", i)
	}
	m := d.top[i]
	e := &expander{d: d}
	// the position of a top level node belongs to the canvas, not to the screen
	m = clone(m)
	delete(m, "x")
	delete(m, "y")
	root := e.expand(m, "", nil, d.o.Mode, 0)
	if root == nil {
		return nil, fmt.Errorf("pen: top level node %d is empty", i)
	}
	cnt := &Counters{}
	l := &layouter{d: d, cnt: cnt}
	l.wire(root)
	root.Layout()
	name, _ := m["name"].(string)
	sc := &Screen{Name: name, W: root.RW, H: root.RH, Source: e.nodes}
	b := &builder{d: d, tx: tx, sc: sc, l: l}
	sc.Root = b.emit(root, parent, 0, 0, true)
	sc.Measures = cnt.Measures
	return sc, nil
}

type builder struct {
	d  *Doc
	tx *gfx.Tx
	sc *Screen
	l  *layouter
}

func hasVisual(n *rn) bool {
	return n.fill != nil || (n.stroke != nil && n.sw > 0) || n.shadow != nil
}

// emit adds n and its subtree under parent; ox, oy is where parent's coordinates put n's parent
// frame, so that frames with nothing to draw add no node of their own.
func (b *builder) emit(n *rn, parent *gfx.Node, ox, oy float64, force bool) *gfx.Node {
	x, y := ox+n.RX, oy+n.RY
	clip := n.clip && b.needClip(n)
	own := force || n.typ != "frame" || hasVisual(n) || clip || n.opacity < 1
	if !own {
		for _, k := range n.kids {
			b.emit(k, parent, x, y, false)
		}
		return nil
	}
	node := gfx.NewNode(b.d.newID())
	b.sc.Nodes++
	b.tx.Add(parent, node)
	b.tx.Move(node, x, y)
	if n.opacity < 1 {
		b.tx.SetOpacity(node, n.opacity)
	}
	if pic := b.picture(n); pic != nil {
		b.tx.SetPicture(node, pic)
	}
	if clip {
		switch {
		case n.radius == [4]float64{}:
			b.tx.ClipRect(node, gfx.R(0, 0, n.RW, n.RH))
		case n.radius[0] == n.radius[1] && n.radius[1] == n.radius[2] && n.radius[2] == n.radius[3]:
			b.tx.ClipRRect(node, gfx.R(0, 0, n.RW, n.RH), n.radius[0])
		default:
			b.tx.ClipPath(node, rrect4(0, 0, n.RW, n.RH, n.radius))
		}
	}
	if n.typ == "ellipse" {
		p := &gfx.Path{}
		p.Ellipse(n.RW/2, n.RH/2, n.RW/2, n.RH/2)
		b.tx.SetShape(node, p, gfx.HitFill)
	}
	for _, k := range n.kids {
		b.emit(k, node, 0, 0, false)
	}
	return node
}

// needClip tells whether clipping n changes what is drawn: a child leaves the frame, or the rounded
// corners are touched by a child.
func (b *builder) needClip(n *rn) bool {
	rmax := math.Max(math.Max(n.radius[0], n.radius[1]), math.Max(n.radius[2], n.radius[3]))
	for _, k := range n.kids {
		if k.RX < 0 || k.RY < 0 || k.RX+k.RW > n.RW+0.5 || k.RY+k.RH > n.RH+0.5 {
			return true
		}
		if rmax > 0 && (k.RX < rmax || k.RY < rmax || k.RX+k.RW > n.RW-rmax || k.RY+k.RH > n.RH-rmax) &&
			(k.RX < rmax && k.RY < rmax || k.RX+k.RW > n.RW-rmax && k.RY < rmax ||
				k.RX < rmax && k.RY+k.RH > n.RH-rmax || k.RX+k.RW > n.RW-rmax && k.RY+k.RH > n.RH-rmax) {
			return true
		}
		if k.typ == "frame" && !k.clip && b.leaks(k, n.RW-k.RX, n.RH-k.RY, -k.RX, -k.RY) {
			return true
		}
	}
	return false
}

// leaks reports whether a descendant of an unclipped frame reaches outside the box that starts at
// (-ox, -oy) and ends at (maxX, maxY) in the coordinates of that frame.
func (b *builder) leaks(n *rn, maxX, maxY, minX, minY float64) bool {
	for _, k := range n.kids {
		if k.RX < minX || k.RY < minY || k.RX+k.RW > maxX+0.5 || k.RY+k.RH > maxY+0.5 {
			return true
		}
		if k.typ == "frame" && !k.clip && b.leaks(k, maxX-k.RX, maxY-k.RY, minX-k.RX, minY-k.RY) {
			return true
		}
	}
	return false
}

func rrect4(x, y, w, h float64, r [4]float64) *gfx.Path {
	p := &gfx.Path{}
	if r[0] == r[1] && r[1] == r[2] && r[2] == r[3] {
		p.RoundRect(x, y, w, h, r[0])
		return p
	}
	lim := math.Min(w, h) / 2
	for i := range r {
		r[i] = math.Min(r[i], lim)
	}
	const k = 0.4477152501692064 // 1 - kappa
	p.MoveTo(x+r[0], y)
	p.LineTo(x+w-r[1], y)
	p.CubicTo(x+w-r[1]*k, y, x+w, y+r[1]*k, x+w, y+r[1])
	p.LineTo(x+w, y+h-r[2])
	p.CubicTo(x+w, y+h-r[2]*k, x+w-r[2]*k, y+h, x+w-r[2], y+h)
	p.LineTo(x+r[3], y+h)
	p.CubicTo(x+r[3]*k, y+h, x, y+h-r[3]*k, x, y+h-r[3])
	p.LineTo(x, y+r[0])
	p.CubicTo(x, y+r[0]*k, x+r[0]*k, y, x+r[0], y)
	p.Close()
	return p
}

func (b *builder) gradient(p *paint, w, h float64) *gfx.Gradient {
	a := p.angle * math.Pi / 180
	dx, dy := math.Sin(a), -math.Cos(a)
	half := (math.Abs(w*dx) + math.Abs(h*dy)) / 2
	c := gfx.Point{X: w / 2, Y: h / 2}
	return gfx.NewLinearGradient(gfx.Point{X: c.X - dx*half, Y: c.Y - dy*half}, gfx.Point{X: c.X + dx*half, Y: c.Y + dy*half}, p.grad...)
}

func (b *builder) picture(n *rn) *gfx.Picture {
	w, h := n.RW, n.RH
	switch n.typ {
	case "text":
		return b.textPicture(n)
	case "path":
		return b.pathPicture(n)
	case "ellipse":
		n.radius = [4]float64{w / 2, w / 2, w / 2, w / 2}
	}
	if !hasVisual(n) || w <= 0 || h <= 0 {
		return nil
	}
	r := gfx.NewRecorderWith(b.d.o.Measurer)
	uniform := n.radius[0] == n.radius[1] && n.radius[1] == n.radius[2] && n.radius[2] == n.radius[3]
	shape := rrect4(0, 0, w, h, n.radius)
	b.sc.Frames++
	switch {
	case n.shadow != nil && uniform:
		fill := gfx.Color{R: 1, G: 1, B: 1, A: 1}
		if n.fill != nil {
			fill = n.fill.c
		}
		if n.fill != nil && n.fill.grad != nil {
			fill.A = 0
		}
		sh := gfx.Shadow{DX: n.shadow.dx, DY: n.shadow.dy, Blur: n.shadow.blur, Color: n.shadow.c}
		r.FillShadowRRect(0, 0, w, h, n.radius[0], fill, sh)
		if n.fill != nil && n.fill.grad != nil {
			r.FillGradient(shape, b.gradient(n.fill, w, h))
		}
	case n.fill != nil && n.fill.grad != nil:
		r.FillGradient(shape, b.gradient(n.fill, w, h))
	case n.fill != nil:
		r.FillPath(shape, n.fill.c)
	}
	if n.stroke != nil && n.sw > 0 {
		sp := shape
		if n.inner {
			in := n.sw / 2
			rr := n.radius
			for i := range rr {
				rr[i] = math.Max(rr[i]-in, 0)
			}
			sp = rrect4(in, in, w-n.sw, h-n.sw, rr)
		}
		st := gfx.StrokeStyle{Width: n.sw, MiterLimit: 10}
		if n.stroke.grad != nil {
			r.StrokeGradient(sp, st, b.gradient(n.stroke, w, h))
		} else {
			r.StrokePath(sp, st, n.stroke.c)
		}
	}
	pic := r.Finish()
	if n.fill != nil {
		lr := gfx.NewRecorderWith(b.d.o.Measurer)
		lr.FillPath(shape, n.fill.c)
		pic.Lod, pic.LodBelow = lr.Finish(), lodBelow
	}
	return pic
}

func (b *builder) textPicture(n *rn) *gfx.Picture {
	if len(n.lines) == 0 || n.fill == nil {
		return nil
	}
	b.sc.Texts++
	f := b.l.font(n)
	m := b.d.o.Measurer.MeasureText("Hg", f)
	lh := b.l.lineH(n)
	r := gfx.NewRecorderWith(b.d.o.Measurer)
	for i, ln := range n.lines {
		if ln.s == "" {
			continue
		}
		x := 0.0
		switch n.talign {
		case "center":
			x = (n.RW - ln.w) / 2
		case "right":
			x = n.RW - ln.w
		}
		base := float64(i)*lh + (lh-(m.Ascent+m.Descent))/2 + m.Ascent
		r.Text(ln.s, x, base, f, n.fill.c)
	}
	return r.Finish()
}

func (b *builder) pathPicture(n *rn) *gfx.Picture {
	if n.geom == "" || (n.fill == nil && n.stroke == nil) {
		return nil
	}
	p, err := ParseSVGPath(n.geom)
	if err != nil || p.Empty() {
		b.d.Skipped["bad path"]++
		return nil
	}
	b.sc.Paths++
	sx, sy := 1.0, 1.0
	if n.vb[2] > 0 && n.vb[3] > 0 {
		sx, sy = n.RW/n.vb[2], n.RH/n.vb[3]
	}
	p = p.Transform(gfx.Affine{sx, 0, 0, sy, -n.vb[0] * sx, -n.vb[1] * sy})
	r := gfx.NewRecorderWith(b.d.o.Measurer)
	if n.fill != nil {
		if n.fill.grad != nil {
			r.FillGradient(p, b.gradient(n.fill, n.RW, n.RH))
		} else {
			r.FillPath(p, n.fill.c)
		}
	}
	if n.stroke != nil && n.sw > 0 {
		r.StrokePath(p, gfx.StrokeStyle{Width: n.sw, Cap: gfx.CapRound, Join: gfx.JoinRound, MiterLimit: 10}, n.stroke.c)
	}
	return r.Finish()
}
