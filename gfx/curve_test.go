package gfx

import (
	"math"
	"math/rand"
	"testing"
)

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestBoundsTight(t *testing.T) {
	// the control points reach y=100 but the curve peaks at y=75
	var p Path
	p.MoveTo(0, 0)
	p.CubicTo(0, 100, 100, 100, 100, 0)
	b := p.Bounds()
	if !near(b.Max.Y, 75, 1e-9) || !near(b.Min.Y, 0, 1e-9) || !near(b.Min.X, 0, 1e-9) || !near(b.Max.X, 100, 1e-9) {
		t.Fatalf("bounds %+v", b)
	}
	var q Path
	q.MoveTo(0, 0)
	q.QuadTo(50, 100, 100, 0)
	if b := q.Bounds(); !near(b.Max.Y, 50, 1e-9) {
		t.Fatalf("quad bounds %+v", b)
	}
	var e Path
	e.Ellipse(10, 20, 5, 3)
	b = e.Bounds()
	if !near(b.Min.X, 5, 1e-3) || !near(b.Max.X, 15, 1e-3) || !near(b.Min.Y, 17, 1e-3) || !near(b.Max.Y, 23, 1e-3) {
		t.Fatalf("ellipse bounds %+v", b)
	}
	if !(&Path{}).Bounds().Empty() {
		t.Fatal("empty path must have empty bounds")
	}
}

func TestFlattenWithinTolerance(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.CubicTo(0, 200, 300, -100, 300, 100)
	const tol = 0.1
	pl := p.Flatten(tol)[0]
	s := seg{P: [4]Point{{0, 0}, {0, 200}, {300, -100}, {300, 100}}}
	// each curve sample must lie within tol of the polyline
	for k := 0; k <= 2000; k++ {
		c := s.eval(float64(k) / 2000)
		d := math.Inf(1)
		for i := 0; i+1 < len(pl.Pts); i++ {
			l := seg{P: [4]Point{pl.Pts[i], {}, {}, pl.Pts[i+1]}, Line: true}
			d = math.Min(d, l.nearest(c).Dist)
		}
		if d > tol {
			t.Fatalf("t=%v distance %v > %v", float64(k)/2000, d, tol)
		}
	}
}

func TestLengthAndAt(t *testing.T) {
	var p Path
	p.Ellipse(0, 0, 50, 50)
	m := p.Measure(0.01)
	if !near(m.Length(), 2*math.Pi*50, 0.2) {
		t.Fatalf("circle length %v", m.Length())
	}
	pt, tan := m.At(m.Length() / 4) // quarter turn from (50,0) towards +y
	if !near(pt.X, 0, 0.1) || !near(pt.Y, 50, 0.1) || !near(tan.X, -1, 1e-2) || !near(tan.Y, 0, 1e-2) {
		t.Fatalf("quarter point %+v tangent %+v", pt, tan)
	}
	var l Path
	l.MoveTo(0, 0)
	l.LineTo(30, 40)
	l.LineTo(30, 100)
	ml := l.Measure(0.1)
	if !near(ml.Length(), 110, 1e-9) {
		t.Fatalf("polyline length %v", ml.Length())
	}
	pt, tan = ml.At(50 + 10)
	if pt != (Point{30, 50}) || tan != (Point{0, 1}) {
		t.Fatalf("at 60: %+v %+v", pt, tan)
	}
	if pt, _ = ml.At(1e9); pt != (Point{30, 100}) {
		t.Fatalf("clamp %+v", pt)
	}
}

func TestNearest(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.CubicTo(0, 100, 100, 100, 100, 0)
	h, ok := p.Nearest(Point{50, 200})
	if !ok || !near(h.Pt.X, 50, 1e-3) || !near(h.Pt.Y, 75, 1e-3) || !near(h.U, 0.5, 1e-3) {
		t.Fatalf("hit %+v", h)
	}
	// brute-force check on random points
	s := seg{P: [4]Point{{0, 0}, {0, 100}, {100, 100}, {100, 0}}}
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		q := Point{r.Float64()*200 - 50, r.Float64()*200 - 50}
		h, _ := p.Nearest(q)
		best := math.Inf(1)
		for k := 0; k <= 20000; k++ {
			best = math.Min(best, s.eval(float64(k)/20000).Dist(q))
		}
		if h.Dist > best+1e-3 {
			t.Fatalf("q=%v got %v brute %v", q, h.Dist, best)
		}
	}
	if _, ok := (&Path{}).Nearest(Point{}); ok {
		t.Fatal("empty path has no nearest point")
	}
}

func TestSplit(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.LineTo(100, 0)
	p.CubicTo(150, 0, 200, 50, 200, 100)
	a, b := p.Split(1.5)
	ma, mb, mp := a.Measure(0.01), b.Measure(0.01), p.Measure(0.01)
	if !near(ma.Length()+mb.Length(), mp.Length(), 1e-2) {
		t.Fatalf("lengths %v + %v != %v", ma.Length(), mb.Length(), mp.Length())
	}
	s := seg{P: [4]Point{{100, 0}, {150, 0}, {200, 50}, {200, 100}}}
	want := s.eval(0.5)
	end := a.Pts[len(a.Pts)-1]
	if end.Dist(want) > 1e-9 || b.Pts[0].Dist(want) > 1e-9 {
		t.Fatalf("split point %+v / %+v want %+v", end, b.Pts[0], want)
	}
	if !a.Bounds().Union(b.Bounds()).Contains(Point{200, 100}) {
		t.Fatal("end lost")
	}
}

func TestContains(t *testing.T) {
	var p Path
	p.RoundRect(0, 0, 100, 50, 10)
	for _, c := range []struct {
		q  Point
		in bool
	}{{Point{50, 25}, true}, {Point{1, 1}, false}, {Point{5, 5}, true}, {Point{99, 49}, false}, {Point{101, 25}, false}} {
		if got := p.Contains(c.q, 0.05); got != c.in {
			t.Errorf("%v: got %v", c.q, got)
		}
	}
}

func TestArc(t *testing.T) {
	var p Path
	p.Arc(0, 0, 10, 0, math.Pi) // half circle through (0,10)
	m := p.Measure(0.005)
	if !near(m.Length(), math.Pi*10, 0.05) {
		t.Fatalf("arc length %v", m.Length())
	}
	if pt, _ := m.At(m.Length() / 2); pt.Dist(Point{0, 10}) > 0.05 {
		t.Fatalf("arc mid %+v", pt)
	}
}
