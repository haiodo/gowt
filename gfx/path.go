package gfx

import "math"

// Verb is one drawing step of a Path.
type Verb uint8

// The verbs of a Path.
const (
	MoveTo Verb = iota
	LineTo
	QuadTo
	CubicTo
	ClosePath
)

// Path is a list of verbs over a flat point list: MoveTo and LineTo take one point,
// QuadTo two, CubicTo three, ClosePath none. Arcs and ellipses are stored as cubics.
type Path struct {
	Verbs []Verb
	Pts   []Point
	// EvenOdd selects the even-odd fill rule, for fills and clips; the default is non-zero winding.
	EvenOdd bool
}

// MoveTo starts a new subpath at (x, y).
func (p *Path) MoveTo(x, y float64) {
	p.Verbs = append(p.Verbs, MoveTo)
	p.Pts = append(p.Pts, Point{x, y})
}

// LineTo adds a straight line to (x, y).
func (p *Path) LineTo(x, y float64) {
	p.Verbs = append(p.Verbs, LineTo)
	p.Pts = append(p.Pts, Point{x, y})
}

// QuadTo adds a quadratic curve with control point (cx, cy) ending at (x, y).
func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.Verbs = append(p.Verbs, QuadTo)
	p.Pts = append(p.Pts, Point{cx, cy}, Point{x, y})
}

// CubicTo adds a cubic curve with two control points ending at (x, y).
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.Verbs = append(p.Verbs, CubicTo)
	p.Pts = append(p.Pts, Point{c1x, c1y}, Point{c2x, c2y}, Point{x, y})
}

// Close ends the subpath with a line back to its start.
func (p *Path) Close() { p.Verbs = append(p.Verbs, ClosePath) }

// Empty reports whether p has no verbs.
func (p *Path) Empty() bool { return len(p.Verbs) == 0 }

// Reset removes every verb, keeping the memory.
func (p *Path) Reset() { p.Verbs, p.Pts = p.Verbs[:0], p.Pts[:0] }

// Clone returns a deep copy.
func (p *Path) Clone() *Path {
	return &Path{append([]Verb(nil), p.Verbs...), append([]Point(nil), p.Pts...), p.EvenOdd}
}

