package shot

import (
	"image"
	"testing"
)

func TestFit(t *testing.T) {
	for _, c := range []struct {
		iw, ih    int
		scale     float64
		stretched bool
	}{
		{360, 240, 1, true},
		{720, 480, 2, false},
		{540, 360, 1.5, true}, // 1.5x capture, exact: scaled up
		{543, 352, 1.5, true}, // a few pixels off: stretched
		{352, 120, 1, false},  // far too short: padded
	} {
		out, scale, stretched := fit(image.NewNRGBA(image.Rect(0, 0, c.iw, c.ih)), 360, 240)
		if out.Bounds().Dx() != 720 || out.Bounds().Dy() != 480 || scale != c.scale || stretched != c.stretched {
			t.Errorf("%dx%d: got %v scale %g stretched %v", c.iw, c.ih, out.Bounds(), scale, stretched)
		}
	}
}
