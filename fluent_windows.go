package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// An edit control cannot hide its scroll bar when there is nothing to scroll, so Multiline asks for none
// and Scrollbars() brings the always-visible one back.
const multilineScroll = 0

// textHeightDips is the height of a WinUI 3 TextBox.
const textHeightDips = 32

// fluentText gives a Text the Windows 11 text box look; noBorder (NoBorder) keeps the plain edit.
func fluentText(t *TextView, noBorder bool) {
	if noBorder {
		return
	}
	fluent.Text(t.t)
	if t.multi() {
		if t.inset == [2]int{} {
			t.SetInset(0, 5)
		}
		return
	}
	if _, h, ok := sizeHints(&t.t.Control); ok && *h == swt.DEFAULT {
		height := int32(textHeightDips)
		if t.aff != nil && t.aff.small {
			height = smallTextHeightDips
		}
		*h = max(height, preferred(&t.t.Control).Y)
	}
}

// fluentBorder replaces the 3D edge of a container, list or table with a flat hairline.
func fluentBorder(c *swt.Control) {
	if c.GetStyle()&swt.BORDER == 0 {
		return
	}
	switch c.GetName() {
	case "Composite", "ScrolledComposite", "Table", "Tree", "List", "StyledText":
		fluent.FlatBorder(c)
	}
}

// fluentCard is the colour of a Section card.
func fluentCard(d *swt.Display) *swt.Color {
	c := fluent.Current(d).Card()
	return swt.NewColorRedGreenBlue(int32(c.R), int32(c.G), int32(c.B))
}

func fluentTable(t *swt.Table) { fluent.Table(t, rowHeight()) }

// rowHeight is Spacing.RowHeight once App.SetDensity chose a density, else 0 for the native height.
func rowHeight() int {
	if ChosenDensity().Legacy() {
		return 0
	}
	return Spacing.RowHeight
}

func fluentList(l *swt.List) { fluent.List(l, rowHeight()) }

func fluentTree(t *swt.Tree) { fluent.Tree(t, rowHeight()) }

// fluentStyledText makes the StyledText background its own. Under the Fluent shell (INHERIT_DEFAULT) SWT
// fills the space below the last line with the parent's background while the lines get the list one.
// Ceiling: setEnabled(false) no longer greys it, as for any StyledText given a background.
func fluentStyledText(t *swt.StyledText) { t.SetBackgroundWithColor(t.GetBackground()) }

// fluentShell hides focus rectangles until the keyboard is used and sets the Fluent window background.
func fluentShell(s *swt.Shell) {
	fluent.Shell(s)
	fluent.Background(&s.Composite)
}

// fluentScroll gives the content of a Scroll the window background: a ScrolledComposite does not pass
// the inherited one on to its content.
func fluentScroll(body *swt.Composite) { fluent.Background(body) }

// controlHeight gives a push or toggle button, drop-down Combo, Spinner and date field the height of
// Spacing.ControlHeight, like the Text; without App.SetDensity the heights stay the native ones.
func controlHeight(opts []Option) []Option {
	return append(opts[:len(opts):len(opts)], lateHook(func(w Widget) error {
		if v, ok := w.(*SpinnerView); ok {
			fluent.Spinner(v.s)
			if wh, _, ok := sizeHints(&v.s.Control); ok && *wh == swt.DEFAULT {
				*wh = int32(preferred(&v.s.Control).X) + fluent.SpinnerExtra
			}
		}
		if ChosenDensity().Legacy() {
			return nil
		}
		c := w.control()
		switch v := w.(type) {
		case *ComboView:
			if v.c.GetStyle()&swt.DROP_DOWN != 0 {
				fluent.ComboHeight(v.c, Spacing.ControlHeight)
			}
			return nil
		case *ButtonView:
			if v.b.GetStyle()&(swt.PUSH|swt.TOGGLE) == 0 {
				return nil
			}
		case *DateTimeView:
			if v.d.GetStyle()&swt.CALENDAR != 0 {
				return nil
			}
		}
		if _, h, ok := sizeHints(c); ok && *h == swt.DEFAULT {
			*h = int32(max(Spacing.ControlHeight, int(preferred(c).Y)))
		}
		return nil
	}))
}

// fluentRange gives Progress and Scale the Windows 11 bar and slider.
func fluentRange(opts []Option) []Option {
	return append(opts[:len(opts):len(opts)], lateHook(func(w Widget) error {
		switch v := w.(type) {
		case *ProgressView:
			fluent.Progress(v.b)
		case *ScaleView:
			fluent.Scale(v.s)
		}
		return nil
	}))
}
