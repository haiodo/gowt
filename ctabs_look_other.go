//go:build !darwin && !windows

package gowt

import "github.com/haiodo/gowt/swt"

// tabSeparator is the text colour at 15% over the background, as libadwaita's border.
func tabSeparator(_ *swt.Display, bg, fg RGB, _ bool) RGB { return mixRGB(bg, fg, 0.15) }

// tabAccent is the selection background, which GTK themes set to the accent.
func tabAccent(d *swt.Display, _ bool) RGB {
	return rgbOf(d.GetSystemColor(swt.COLOR_LIST_SELECTION).GetRGB())
}
