package gowt

import (
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestSectionRows(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		s := put[*SectionView](w, Section(GrowX(), Fill(Label("one")), Fill(Label("two"))))
		s.Add(Fill(Label("three")))
		kids := s.Unwrap().GetChildren()
		if len(kids) != 5 {
			t.Fatalf("3 rows need 2 separators, got %d children", len(kids))
		}
		for i, k := range kids {
			_, sep := k.Impl().(*swt.Label)
			if sep != (i%2 == 1) {
				t.Errorf("child %d: separator %v", i, sep)
			}
		}
	})
}

func TestCardStrokeColor(t *testing.T) {
	if _, ok := cardStrokeColor([]Option{GrowX()}); ok {
		t.Fatal("cardStrokeColor found a colour without CardBorder")
	}
	if got, ok := cardStrokeColor([]Option{CardBorder(RGB{R: 1, G: 2, B: 3})}); !ok || got.in(false) != (RGB{R: 1, G: 2, B: 3}) {
		t.Fatalf("cardStrokeColor = %v, %v", got, ok)
	}
}

func TestSectionFillColor(t *testing.T) {
	if _, ok := cardFillColor([]Option{GrowX()}); ok {
		t.Fatal("cardFillColor found a colour without SectionFill")
	}
	if got, ok := cardFillColor([]Option{SectionFill(RGB{R: 4, G: 5, B: 6})}); !ok || got != (RGB{R: 4, G: 5, B: 6}) {
		t.Fatalf("cardFillColor = %v, %v", got, ok)
	}
}
