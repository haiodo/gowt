// Hand-written stub for org.eclipse.swt.graphics.RoundingMode (manual.txt): the translator has
// no rule for Java enums in v0. Zero value is ROUND, matching the Java field defaults j2go drops.
package swt

import "math"

type RoundingMode int32

const (
	RoundingModeROUND RoundingMode = iota
	RoundingModeUP
	RoundingModeDOWN
)

func (r RoundingMode) Round(x float32) int32 {
	switch r {
	case RoundingModeROUND:
		return int32(math.Round(float64(x)))
	case RoundingModeUP:
		return int32(math.Ceil(float64(x)))
	case RoundingModeDOWN:
		return int32(math.Floor(float64(x)))
	}
	return int32(x)
}
