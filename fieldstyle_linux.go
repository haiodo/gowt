package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// setFieldFill gives the GtkEntry the fill of FieldFill and its radius, without the theme's frame.
func setFieldFill(t *swt.Text, radius int) {
	c := t.GetBackground()
	gtk.SetEntryFill(t.Handle, int32(c.GetRed()), int32(c.GetGreen()), int32(c.GetBlue()), radius)
	releaseEntryLook(t)
}

// setFieldStroke draws the border of the entry fill.
func setFieldStroke(t *swt.Text, r, g, b, a uint8) {
	gtk.SetEntryStroke(t.Handle, int32(r), int32(g), int32(b), int32(a))
	releaseEntryLook(t)
}

// setFieldInset widens the entry's padding by x points.
func setFieldInset(t *swt.Text, x int) {
	gtk.SetEntryInset(t.Handle, x)
	releaseEntryLook(t)
}
