// Package snapcmp compares two screenshots with a per-pixel tolerance (stdlib only, no GUI).
package snapcmp

import (
	"image"
	"image/color"
)

// Result of one comparison. Size mismatch is a failure with Diff == nil.
type Result struct {
	Differing, Total int
	SizeMismatch     bool
	Diff             image.Image
}

// Fraction is the share of differing pixels.
func (r Result) Fraction() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Differing) / float64(r.Total)
}

// Compare counts pixels whose largest channel delta (0..255) exceeds threshold; Diff shows
// them in red over a dimmed reference.
func Compare(ref, got image.Image, threshold int) Result {
	rb, gb := ref.Bounds(), got.Bounds()
	if rb.Dx() != gb.Dx() || rb.Dy() != gb.Dy() {
		return Result{SizeMismatch: true}
	}
	diff := image.NewRGBA(image.Rect(0, 0, rb.Dx(), rb.Dy()))
	res := Result{Total: rb.Dx() * rb.Dy(), Diff: diff}
	for y := 0; y < rb.Dy(); y++ {
		for x := 0; x < rb.Dx(); x++ {
			r1, g1, b1, a1 := ref.At(rb.Min.X+x, rb.Min.Y+y).RGBA()
			r2, g2, b2, a2 := got.At(gb.Min.X+x, gb.Min.Y+y).RGBA()
			d := max(delta(r1, r2), delta(g1, g2), delta(b1, b2), delta(a1, a2))
			if d > threshold {
				res.Differing++
				diff.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
				continue
			}
			diff.SetRGBA(x, y, color.RGBA{uint8(r1>>8)/2 + 127, uint8(g1>>8)/2 + 127, uint8(b1>>8)/2 + 127, 255})
		}
	}
	return res
}

func delta(a, b uint32) int {
	d := int(a>>8) - int(b>>8)
	if d < 0 {
		return -d
	}
	return d
}

// Within reports whether the differing share is at most maxFraction.
func (r Result) Within(maxFraction float64) bool {
	return !r.SizeMismatch && r.Fraction() <= maxFraction
}
