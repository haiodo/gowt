package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func multiHint(t *swt.Text, s string) { cocoa.SetTextViewPlaceholder(t.View, s) }
