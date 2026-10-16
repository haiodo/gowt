package gowt

import "testing"

func TestSpacingDefaults(t *testing.T) {
	s := defaultSpacing()
	if s.XS != 4 || s.SM != 8 || s.MD != 12 || s.LG != 16 || s.XL != 32 {
		t.Fatalf("tokens: %+v", s)
	}
	if s.PageMargin != s.MD && s.PageMargin != s.LG {
		t.Fatalf("page margin %d is neither MD nor LG", s.PageMargin)
	}
}

func TestSpacingTuning(t *testing.T) {
	old := Spacing
	defer func() { Spacing = old }()
	bg, _ := spacingOf(nil, 0)
	Spacing.SM, Spacing.PageMargin = 6, 20
	ag, ap := spacingOf(nil, Spacing.PageMargin)
	if bg != old.SM || ag != 6 || ap != 20 {
		t.Fatalf("gap before %d, after %d, padding %d", bg, ag, ap)
	}
}

func TestGolden(t *testing.T) {
	for _, tc := range [][3]int{{0, 0, 0}, {1000, 382, 618}, {1130, 432, 698}} {
		s, l := Golden(tc[0])
		if s != tc[1] || l != tc[2] {
			t.Errorf("Golden(%d) = %d, %d", tc[0], s, l)
		}
	}
}
