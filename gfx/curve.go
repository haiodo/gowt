package gfx

import (
	"math"
	"sort"
)

// Bounds is the tight bounding box of the drawn geometry: curve extrema come from the roots
// of the derivative, not from the control points. An empty path gives EmptyRect.
func (p *Path) Bounds() Rect {
	r := EmptyRect()
	var cur Point
	pi := 0
	for _, v := range p.Verbs {
		switch v {
		case MoveTo, LineTo:
			cur = p.Pts[pi]
			pi++
			r = r.AddPoint(cur)
		case QuadTo, CubicTo:
			var s seg
			if v == QuadTo {
				c, e := p.Pts[pi], p.Pts[pi+1]
				s = seg{P: [4]Point{cur, lerp(cur, c, 2.0/3), lerp(e, c, 2.0/3), e}}
				pi += 2
			} else {
				s = seg{P: [4]Point{cur, p.Pts[pi], p.Pts[pi+1], p.Pts[pi+2]}}
				pi += 3
			}
			cur = s.P[3]
			r = r.Union(s.bounds())
		}
	}
	return r
}

func (s *seg) bounds() Rect {
	r := Rect{s.P[0], s.P[0]}.AddPoint(s.P[3])
	if s.Line {
		return r
	}
	for _, axis := range [2]int{0, 1} {
		var p0, p1, p2, p3 float64
		if axis == 0 {
			p0, p1, p2, p3 = s.P[0].X, s.P[1].X, s.P[2].X, s.P[3].X
		} else {
			p0, p1, p2, p3 = s.P[0].Y, s.P[1].Y, s.P[2].Y, s.P[3].Y
		}
		// B'(t)/3 = a t^2 + b t + c
		a, b, c := -p0+3*p1-3*p2+p3, 2*(p0-2*p1+p2), p1-p0
		var roots [2]float64
		n := 0
		if math.Abs(a) < 1e-12 {
			if math.Abs(b) > 1e-12 {
				roots[0], n = -c/b, 1
			}
		} else if disc := b*b - 4*a*c; disc >= 0 {
			sq := math.Sqrt(disc)
			roots[0], roots[1], n = (-b+sq)/(2*a), (-b-sq)/(2*a), 2
		}
		for _, t := range roots[:n] {
			if t > 0 && t < 1 {
				r = r.AddPoint(s.eval(t))
			}
		}
	}
	return r
}

// flatCount is the number of equal-t pieces for a cubic to stay within tol of the curve
// (Wang's bound on the second difference of the control polygon).
func (s *seg) flatCount(tol float64) int {
	if s.Line {
		return 1
	}
	d1 := s.P[0].Sub(s.P[1].Mul(2)).Add(s.P[2])
	d2 := s.P[1].Sub(s.P[2].Mul(2)).Add(s.P[3])
	m := math.Max(d1.Len(), d2.Len())
	n := int(math.Ceil(math.Sqrt(0.75 * m / tol)))
	return min(max(n, 1), 1000)
}

// Polyline is a flattened subpath.
type Polyline struct {
	Pts    []Point
	Closed bool
}

// Flatten converts the path to polylines whose distance to the curves is at most tol.
func (p *Path) Flatten(tol float64) []Polyline {
	var out []Polyline
	sub := -1
	p.each(func(s *seg, i int, closing bool) {
		if i != sub {
			sub = i
			out = append(out, Polyline{Pts: []Point{s.P[0]}})
		}
		pl := &out[len(out)-1]
		n := s.flatCount(tol)
		for k := 1; k < n; k++ {
			pl.Pts = append(pl.Pts, s.eval(float64(k)/float64(n)))
		}
		if closing {
			pl.Closed = true // the start point is implied
		} else {
			pl.Pts = append(pl.Pts, s.P[3])
		}
	})
	return out
}

// Split cuts the path at u = segment index + t (t in 0..1; segments count every drawn line or
// curve, including the line a ClosePath adds). Both halves are open paths; closes become lines.
func (p *Path) Split(u float64) (a, b *Path) {
	a, b = &Path{}, &Path{}
	k := int(math.Floor(u))
	t := u - float64(k)
	i := 0
	lastSub := [2]int{-1, -1}
	emit := func(dst *Path, di int, s *seg, sub int) {
		if lastSub[di] != sub || len(dst.Verbs) == 0 {
			dst.MoveTo(s.P[0].X, s.P[0].Y)
			lastSub[di] = sub
		}
		if s.Line {
			dst.LineTo(s.P[3].X, s.P[3].Y)
		} else {
			dst.CubicTo(s.P[1].X, s.P[1].Y, s.P[2].X, s.P[2].Y, s.P[3].X, s.P[3].Y)
		}
	}
	p.each(func(s *seg, sub int, _ bool) {
		switch {
		case i < k:
			emit(a, 0, s, sub)
		case i == k:
			l, r := s.split(t)
			emit(a, 0, &l, sub)
			emit(b, 1, &r, sub)
		default:
			emit(b, 1, s, sub)
		}
		i++
	})
	return a, b
}

// Measure is a path prepared for arc-length queries.
type Measure struct {
	segs []seg
	cum  []float64 // cum[i] is the length before piece i; cum[len(pieces)] is the total
	pcs  []piece
}

