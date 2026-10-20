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
		e.Gc.DrawLine(size.X/2, 0, size.X/2, size.Y)
		c.Dispose()
	}))
}
