package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func overRGB(c [4]float64, bg RGB) RGB {
	m := func(f float64, b uint8) uint8 { return uint8(255*f*c[3] + float64(b)*(1-c[3]) + 0.5) }
	return RGB{m(c[0], bg.R), m(c[1], bg.G), m(c[2], bg.B)}
}

// tabSeparator is NSColor.separatorColor over the tab background.
func tabSeparator(_ *swt.Display, bg, _ RGB, dark bool) RGB {
	return overRGB(cocoa.SeparatorRGBA(dark), bg)
}

// glassSeparator is NSColor.separatorColor with its own alpha, for drawing over glass.
func glassSeparator(_ RGB, dark bool) (RGB, int32) {
	c := cocoa.SeparatorRGBA(dark)
	b := func(f float64) uint8 { return uint8(255*f + 0.5) }
	return RGB{b(c[0]), b(c[1]), b(c[2])}, int32(b(c[3]))
}

func tabAccent(_ *swt.Display, dark bool) RGB {
	return overRGB(cocoa.AccentRGBA(dark), RGB{})
}

// tabRadius is the corner radius of the selected tab, tabDividers whether unselected neighbours have a divider.
const (
	tabRadius   = 0
	tabDividers = true
)