// piece is one flat step: segment seg from t0 to t1.
type piece struct {
	seg    int
	t0, t1 float64
}

// Measure flattens the path for arc-length queries; the length is the polyline length, so it
// is slightly short on curves. Gaps between subpaths are not counted.
func (p *Path) Measure(tol float64) *Measure {
	m := &Measure{}
	p.each(func(s *seg, _ int, _ bool) {
		si := len(m.segs)
		m.segs = append(m.segs, *s)
		n := s.flatCount(tol)
		for k := 0; k < n; k++ {
			m.pcs = append(m.pcs, piece{si, float64(k) / float64(n), float64(k+1) / float64(n)})
		}
	})
	m.cum = make([]float64, len(m.pcs)+1)
	for i, pc := range m.pcs {
		s := &m.segs[pc.seg]
		m.cum[i+1] = m.cum[i] + s.eval(pc.t0).Dist(s.eval(pc.t1))
	}
	return m
}

// Length returns the length of the flattened path.
func (m *Measure) Length() float64 { return m.cum[len(m.cum)-1] }

// At returns the point and unit tangent at arc length s (clamped to 0..Length).
func (m *Measure) At(s float64) (Point, Point) {
	if len(m.pcs) == 0 {
		return Point{}, Point{}
	}
	s = math.Min(math.Max(s, 0), m.Length())
	i := sort.Search(len(m.pcs), func(i int) bool { return m.cum[i+1] >= s })
	i = min(i, len(m.pcs)-1)
	pc := m.pcs[i]
	f := 0.0
	if l := m.cum[i+1] - m.cum[i]; l > 0 {
		f = (s - m.cum[i]) / l
	}
	t := pc.t0 + (pc.t1-pc.t0)*f
	sg := &m.segs[pc.seg]
	return sg.eval(t), sg.tangent(t)
}

// Hit is the result of Nearest: the closest point, its position u (segment index + t, as in
// Split) and the distance to the query point.
type Hit struct {
	Pt   Point
	U    float64
	Dist float64
}

// Nearest finds the point on the path closest to q. The second result is false for a path
// with no segments.
func (p *Path) Nearest(q Point) (Hit, bool) {
	best := Hit{Dist: math.Inf(1)}
	i := 0
	p.each(func(s *seg, _ int, _ bool) {
		// the segment lies inside its control polygon's box
		if boxDist(s, q) < best.Dist {
			if h := s.nearest(q); h.Dist < best.Dist {
				best = h
				best.U += float64(i)
			}
		}
		i++
	})
	return best, !math.IsInf(best.Dist, 1)
}

func boxDist(s *seg, q Point) float64 {
	r := Rect{s.P[0], s.P[0]}.AddPoint(s.P[3])
	if !s.Line {
		r = r.AddPoint(s.P[1]).AddPoint(s.P[2])
	}
	dx := math.Max(math.Max(r.Min.X-q.X, 0), q.X-r.Max.X)
	dy := math.Max(math.Max(r.Min.Y-q.Y, 0), q.Y-r.Max.Y)
	return math.Hypot(dx, dy)
}

func (s *seg) nearest(q Point) Hit {
	if s.Line {
		d := s.P[3].Sub(s.P[0])
		t := 0.0
		if l2 := d.Dot(d); l2 > 0 {
			t = math.Min(math.Max(q.Sub(s.P[0]).Dot(d)/l2, 0), 1)
		}
		pt := s.eval(t)
		return Hit{pt, t, pt.Dist(q)}
	}
	const coarse = 16
	bt, bd := 0.0, math.Inf(1)
	for k := 0; k <= coarse; k++ {
		t := float64(k) / coarse
		if d := s.eval(t).Dist(q); d < bd {
			bt, bd = t, d
		}
	}
	lo, hi := math.Max(bt-1.0/coarse, 0), math.Min(bt+1.0/coarse, 1)
	for it := 0; it < 24; it++ {
		m1, m2 := lo+(hi-lo)/3, hi-(hi-lo)/3
		if s.eval(m1).Dist(q) < s.eval(m2).Dist(q) {
			hi = m2
		} else {
			lo = m1
		}
	}
	t := (lo + hi) / 2
	pt := s.eval(t)
	return Hit{pt, t, pt.Dist(q)}
}

// Contains reports whether q is inside the filled path under the non-zero winding rule.
// Curves are flattened with tolerance tol; open subpaths are treated as closed.
func (p *Path) Contains(q Point, tol float64) bool {
	wn := 0
	for _, pl := range p.Flatten(tol) {
		n := len(pl.Pts)
		for i := 0; i < n; i++ {
			a, b := pl.Pts[i], pl.Pts[(i+1)%n]
			if a.Y <= q.Y {
				if b.Y > q.Y && cross(a, b, q) > 0 {
					wn++
				}
			} else if b.Y <= q.Y && cross(a, b, q) < 0 {
				wn--
			}
		}
	}
	return wn != 0
}

func cross(a, b, q Point) float64 { return (b.X-a.X)*(q.Y-a.Y) - (q.X-a.X)*(b.Y-a.Y) }
