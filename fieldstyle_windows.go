package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// setFieldFill gives the WinUI frame of a Text the fill of FieldFill and its radius.
func setFieldFill(t *swt.Text, radius int) { fluent.SetFieldFill(t, radius) }

// setFieldStroke draws the border of a field fill in the WinUI frame.
func setFieldStroke(t *swt.Text, r, g, b, a uint8) { fluent.SetFieldStroke(t, r, g, b, a) }

// setFieldInset widens the text padding of a framed Text by x points.
func setFieldInset(t *swt.Text, x int) { fluent.SetFieldInset(t, x) }
