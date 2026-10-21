package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

func nativeInset(t *swt.Text, x, y int) { gtk.SetTextViewMargins(t.Handle, int32(x), int32(y)) }

func nativeContentHeight(t *swt.Text, width, x, y int) int {
	return int(gtk.TextViewContentHeight(t.Handle, t.GetText(), int32(width-2*x))) + 2*y
}

// nativeWidth is the width the text wraps in: a GtkTextView reports its own, wider than the field, as client area.
func nativeWidth(t *swt.Text) int { return int(t.GetSize().X) }
