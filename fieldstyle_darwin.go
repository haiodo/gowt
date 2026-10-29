package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// setFieldFill gives a single-line field the rounded fill of FieldFill; the background colour is on
// the field already.
func setFieldFill(t *swt.Text, radius int) { cocoa.SetFieldFill(t.View, float64(radius)) }

// setFieldStroke draws the border of a field fill.
func setFieldStroke(t *swt.Text, r, g, b, a uint8) {
	cocoa.SetFieldStroke(t.View, float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255, 1)
}

// setFieldInset moves the text of a single-line field in by x points.
func setFieldInset(t *swt.Text, x int) { cocoa.SetFieldInset(t.View, float64(x)) }
