package gowt

import (
	"errors"
	"strings"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func placed(t *testing.T, layout string, opts ...Option) any {
	t.Helper()
	var pl placement
	for _, o := range opts {
		o.(place)(&pl)
	}
	d, err := pl.data("Button", layout)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func misplacedErr(layout string, opts ...Option) error {
	var pl placement
	for _, o := range opts {
		o.(place)(&pl)
	}
	_, err := pl.data("Button", layout)
	return err
}

func TestPlacementComposes(t *testing.T) {
	d := placed(t, "Grid", GrowX(), Span(2), Size(0, 40)).(*swt.GridData)
	if d.HorizontalAlignment != swt.FILL || !d.GrabExcessHorizontalSpace || d.HorizontalSpan != 2 || d.HeightHint != 40 || d.WidthHint != swt.DEFAULT {
		t.Errorf("GrowX, Span(2), Size(0, 40): %+v", d)
	}
	if d.VerticalAlignment != swt.BEGINNING || d.GrabExcessVerticalSpace {
		t.Errorf("GrowX must leave the vertical axis alone: %+v", d)
	}
	d = placed(t, "Grid", Grow(), Place(AlignEnd, AlignCenter)).(*swt.GridData)
	if d.HorizontalAlignment != swt.END || d.VerticalAlignment != swt.CENTER || !d.GrabExcessHorizontalSpace || !d.GrabExcessVerticalSpace {
		t.Errorf("Place after Grow must change only the alignment: %+v", d)
	}
	d = placed(t, "Grid", Cell(GridCell{SpanX: 3, Width: 100}), GrowY()).(*swt.GridData)
	if d.HorizontalSpan != 3 || d.WidthHint != 100 || d.VerticalAlignment != swt.FILL || !d.GrabExcessVerticalSpace {
		t.Errorf("GrowY after Cell must keep the cell: %+v", d)
	}
	if r := placed(t, "Flow", Size(120, 0)).(*swt.RowData); r.Width != 120 || r.Height != swt.DEFAULT {
		t.Errorf("Size in a Flow: %+v", r)
	}
	if f := placed(t, "Form", Anchor(FormCell{Left: Percent(0, 5)}), Size(80, 0)).(*swt.FormData); f.Width != 80 || f.Left == nil {
		t.Errorf("Size after Anchor in a Form: %+v", f)
	}
	for _, l := range []string{"VBox", "HBox"} {
		if _, ok := placed(t, l, GrowX()).(*swt.GridData); !ok {
			t.Errorf("GrowX in a %s must give GridData", l)
		}
	}
}

func TestPlacementUnderForeignLayout(t *testing.T) {
	cases := []struct {
		layout string
		opts   []Option
		want   string
	}{
		{"Flow", []Option{Cell(GridCell{})}, "Cell on Button needs a VBox, HBox or Grid parent, not Flow"},
		{"Fill", []Option{Size(10, 10), GrowX()}, "GrowX on Button needs a VBox, HBox or Grid parent, not Fill"},
		{"Grid", []Option{Anchor(FormCell{})}, "Anchor on Button needs a Form parent, not Grid"},
		{"Pages", []Option{Size(10, 0)}, "Size on Button needs a VBox, HBox, Grid, Flow or Form parent, not Pages"},
		{"", []Option{GrowX()}, "GrowX on Button needs a VBox, HBox or Grid parent, not a parent without a layout"},
	}
	for _, c := range cases {
		err := misplacedErr(c.layout, c.opts...)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.layout, err, c.want)
		}
	}
}

func TestDispatchReportsAndContinues(t *testing.T) {
	var got []error
	a := &App{}
	a.OnError(func(err error) { got = append(got, err) })
	if !a.dispatch(func() bool { panic("boom") }) {
		t.Error("a step that panicked must count as busy so the loop reads the next event")
	}
	a.dispatch(func() bool { panic(swt.NewSWTExceptionCode(swt.ERROR_WIDGET_DISPOSED)) })
	if a.dispatch(func() bool { return false }) {
		t.Error("an idle step must stay idle")
	}
	var se *swt.SWTException
	if len(got) != 2 || got[0].Error() != "gowt: panic: boom" || !errors.As(got[1], &se) {
		t.Errorf("OnError got %v", got)
	}
}

func TestIgnoreDisposed(t *testing.T) {
	func() {
		defer ignoreDisposed()
		swt.Error(swt.ERROR_DEVICE_DISPOSED)
	}()
	defer func() {
		if recover() == nil {
			t.Error("an error other than a disposed display must pass")
		}
	}()
	func() {
		defer ignoreDisposed()
		swt.Error(swt.ERROR_WIDGET_DISPOSED)
	}()
}
