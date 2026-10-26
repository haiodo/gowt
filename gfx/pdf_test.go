package gfx

import (
	"math"
	"testing"
)

func TestPDFLayout(t *testing.T) {
	area := R(100, 200, 400, 200)
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

	l, err := PDFOptions{Margin: 10, Scale: 0.5}.Layout(area)
	if err != nil || !near(l.W, 220) || !near(l.H, 120) || l.Scale != 0.5 {
		t.Errorf("sized to content: %+v, %v", l, err)
	}
	if o := l.Origin; !near(o.X, 100-20) || !near(o.Y, 200-20) { // the margin shows 20 scene units
		t.Errorf("origin %v", o)
	}

	// a 500 wide box for a 400 wide area, 700 high for 200: the width allows 1.25
	l, err = PDFOptions{PageW: 600, PageH: 800, Margin: 50}.Layout(area)
	if err != nil || !near(l.Scale, 1.25) {
		t.Errorf("fit scale %v, want 1.25 (%v)", l.Scale, err)
	}
	// the area, scaled, is centred on the page
	c := Point{(area.Min.X + 200 - l.Origin.X) * l.Scale, (area.Min.Y + 100 - l.Origin.Y) * l.Scale}
	if !near(c.X, 300) || !near(c.Y, 400) {
		t.Errorf("area centre lands at %v, want the page centre (300, 400)", c)
	}

	l, _ = PDFOptions{PageW: 600, PageH: 800, Margin: 50, Scale: 1}.Layout(area)
	if !near(l.Scale, 1) {
		t.Errorf("a Scale cap of 1 gave %v", l.Scale)
	}

	for _, bad := range []PDFOptions{{PageW: 600}, {PageW: 100, PageH: 100, Margin: 60}} {
		if _, err := bad.Layout(area); err == nil {
			t.Errorf("%+v is accepted", bad)
		}
	}
	if _, err := (PDFOptions{}).Layout(EmptyRect()); err == nil {
		t.Error("an empty area is accepted")
	}
}
