//go:build !darwin && !windows

package gowt

import "github.com/haiodo/gowt/swt"

// tabSeparator is the text colour at 15% over the background, as libadwaita's border.
func tabSeparator(_ *swt.Display, bg, fg RGB, _ bool) RGB { return mixRGB(bg, fg, 0.15) }

// glassSeparator is tabSeparator as a colour and alpha, for drawing over a background not known here.
func glassSeparator(fg RGB, _ bool) (RGB, int32) { return fg, 38 }

// tabAccent is the selection background, which GTK themes set to the accent.
func tabAccent(d *swt.Display, _ bool) RGB {
	return rgbOf(d.GetSystemColor(swt.COLOR_LIST_SELECTION).GetRGB())
}

// tabRadius is the corner radius of the selected tab, tabDividers whether unselected neighbours have a divider.
const (
	tabRadius   = 6
	tabDividers = false
)
