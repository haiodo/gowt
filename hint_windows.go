package gowt

import (
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

const emGetRect = 0xb2 // EM_GETRECT

// paintOver runs draw in the Paint event, which an edit control sends after its own WM_PAINT.
func paintOver(t *swt.Text, draw func(g *swt.GC)) {
	t.AddPaintListener(painter(func(e *swt.PaintEvent) { draw(e.Gc) }))
}

// hintColor is COLOR_GRAYTEXT, the colour of a cue banner.
func hintColor(t *swt.Text) *swt.Color {
	return t.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_DISABLED_FOREGROUND)
}

// hintOrigin is the top left of the edit's formatting rectangle. Text.GetCaretLocation is not used:
// on an empty edit it inserts and removes a character to measure.
func hintOrigin(t *swt.Text) (x, y int32) {
	r := win32.NewRECT()
	win32.OSSendMessageOverload13(t.Handle, emGetRect, 0, r)
	z := t.GetAutoscalingZoom()
	return swt.DPIUtilPixelToPoint(r.Left, z), swt.DPIUtilPixelToPoint(r.Top, z)
}
