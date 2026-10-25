package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// tabSeparator is the WinUI divider: the text colour at 15% over the background.
func tabSeparator(_ *swt.Display, bg, fg RGB, _ bool) RGB { return mixRGB(bg, fg, 0.15) }

func tabAccent(d *swt.Display, _ bool) RGB {
	a := fluent.Current(d).Accent
	return RGB{a.R, a.G, a.B}
}
