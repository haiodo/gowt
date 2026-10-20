package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// paintHint draws the hint in AppKit's placeholder colour and the view's font. SWT sends Paint
// after the NSTextView has drawn itself, so the hint lies over its background.
func paintHint(t *swt.Text) {
	t.AddPaintListener(painter(func(*swt.PaintEvent) {
		if t.GetCharCount() == 0 {
			cocoa.DrawTextViewPlaceholder(t.View, t.GetMessage())
		}
	}))
}
