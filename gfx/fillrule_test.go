package gfx

import (
	"bytes"
	"strings"
	"testing"
)

func ring() *Path {
	p := &Path{EvenOdd: true}
	p.Rect(0, 0, 40, 40)
	p.Rect(10, 10, 20, 20)
	return p
}

func TestFillRuleRoundTrip(t *testing.T) {
	r := NewRecorder()
	r.FillPath(ring(), Color{A: 1})
	nz := ring()
	nz.EvenOdd = false
	r.FillPath(nz, Color{A: 1})
	r.ClipPath(ring())
	var got []bool
	var it Iter
	it.Reset(r.Finish())
	for it.Next() {
		got = append(got, it.Op().Path.EvenOdd)
		if len(it.Op().Path.Verbs) != 10 {
			t.Errorf("%d verbs", len(it.Op().Path.Verbs))
		}
	}
	if len(got) != 3 || !got[0] || got[1] || !got[2] {
		t.Errorf("even-odd flags %v", got)
	}
	if !ring().Clone().EvenOdd || !ring().Transform(Identity()).EvenOdd {
		t.Error("Clone or Transform dropped the fill rule")
	}
}

func TestSVGFillRule(t *testing.T) {
	r := NewRecorder()
	r.FillPath(ring(), Color{A: 1})
	r.ClipPath(ring())
	r.FillPath(ring(), Color{A: 1})
	nz := ring()
	nz.EvenOdd = false
	r.FillPath(nz, Color{A: 1})
	var b bytes.Buffer
	if err := r.Finish().WriteSVG(&b, 1); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	wellFormed(t, s)
	if n := strings.Count(s, `fill-rule="evenodd"`); n != 2 {
		t.Errorf("%d fill-rule attributes, want 2:\n%s", n, s)
	}
	if !strings.Contains(s, `clip-rule="evenodd"`) {
		t.Errorf("no clip-rule:\n%s", s)
	}
}
