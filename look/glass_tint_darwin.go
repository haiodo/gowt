//go:build darwin

package look

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func setGlassTint(c *swt.Composite, tint gowt.RGBA) {
	c.SetDataKeyValue(glassTintKey, tint)
	applyGlassTint(c)
}

// applyGlassTint puts the tint stored on c on its glass; it does nothing without a stored tint or
// glass. setGlass calls it after a glass reinstall, so the tint survives.
func applyGlassTint(c *swt.Composite) {
	tint, ok := c.GetDataKey(glassTintKey).(gowt.RGBA)
	if !ok {
		return
	}
	cocoa.SetBackdropTint(c.View, true,
		float64(tint.R)/255, float64(tint.G)/255, float64(tint.B)/255, float64(tint.A)/255)
}
