package gowt

import "runtime"

// SpacingScale is the set of spacing tokens, in points like every other facade size.
// The platform scales points to pixels, so the values are whole pixels at 100..200% in steps of 25%
// as long as they are multiples of 4.
type SpacingScale struct {
	XS, SM, MD, LG, XL int
	// PageMargin is the margin between a window or page edge and its content.
	PageMargin int
}

// Spacing is the application-wide scale: the default Gap is Spacing.SM and the default Padding of a
// Window and a Tab is Spacing.PageMargin; Gap(gowt.Spacing.LG) picks another step.
// An application tunes it once at startup; a change applies to containers created afterwards, never to existing ones.
var Spacing = defaultSpacing()

func defaultSpacing() SpacingScale {
	s := SpacingScale{XS: 4, SM: 8, MD: 12, LG: 16, XL: 32}
	s.PageMargin = s.MD
	if runtime.GOOS == "darwin" {
		s.PageMargin = s.LG
	}
	return s
}

// Golden splits total into a smaller and a larger part in the golden ratio (about 38% and 62%); small+large == total.
func Golden(total int) (small, large int) {
	small = int(float64(total)*0.381966 + 0.5)
	return small, total - small
}
