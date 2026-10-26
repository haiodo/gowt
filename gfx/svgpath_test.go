package gfx

import (
	"math"
	"testing"
)

func TestSVGPath(t *testing.T) {
	p, err := ParseSVGPath("M7 1a6 6 0 0 1 0.55 0.03m2.98 1.12l-1-1h2v3z")
	if err != nil {
		t.Fatal(err)
	}
	b := p.Bounds()
	if b.Min.X < 7 || b.Max.X > 12 || b.Min.Y < 0.9 {
		t.Errorf("bounds %v", b)
	}
	// a half circle of radius 5 from (0,0) to (10,0), sweeping through the lower side
	p, err = ParseSVGPath("M0 0A5 5 0 0 0 10 0")
	if err != nil {
		t.Fatal(err)
	}
	if b := p.Bounds(); math.Abs(b.Max.Y-5) > 0.05 || math.Abs(b.Min.Y) > 0.05 {
		t.Errorf("half circle bounds %v", b)
	}
	if _, err := ParseSVGPath("M1 x"); err == nil {
		t.Error("garbage accepted")
	}
}
