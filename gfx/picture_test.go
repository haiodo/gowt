package gfx

import (
	"math"
	"testing"
)

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestPictureRoundTrip(t *testing.T) {
	r := NewRecorder()
	r.Save()
	r.Translate(10, 20)
	r.Scale(2, 3)
	r.Rotate(0.5)
	r.Transform(Affine{1, 0, 0, 1, 5, 6})
	r.ClipRect(R(0, 0, 50, 40))
	r.ClipRRect(R(1, 2, 30, 20), 4)
	var tri Path
	tri.MoveTo(0, 0)
	tri.LineTo(10, 0)
	tri.QuadTo(12, 5, 10, 10)
	tri.CubicTo(8, 12, 2, 12, 0, 10)
	tri.Close()
	r.ClipPath(&tri)
	r.BeginLayer(0.5)
	r.FillPath(&tri, Color{1, 0, 0, 1})
	r.StrokePath(&tri, StrokeStyle{Width: 2, Cap: CapRound, Join: JoinBevel, MiterLimit: 4, Dash: []float64{3, 1}, DashPhase: 0.5}, Color{0, 1, 0, 0.5})
	g := NewLinearGradient(Point{0, 0}, Point{10, 0}, Stop{0, Color{1, 1, 1, 1}}, Stop{1, Color{0, 0, 0, 1}})
	r.FillGradient(&tri, g)
	r.StrokeGradient(&tri, StrokeStyle{Width: 1}, NewRadialGradient(Point{1, 1}, 0, Point{2, 2}, 9, Stop{0, Color{}}, Stop{1, Color{1, 1, 1, 1}}))
	r.FillShadowRRect(0, 0, 20, 10, 3, Color{1, 1, 1, 1}, Shadow{DX: 1, DY: 2, Blur: 4, Color: Color{0, 0, 0, 0.5}})
	r.Text("héllo", 3, 9, Font{Name: "Helvetica", Size: 12}, Color{0, 0, 1, 1})
	r.Text("again", 3, 19, Font{Name: "Helvetica", Size: 12}, Color{0, 0, 1, 1})
	img := NewRaster(2, 2)
	r.DrawImage(img, 0, 0, 4, 4)
	r.EndLayer()
	r.Restore()
	p := r.Finish()

	var kinds []OpKind
	var it Iter
	it.Reset(p)
	for it.Next() {
		o := it.Op()
		kinds = append(kinds, o.Kind)
		switch o.Kind {
		case OpTranslate:
			if o.A != 10 || o.B != 20 {
				t.Errorf("translate %v %v", o.A, o.B)
			}
		case OpClipRRect:
			if o.A != 1 || o.C != 30 || o.E != 4 {
				t.Errorf("clip rrect %+v", o)
			}
		case OpStroke:
			st := o.Style
			if st.Width != 2 || st.Cap != CapRound || st.Join != JoinBevel || st.MiterLimit != 4 || len(st.Dash) != 2 || st.Dash[0] != 3 || st.DashPhase != 0.5 {
				t.Errorf("stroke style %+v", *st)
			}
			if !approx(o.Color.A, 0.5, 0.01) || o.Color.G != 1 {
				t.Errorf("stroke color %+v", o.Color)
			}
			if len(o.Path.Verbs) != 5 || len(o.Path.Pts) != 7 {
				t.Errorf("path %v verbs %v pts", len(o.Path.Verbs), len(o.Path.Pts))
			}
		case OpFillGrad:
			if o.Grad.Kind != LinearGradient || len(o.Grad.Stops) != 2 || o.Grad.P1.X != 10 || o.Grad.Stops[1].Color.R != 0 {
				t.Errorf("gradient %+v", *o.Grad)
			}
		case OpStrokeGrad:
			if o.Grad.Kind != RadialGradient || o.Grad.R1 != 9 {
				t.Errorf("radial %+v", *o.Grad)
			}
		case OpShadowBox:
			if o.E != 3 || o.Shadow.DY != 2 || o.Shadow.Blur != 4 || !approx(o.Shadow.Color.A, 0.5, 0.01) {
				t.Errorf("shadow %+v", o)
			}
		case OpText:
			if o.Text != "héllo" && o.Text != "again" || o.Font.Size != 12 {
				t.Errorf("text %q %+v", o.Text, o.Font)
			}
		case OpImage:
			if o.Image != Image(img) || o.C != 4 {
				t.Errorf("image op %+v", o)
			}
		}
	}
	want := []OpKind{OpSave, OpTranslate, OpScale, OpRotate, OpTransform, OpClipRect, OpClipRRect, OpClipPath, OpBeginLayer,
		OpFill, OpStroke, OpFillGrad, OpStrokeGrad, OpShadowBox, OpText, OpText, OpImage, OpEndLayer, OpRestore}
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kind %d: %v, want %v", i, kinds[i], want[i])
		}
	}
}

