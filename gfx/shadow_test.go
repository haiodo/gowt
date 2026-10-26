package gfx

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func near6(a, b float64) bool { return near(a, b, 1e-6) }

func TestArcToQuarter(t *testing.T) {
	// canvas: moveTo(0,0); arcTo(100,0,100,100,50) is a line to (50,0) and a quarter circle about (50,50)
	var p Path
	p.MoveTo(0, 0)
	p.ArcTo(100, 0, 100, 100, 50)
	k := 50 * kappa
	want := []Point{{0, 0}, {50, 0}, {50 + k, 0}, {100, 50 - k}, {100, 50}}
	if len(p.Pts) != len(want) {
		t.Fatalf("points %v, want %v", p.Pts, want)
	}
	for i, q := range want {
		if !near6(p.Pts[i].X, q.X) || !near6(p.Pts[i].Y, q.Y) {
			t.Errorf("point %d is %v, want %v", i, p.Pts[i], q)
		}
	}
	if p.Verbs[1] != LineTo || p.Verbs[2] != CubicTo {
		t.Errorf("verbs %v", p.Verbs)
	}
}

func TestArcToObtuse(t *testing.T) {
	// canvas: moveTo(0,0); arcTo(100,0,200,100,10): tangent points (95.858,0) and (102.929,2.929),
	// center (95.858,10)
	var p Path
	p.MoveTo(0, 0)
	p.ArcTo(100, 0, 200, 100, 10)
	tn := 10 * math.Tan(math.Pi/8)
	c := Point{100 - tn, 10}
	if q := p.Pts[1]; !near6(q.X, c.X) || !near6(q.Y, 0) {
		t.Errorf("first tangent point %v", q)
	}
	end := p.Pts[len(p.Pts)-1]
	if !near6(end.X, 100+tn*math.Sqrt2/2) || !near6(end.Y, tn*math.Sqrt2/2) {
		t.Errorf("second tangent point %v", end)
	}
	if d := end.Dist(c); !near6(d, 10) {
		t.Errorf("end is %v from the center", d)
	}
	if len(p.Verbs) != 3 { // one cubic: the sweep is 45 degrees
		t.Errorf("verbs %v", p.Verbs)
	}
}

func TestArcToCounterClockwise(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.ArcTo(100, 0, 100, -100, 50) // turns up on screen
	end := p.Pts[len(p.Pts)-1]
	if !near6(end.X, 100) || !near6(end.Y, -50) {
		t.Errorf("end %v, want (100,-50)", end)
	}
	if c1 := p.Pts[2]; !near6(c1.Y, 0) || !near6(c1.X, 50+50*kappa) {
		t.Errorf("first control %v", c1)
	}
}

func TestArcToDegenerate(t *testing.T) {
	cases := []struct {
		name              string
		x1, y1, x2, y2, r float64
	}{
		{"zero radius", 100, 0, 100, 100, 0},
		{"same point", 0, 0, 100, 100, 5},
		{"collinear", 50, 0, 100, 0, 5},
		{"back on itself", 100, 0, 0, 0, 5},
	}
	for _, c := range cases {
		var p Path
		p.MoveTo(0, 0)
		p.ArcTo(c.x1, c.y1, c.x2, c.y2, c.r)
		if len(p.Verbs) != 2 || p.Verbs[1] != LineTo || p.Pts[1] != (Point{c.x1, c.y1}) {
			t.Errorf("%s: %v %v", c.name, p.Verbs, p.Pts)
		}
	}
	var p Path
	p.ArcTo(10, 20, 30, 40, 5)
	if len(p.Verbs) != 1 || p.Verbs[0] != MoveTo || p.Pts[0] != (Point{10, 20}) {
		t.Errorf("without a current point: %v %v", p.Verbs, p.Pts)
	}
}

func TestArcToAfterClose(t *testing.T) {
	var p Path
	p.MoveTo(10, 10)
	p.LineTo(50, 10)
	p.Close()
	p.ArcTo(110, 10, 110, 60, 20) // the current point is the subpath start (10,10)
	if q := p.Pts[len(p.Pts)-1]; !near6(q.X, 110) || !near6(q.Y, 30) {
		t.Errorf("end %v, want (110,30)", q)
	}
}

func TestShadowOpsRoundTrip(t *testing.T) {
	sh := Shadow{DX: 1, DY: 2, Blur: 6, Color: Color{0, 0, 0, 0.5}}
	r := NewRecorder()
	r.FillShadowPath(EllipsePath(R(0, 0, 20, 10)), Color{1, 0, 0, 1}, sh)
	r.StrokeShadowPath(PolygonPath(Point{0, 30}, Point{40, 30}), StrokeStyle{Width: 2}, Color{0, 1, 0, 1}, sh)
	r.TextShadow("hi", 0, 50, Font{Size: 10}, Color{0, 0, 1, 1}, sh)
	pic := r.Finish()
	var it Iter
	it.Reset(pic)
	var kinds []OpKind
	for it.Next() {
		o := it.Op()
		kinds = append(kinds, o.Kind)
		if o.Shadow.DX != 1 || o.Shadow.DY != 2 || o.Shadow.Blur != 6 || o.Shadow.Color.A < 0.49 {
			t.Errorf("op %d shadow %+v", o.Kind, o.Shadow)
		}
		if o.Kind == OpTextShadow && (o.Text != "hi" || o.Color.B != 1) {
			t.Errorf("text op %+v", o)
		}
	}
	if len(kinds) != 3 || kinds[0] != OpFillShadow || kinds[1] != OpStrokeShadow || kinds[2] != OpTextShadow {
		t.Fatalf("kinds %v", kinds)
	}
	if b := pic.Bounds; b.Min.X > -9 || b.Min.Y > -9 || b.Max.X < 49 {
		t.Errorf("bounds %v do not hold the blur", b)
	}
	var out bytes.Buffer
	if err := pic.WriteSVG(&out, 1); err != nil {
		t.Fatal(err)
	}
	wellFormed(t, out.String())
	if n := strings.Count(out.String(), "<filter"); n != 3 {
		t.Errorf("%d filters in\n%s", n, out.String())
	}
}

func TestSVGBackgroundAndDefaultFamily(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 10, 10, 50, 50)
	s.Update(func(tx *Tx) {
		r := NewRecorder()
		r.Text("x", 0, 10, Font{Size: 10}, Color{A: 1})
		tx.SetPicture(f, r.Finish())
	})
	s.Commit()
	var b bytes.Buffer
	if err := WriteSubtreeSVGWith(&b, s, f, R(0, 0, 80, 70), 1, SVGOptions{Background: Color{1, 1, 1, 1}}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	wellFormed(t, out)
	for _, want := range []string{`<rect x="0" y="0" width="80" height="70" fill="rgb(255,255,255)"`, `font-family="Helvetica Neue, Helvetica, Arial, sans-serif"`} {
		if !strings.Contains(out, want) {
			t.Errorf("SVG lacks %q:\n%s", want, out)
		}
	}
	b.Reset()
	if err := WriteSubtreeSVG(&b, s, f, R(0, 0, 80, 70), 1); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), `width="80" height="70" fill`) {
		t.Error("no background expected")
	}
}
