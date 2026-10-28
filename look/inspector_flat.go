//go:build !darwin

package look

import "github.com/haiodo/gowt/swt"

// setInspectorChrome draws a hairline down the middle of the sash gap, where a GTK or Windows split
// view shows its divider: inspector and content both have the window colour and would run together.
func setInspectorChrome(_ *swt.Shell, _ *swt.Composite, sash *swt.Control) {
	sash.AddPaintListener(painter(func(e *swt.PaintEvent) {
		size := sash.GetSize()
		bg := sash.GetBackground().GetRGB()
		fg := barForeground(*sash.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND).GetRGB(), *bg)
		c := swt.NewColorRgb(blend(*bg, fg, 12))
		e.Gc.SetForeground(c)
		if size.X > size.Y {
			e.Gc.DrawLine(0, size.Y/2, size.X, size.Y/2)
		} else {
			e.Gc.DrawLine(size.X/2, 0, size.X/2, size.Y)
		}
		c.Dispose()
	}))
}

// setBottomChrome also gives the panel the SetMaterial background: on GTK a NoBackground StyledText
// drew white on a panel without a background of its own.
func setBottomChrome(s *swt.Shell, pb *swt.Composite, sash *swt.Control) {
	setMaterial(pb, true)
	setInspectorChrome(s, pb, sash)
}

// syncBottom does nothing: no sidebar floats here, and the panel paints its own background.
func syncBottom(*swt.Composite, [4]int32, *swt.Composite) {}
