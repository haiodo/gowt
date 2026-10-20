package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// cardColor is quaternarySystemFillColor over the window background, flattened to an opaque colour:
// SWT draws the background of every label and panel on the card, and an alpha colour would add up.
func cardColor(*swt.Display) *swt.Color {
	r, g, b := cocoa.GroupedFillRGB()
	return swt.NewColorRedGreenBlue(r, g, b)
}

// roundCard clips the card, rows included, to its rounded rectangle, so on a glass pane the corners show
// the glass rather than a fill in the parent's colour.
func roundCard(c *swt.Composite) bool { return cocoa.RoundCorners(c.View, sectionRadius) }
