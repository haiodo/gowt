package swt

import "math"

// textIconBGRA renders the magnifier (or the cross) of a SEARCH Text into top-down 32-bit BGRA with straight alpha,
// 4x4 supersampled. Pure so that it is testable off Windows.
func textIconBGRA(size int, cancel, dark bool) []byte {
	pix := make([]byte, size*size*4)
	var grey byte = 0x60
	if dark {
		grey = 0xD0
	}
	s := float64(size)
	const sub = 4
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hit := 0
			for sy := 0; sy < sub; sy++ {
				for sx := 0; sx < sub; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/sub) / s
					py := (float64(y) + (float64(sy)+0.5)/sub) / s
					if textIconInside(px, py, cancel) {
						hit++
					}
				}
			}
			i := (y*size + x) * 4
			pix[i], pix[i+1], pix[i+2] = grey, grey, grey
			pix[i+3] = byte(hit * 255 / (sub * sub))
		}
	}
	return pix
}

// textIconInside takes unit-square coordinates.
func textIconInside(x, y float64, cancel bool) bool {
	if cancel {
		return segDist(x, y, 0.3, 0.3, 0.7, 0.7) <= 0.05 || segDist(x, y, 0.7, 0.3, 0.3, 0.7) <= 0.05
	}
	const cx, cy, r = 0.42, 0.42, 0.24
	if math.Abs(math.Hypot(x-cx, y-cy)-r) <= 0.05 {
		return true
	}
	return segDist(x, y, cx+r*0.707, cy+r*0.707, 0.82, 0.82) <= 0.06
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}
