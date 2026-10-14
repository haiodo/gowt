//go:build darwin

package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

// NSFontWeight values for CSS weights 100..900.
var nsFontWeights = [9]float64{-0.8, -0.6, -0.4, 0, 0.23, 0.3, 0.4, 0.56, 0.62}

// NSFontManager weights (0..15) for CSS weights 100..900.
var managerWeights = [9]int64{1, 2, 3, 5, 6, 8, 9, 11, 12}

func weightIndex(w int32) int {
	return min(max(int((w+50)/100), 1), 9) - 1
}

// NSFontWeightOf maps a CSS weight (100..900) to the NSFontWeight constant.
func NSFontWeightOf(w int32) float64 { return nsFontWeights[weightIndex(w)] }

var (
	familyOnce sync.Once
	msgFamily  func(self, sel, family uintptr, traits uint64, weight int64, size float64) uintptr
)

// NSFontSystemFontOfSizeWeight is +[NSFont systemFontOfSize:weight:] (macOS 10.11+) with a CSS weight.
func NSFontSystemFontOfSizeWeight(size float64, weight int32) *NSFont {
	r := OSObjc_msgSendOverload36(OSClass_NSFont, int64(sel("systemFontOfSize:weight:")), size, NSFontWeightOf(weight))
	if r == 0 {
		return nil
	}
	return NewNSFontOverload1(r)
}

// NSFontWithFamilyWeight is -[NSFontManager fontWithFamily:traits:weight:size:] with no traits; nil when the family is unknown.
func NSFontWithFamilyWeight(family string, weight int32, size float64) *NSFont {
	familyOnce.Do(func() { purego.RegisterFunc(&msgFamily, objcMsgSend) })
	mgr := msg(class("NSFontManager"), "sharedFontManager")
	r := msgFamily(mgr, sel("fontWithFamily:traits:weight:size:"), nsString(family), 0, managerWeights[weightIndex(weight)], size)
	if r == 0 {
		return nil
	}
	return NewNSFontOverload1(int64(r))
}