func TestRecorderClosesLayers(t *testing.T) {
	r := NewRecorder()
	r.BeginLayer(0.3)
	r.BeginLayer(0.3)
	r.EndLayer()
	p := r.Finish()
	n := 0
	var it Iter
	it.Reset(p)
	for it.Next() {
		if it.Op().Kind == OpEndLayer {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d EndLayer, want 2", n)
	}
}

func TestBoundsClipAndStroke(t *testing.T) {
	var big Path
	big.Rect(0, 0, 1000, 1000)
	r := NewRecorder()
	r.Save()
	r.ClipRect(R(10, 10, 20, 20))
	r.FillPath(&big, Color{A: 1})
	r.Restore()
	if b := r.Finish().Bounds; b.Min != (Point{10, 10}) || b.Max != (Point{30, 30}) {
		t.Errorf("clipped bounds %+v", b)
	}

	r = NewRecorder()
	r.Translate(5, 5)
	r.Scale(2, 2)
	var box Path
	box.Rect(0, 0, 10, 10)
	// a square with right angles: the miter reaches sqrt(2) half widths
	r.StrokePath(&box, StrokeStyle{Width: 2, MiterLimit: 10}, Color{A: 1})
	b := r.Finish().Bounds
	want := 5 - 2*math.Sqrt2
	if !approx(b.Min.X, want, 0.01) || !approx(b.Max.X, 25+2*math.Sqrt2, 0.01) {
		t.Errorf("stroke bounds %+v, want min %v", b, want)
	}

	r = NewRecorder()
	r.StrokePath(&box, StrokeStyle{Width: 2, Join: JoinRound}, Color{A: 1})
	if b := r.Finish().Bounds; !approx(b.Min.X, -1, 1e-9) {
		t.Errorf("round join bounds %+v", b)
	}
}

type fixedText struct{}

func (fixedText) MeasureText(s string, f Font) TextMetrics { return TextMetrics{100, 10, 4} }

func TestTextBoundsFromMeasurer(t *testing.T) {
	r := NewRecorderWith(fixedText{})
	r.Text("x", 50, 50, Font{Name: "F", Size: 10}, Color{A: 1})
	b := r.Finish().Bounds
	if !approx(b.Min.Y, 40, 1e-9) || !approx(b.Max.Y, 54, 1e-9) || b.W() < 100 {
		t.Errorf("text bounds %+v", b)
	}
}

// A typical box of the synthetic document: shadow, border, dot, two labels. G-0 held 2.2 KB per
// node; the byte stream must stay well under that.
func TestPictureFootprint(t *testing.T) {
	r := NewRecorder()
	var rr, dot Path
	rr.RoundRect(0.5, 0.5, 139, 65, 8)
	dot.Ellipse(126, 14, 4, 4)
	r.FillShadowRRect(0, 0, 140, 66, 8, Color{1, 1, 1, 1}, Shadow{DY: 3, Blur: 8, Color: Color{A: 0.3}})
	r.StrokePath(&rr, StrokeStyle{Width: 1, MiterLimit: 10}, Color{0.7, 0.7, 0.7, 1})
	r.FillPath(&dot, Color{0.2, 0.5, 0.95, 1})
	r.Text("Button 1234", 10, 22, Font{Name: "Helvetica Neue", Size: 13}, Color{A: 1})
	r.Text("Primary action label", 10, 42, Font{Name: "Helvetica Neue", Size: 11}, Color{A: 1})
	p := r.Finish()
	t.Logf("stream %d B, picture %d B", p.Len(), p.Size())
	if p.Size() > 800 {
		t.Errorf("picture takes %d B", p.Size())
	}
}

func TestIterNoAllocs(t *testing.T) {
	r := NewRecorder()
	var rr Path
	rr.RoundRect(0, 0, 10, 10, 2)
	r.FillPath(&rr, Color{A: 1})
	r.StrokePath(&rr, StrokeStyle{Width: 1, Dash: []float64{2, 2}}, Color{A: 1})
	r.Text("a", 0, 0, Font{Name: "F", Size: 9}, Color{A: 1})
	p := r.Finish()
	var it Iter
	it.Reset(p)
	for it.Next() {
	}
	if n := testing.AllocsPerRun(10, func() {
		it.Reset(p)
		for it.Next() {
		}
	}); n != 0 {
		t.Errorf("%v allocs per replay", n)
	}
}
