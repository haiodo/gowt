package gowt

import "github.com/haiodo/gowt/internal/cocoa"

func systemDark() bool { return cocoa.EffectiveAppearanceDark() }

// secondaryText is read from AppKit: SWT's disabled foreground here is a selection background, and
// its system colours drop the alpha of secondaryLabelColor.
func secondaryText() RGB {
	r, g, b := cocoa.SecondaryLabelRGB()
	return RGB{uint8(r), uint8(g), uint8(b)}
}
