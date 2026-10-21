package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func nativeInset(t *swt.Text, x, y int) { cocoa.SetTextViewInset(t.View, float64(x), float64(y)) }

func nativeContentHeight(t *swt.Text, width, x, y int) int {
	return int(cocoa.TextViewContentHeight(t.View, float64(width), float64(x), float64(y)))
}

func nativeWidth(t *swt.Text) int { return int(t.GetClientArea().Width) }
