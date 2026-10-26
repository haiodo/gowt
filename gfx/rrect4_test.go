package gfx

import "testing"

func TestRoundRectPath4(t *testing.T) {
	r := R(10, 20, 100, 40)
	if got, want := RoundRectPath4(r, [4]float64{6, 6, 6, 6}), RoundRectPath(r, 6); !pathsEqual(got, want) {
		t.Error("four equal radii differ from RoundRectPath")
	}
	p := RoundRectPath4(r, [4]float64{0, 30, 8, 100})
	if b := p.Bounds(); b != r {
		t.Errorf("bounds %v, want %v", b, r)
	}
	// top left is square, top right is clamped to 20 (half the height)
	if p.Pts[0] != (Point{10, 20}) || p.Pts[1] != (Point{90, 20}) {
		t.Errorf("top edge %v %v", p.Pts[0], p.Pts[1])
	}
	rec := NewRecorder()
	rec.FillRRect4(r, [4]float64{1, 2, 3, 4}, Black)
	rec.StrokeRRect4(r, [4]float64{1, 2, 3, 4}, StrokeStyle{Width: 2}, Black)
	if b := rec.Finish().Bounds; b.Min.X > 9 || b.Max.X < 111 {
		t.Errorf("recorded bounds %v", b)
	}
}

func pathsEqual(a, b *Path) bool {
	if len(a.Verbs) != len(b.Verbs) || len(a.Pts) != len(b.Pts) {
		return false
	}
	for i := range a.Verbs {
		if a.Verbs[i] != b.Verbs[i] {
			return false
		}
	}
	for i := range a.Pts {
		if a.Pts[i] != b.Pts[i] {
			return false
		}
	}
	return true
}

func TestEstimateTextSpacing(t *testing.T) {
	a := EstimateText{}.MeasureText("abcd", Font{Size: 10})
	b := EstimateText{}.MeasureText("abcd", Font{Size: 10, Spacing: 1.5})
	if b.Width-a.Width != 6 {
		t.Errorf("widths %v %v", a.Width, b.Width)
	}
}
