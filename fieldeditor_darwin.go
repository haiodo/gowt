package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// placeFieldEditor puts the caret of a styled field on the line of its text, where AppKit does not.
func placeFieldEditor(t *swt.Text) { cocoa.PlaceFieldEditor(t.View) }
