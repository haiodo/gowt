package gowt

import (
	"testing"
	"time"
)

func pumpUntil(app *App, until func() bool) {
	for i := 0; i < 50 && !until(); i++ {
		for app.Unwrap().ReadAndDispatch() {
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestFitColumns(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		tb := put[*TableView](w, Table(Size(400, 150)))
		fit := tb.Column("Subject", 10, Fit())
		fixed := tb.Column("Fixed", 50)
		tb.Row("A rather long subject line that does not fit in ten points", "x")
		pumpUntil(app, func() bool { return fit.Unwrap().Impl().(interface{ GetWidth() int32 }).GetWidth() > 100 })
		width := func(c *Column) int32 { return c.Unwrap().Impl().(interface{ GetWidth() int32 }).GetWidth() }
		if width(fit) <= 100 {
			t.Errorf("Fit column width %d, want the text width", width(fit))
		}
		if width(fixed) != 50 {
			t.Errorf("column without Fit changed to %d", width(fixed))
		}
		tb.Row("An even longer subject line than the first one, to widen the column again")
		first := width(fit)
		pumpUntil(app, func() bool { return width(fit) > first })
		if width(fit) <= first {
			t.Errorf("Fit column did not grow with a longer row: %d then %d", first, width(fit))
		}
		tb.FitColumns()
		if width(fixed) < 20 {
			t.Errorf("FitColumns left the second column %d wide", width(fixed))
		}

		tr := put[*TreeView](w, Tree(Size(400, 150)))
		tc := tr.Column("Name", 10, Fit())
		tr.Node("A tree node with a long name that needs more than ten points")
		pumpUntil(app, func() bool { return width(tc) > 100 })
		if width(tc) <= 100 {
			t.Errorf("Tree Fit column width %d", width(tc))
		}
		put[*LabelView](w, Label("l", Fit()))
		if len(*errs) != 1 {
			t.Errorf("Fit on a Label reported %q", *errs)
		}
	})
}
