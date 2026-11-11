//go:build !darwin

package look

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// setGlassTint stores the tint so a later darwin build of the app tree keeps it; other systems
// have no glass tint.
func setGlassTint(c *swt.Composite, tint gowt.RGBA) {
	c.SetDataKeyValue(glassTintKey, tint)
}

func applyGlassTint(*swt.Composite) {}
