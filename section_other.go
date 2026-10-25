//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// cardColor is 6% of the way from the window colour toward the text colour; Windows 11 has its own card fill.
func cardColor(d *swt.Display) *swt.Color {
	if c := fluentCard(d); c != nil {
		return c
	}
	bg, fg := d.GetSystemColor(swt.COLOR_WIDGET_BACKGROUND), d.GetSystemColor(swt.COLOR_WIDGET_FOREGROUND)
	return swt.NewColorRedGreenBlue(mix(bg.GetRed(), fg.GetRed()), mix(bg.GetGreen(), fg.GetGreen()), mix(bg.GetBlue(), fg.GetBlue()))
}

// mix moves 6% of the way from a to b.
func mix(a, b int32) int32 { return a + (b-a)*6/100 }

// roundCard does nothing here: the card's corners are painted in the parent's colour.
func roundCard(*swt.Composite) bool { return false }
