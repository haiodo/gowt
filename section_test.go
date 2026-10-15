package gowt

import (
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestSectionRows(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		w.SetLayout(Grid{Columns: 1, Margin: 10})
		s := w.Section(Cell(GridCell{GrowX: true, Align: AlignFill}))
		s.Row().SetLayout(Fill{}).Label("one")
		s.Row().SetLayout(Fill{}).Label("two")
		s.Row().SetLayout(Fill{}).Label("three")
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