// Rect adds a closed rectangle.
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// RoundRect adds a rectangle with corner radius r (clamped to half the shorter side).
func (p *Path) RoundRect(x, y, w, h, r float64) {
	r = math.Min(r, math.Min(w, h)/2)
	if r <= 0 {
		p.Rect(x, y, w, h)
		return
	}
	k := r * (1 - kappa)
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.CubicTo(x+w-k, y, x+w, y+k, x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.CubicTo(x+w, y+h-k, x+w-k, y+h, x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.CubicTo(x+k, y+h, x, y+h-k, x, y+h-r)
	p.LineTo(x, y+r)
	p.CubicTo(x, y+k, x+k, y, x+r, y)
	p.Close()
}

// RoundRect4 adds a rectangle with its own radius at each corner, in the order top left, top right,
// bottom right, bottom left. Each radius is clamped to half the shorter side.
func (p *Path) RoundRect4(x, y, w, h float64, r [4]float64) {
	if r[0] == r[1] && r[1] == r[2] && r[2] == r[3] {
		p.RoundRect(x, y, w, h, r[0])
		return
	}
	lim := math.Min(w, h) / 2
	for i := range r {
		r[i] = math.Min(r[i], lim)
	}
	const k = 1 - kappa
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
}

// kappa is the cubic control distance for a quarter circle of radius 1.
const kappa = 0.5522847498307936

// Ellipse adds a closed ellipse centred at (cx, cy) as four cubics.
func (p *Path) Ellipse(cx, cy, rx, ry float64) {
	kx, ky := rx*kappa, ry*kappa
	p.MoveTo(cx+rx, cy)
	p.CubicTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	p.CubicTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	p.CubicTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	p.CubicTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	p.Close()
}

// Arc adds a circular arc from angle a0 to a1 (radians, y down: positive is clockwise on screen)
// as cubics of at most 90 degrees; it joins the current subpath with a line to the arc start.
func (p *Path) Arc(cx, cy, r, a0, a1 float64) {
	s, c := math.Sincos(a0)
	if len(p.Verbs) == 0 {
		p.MoveTo(cx+r*c, cy+r*s)
	} else {
		p.LineTo(cx+r*c, cy+r*s)
	}
	sweep := a1 - a0
	n := int(math.Ceil(math.Abs(sweep)/(math.Pi/2) - 1e-9)) // rounding must not split a quarter turn
	if n == 0 {
		return
	}
	step := sweep / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	for i := 0; i < n; i++ {
		b0 := a0 + float64(i)*step
		b1 := b0 + step
		s0, c0 := math.Sincos(b0)
		s1, c1 := math.Sincos(b1)
		p.CubicTo(cx+r*(c0-k*s0), cy+r*(s0+k*c0), cx+r*(c1+k*s1), cy+r*(s1-k*c1), cx+r*c1, cy+r*s1)
	}
}

// ArcTo adds the arc of radius r that is tangent to the line from the current point to (x1, y1) and
// to the line from (x1, y1) to (x2, y2), as the HTML canvas arcTo does: a straight segment to the
// first tangent point, then the arc to the second. A zero radius, a repeated point or three points
// on a line give a straight segment to (x1, y1); without a current point the path starts at (x1, y1).
func (p *Path) ArcTo(x1, y1, x2, y2, r float64) {
	if len(p.Verbs) == 0 {
		p.MoveTo(x1, y1)
		return
	}
	p0, p1, p2 := p.current(), Point{x1, y1}, Point{x2, y2}
	d1, d2 := p0.Sub(p1), p2.Sub(p1)
	l1, l2 := d1.Len(), d2.Len()
	if l1 == 0 || l2 == 0 || r <= 0 {
		p.LineTo(x1, y1)
		return
	}
	d1, d2 = d1.Mul(1/l1), d2.Mul(1/l2)
	cross := d1.X*d2.Y - d1.Y*d2.X
	if math.Abs(cross) < 1e-12 {
		p.LineTo(x1, y1)
		return
	}
	half := math.Acos(math.Max(-1, math.Min(1, d1.Dot(d2)))) / 2 // half the angle at p1
	t := r / math.Tan(half)
	c := p1.Add(d1.Add(d2).Unit().Mul(r / math.Sin(half)))
	t1, t2 := p1.Add(d1.Mul(t)), p1.Add(d2.Mul(t))
	a0, a1 := math.Atan2(t1.Y-c.Y, t1.X-c.X), math.Atan2(t2.Y-c.Y, t2.X-c.X)
	sweep := a1 - a0
	for sweep > math.Pi {
		sweep -= 2 * math.Pi
	}
	for sweep < -math.Pi {
		sweep += 2 * math.Pi
	}
	p.Arc(c.X, c.Y, r, a0, a0+sweep)
}

// current is the end of the path: its last point, or the start of the last subpath after a Close.
func (p *Path) current() Point {
	if n := len(p.Verbs); n == 0 || p.Verbs[n-1] != ClosePath {
		return p.Pts[len(p.Pts)-1]
	}
	i, start := 0, 0
	for _, v := range p.Verbs {
		switch v {
		case MoveTo:
			start = i
			i++
		case LineTo:
			i++
		case QuadTo:
			i += 2
		case CubicTo:
			i += 3
		}
	}
	return p.Pts[start]
}

// Transform returns a copy of p with every point mapped by m.
func (p *Path) Transform(m Affine) *Path {
	out := p.Clone()
	for i, q := range out.Pts {
		out.Pts[i] = m.Apply(q)
	}
	return out
}

// seg is one drawn segment: a line (Line set, end point in P[3]) or a cubic. Quads are
// promoted to cubics, which is exact.
type seg struct {
	P    [4]Point
	Line bool
}

func (s *seg) eval(t float64) Point {
	if s.Line {
		return lerp(s.P[0], s.P[3], t)
	}
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return Point{a*s.P[0].X + b*s.P[1].X + c*s.P[2].X + d*s.P[3].X, a*s.P[0].Y + b*s.P[1].Y + c*s.P[2].Y + d*s.P[3].Y}
}

func (s *seg) deriv(t float64) Point {
	if s.Line {
		return s.P[3].Sub(s.P[0])
	}
	u := 1 - t
	a, b, c := 3*u*u, 6*u*t, 3*t*t
	d0, d1, d2 := s.P[1].Sub(s.P[0]), s.P[2].Sub(s.P[1]), s.P[3].Sub(s.P[2])
	return Point{a*d0.X + b*d1.X + c*d2.X, a*d0.Y + b*d1.Y + c*d2.Y}
}

// tangent is the unit direction at t; at a degenerate point of a cubic it falls back to the chord.
func (s *seg) tangent(t float64) Point {
	d := s.deriv(t)
	if d.X == 0 && d.Y == 0 {
		d = s.P[3].Sub(s.P[0])
	}
	return d.Unit()
}

func (s *seg) split(t float64) (a, b seg) {
	if s.Line {
		m := lerp(s.P[0], s.P[3], t)
		return seg{P: [4]Point{s.P[0], {}, {}, m}, Line: true}, seg{P: [4]Point{m, {}, {}, s.P[3]}, Line: true}
	}
	p01, p12, p23 := lerp(s.P[0], s.P[1], t), lerp(s.P[1], s.P[2], t), lerp(s.P[2], s.P[3], t)
	p012, p123 := lerp(p01, p12, t), lerp(p12, p23, t)
	m := lerp(p012, p123, t)
	return seg{P: [4]Point{s.P[0], p01, p012, m}}, seg{P: [4]Point{m, p123, p23, s.P[3]}}
}

// each calls fn for every drawn segment in order. sub is the subpath index and closing marks
// the line that ClosePath adds back to the subpath start (zero-length closes are skipped).
func (p *Path) each(fn func(s *seg, sub int, closing bool)) {
	var cur, start Point
	sub := -1
	pi := 0
	var s seg
	for _, v := range p.Verbs {
		switch v {
		case MoveTo:
			cur = p.Pts[pi]
			start = cur
			pi++
			sub++
		case LineTo:
			s = seg{P: [4]Point{cur, {}, {}, p.Pts[pi]}, Line: true}
			cur = p.Pts[pi]
			pi++
			fn(&s, sub, false)
		case QuadTo:
			c, e := p.Pts[pi], p.Pts[pi+1]
			s = seg{P: [4]Point{cur, lerp(cur, c, 2.0/3), lerp(e, c, 2.0/3), e}}
			cur = e
			pi += 2
			fn(&s, sub, false)
		case CubicTo:
			s = seg{P: [4]Point{cur, p.Pts[pi], p.Pts[pi+1], p.Pts[pi+2]}}
			cur = p.Pts[pi+2]
			pi += 3
			fn(&s, sub, false)
		case ClosePath:
			if cur != start {
				s = seg{P: [4]Point{cur, {}, {}, start}, Line: true}
				cur = start
				fn(&s, sub, true)
			}
		}
	}
}

// RectPath returns a closed path around r; it is also the hit shape of a plain box.
func RectPath(r Rect) *Path {
	p := &Path{}
	p.Rect(r.Min.X, r.Min.Y, r.W(), r.H())
	return p
}

// RoundRectPath returns RectPath with the corners rounded by radius.
func RoundRectPath(r Rect, radius float64) *Path {
	p := &Path{}
	p.RoundRect(r.Min.X, r.Min.Y, r.W(), r.H(), radius)
	return p
}

// RoundRectPath4 returns RectPath with each corner rounded by its own radius (top left, top right,
// bottom right, bottom left).
func RoundRectPath4(r Rect, radii [4]float64) *Path {
	p := &Path{}
	p.RoundRect4(r.Min.X, r.Min.Y, r.W(), r.H(), radii)
	return p
}

// EllipsePath returns the ellipse that fits in r.
func EllipsePath(r Rect) *Path {
	p := &Path{}
	c := r.Center()
	p.Ellipse(c.X, c.Y, r.W()/2, r.H()/2)
	return p
}

// PolygonPath returns the closed path through pts, in order.
func PolygonPath(pts ...Point) *Path {
	p := &Path{}
	for i, q := range pts {
		if i == 0 {
			p.MoveTo(q.X, q.Y)
		} else {
			p.LineTo(q.X, q.Y)
		}
	}
	if len(pts) > 0 {
		p.Close()
	}
	return p
}
