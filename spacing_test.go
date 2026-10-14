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
	before := Grid{Columns: 2}.Spaced()
	Spacing.SM, Spacing.PageMargin = 6, 20
	after := Grid{Columns: 2}.Spaced().Padded()
	if before.Spacing != old.SM || after.Spacing != 6 || after.Margin != 20 {
		t.Fatalf("before %+v after %+v", before, after)
	}
}

func TestSpacedPadded(t *testing.T) {
	if g := (Grid{Columns: 2}); g.Margin != 0 || g.Spacing != 0 {
		t.Fatal("zero must stay none")
	}
	g := Grid{Columns: 3, Margin: 1}.Spaced().Padded()
	if g.Columns != 3 || g.Spacing != Spacing.SM || g.Margin != Spacing.PageMargin {
		t.Fatalf("grid %+v", g)
	}
	r := Row{Vertical: true}.Spaced().Padded()
	if !r.Vertical || r.Spacing != Spacing.SM || r.Margin != Spacing.PageMargin {
		t.Fatalf("row %+v", r)
	}
	f := Form{}.Spaced().Padded()
	if f.Spacing != Spacing.SM || f.Margin != Spacing.PageMargin {
		t.Fatalf("form %+v", f)
	}
	if s := (Stack{}).Padded(); s.Margin != Spacing.PageMargin {
		t.Fatalf("stack %+v", s)
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
