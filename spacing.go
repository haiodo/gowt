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

// Spacing is the application-wide scale used by Spaced, Padded and available for Margin and Spacing fields:
// Grid{Columns: 2, Margin: gowt.Spacing.LG, Spacing: gowt.Spacing.SM}.
// An application tunes it once at startup; a change applies to layouts created afterwards, never to existing ones.
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

// Spaced returns g with Spacing set to Spacing.SM. A zero Spacing keeps meaning "none"; only this call changes it.
func (g Grid) Spaced() Grid { g.Spacing = Spacing.SM; return g }

// Padded returns g with Margin set to Spacing.PageMargin; use it on the root layout of a window or page, not on nested ones.
func (g Grid) Padded() Grid { g.Margin = Spacing.PageMargin; return g }

// Spaced returns r with Spacing set to Spacing.SM.
func (r Row) Spaced() Row { r.Spacing = Spacing.SM; return r }

// Padded returns r with Margin set to Spacing.PageMargin.
func (r Row) Padded() Row { r.Margin = Spacing.PageMargin; return r }

// Spaced returns f with Spacing set to Spacing.SM.
func (f Form) Spaced() Form { f.Spacing = Spacing.SM; return f }

// Padded returns f with Margin set to Spacing.PageMargin.
func (f Form) Padded() Form { f.Margin = Spacing.PageMargin; return f }

// Padded returns s with Margin set to Spacing.PageMargin.
func (s Stack) Padded() Stack { s.Margin = Spacing.PageMargin; return s }
