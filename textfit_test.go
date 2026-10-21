package gowt

import "testing"

func TestAutoGrowFitTo(t *testing.T) {
	for _, c := range []struct {
		name          string
		min, max      int
		need          int
		wantH, wantEx int
	}{
		{"short text centred", 40, 120, 32, 40, 4},
		{"between", 40, 120, 64, 64, 0},
		{"capped", 40, 120, 208, 120, 0},
		{"no cap", 40, 0, 500, 500, 0},
		{"no minimum", 0, 120, 20, 20, 0},
		{"odd space", 41, 0, 32, 41, 4},
	} {
		h, ex := (&autoGrow{min: c.min, max: c.max}).fitTo(c.need)
		if h != c.wantH || ex != c.wantEx {
			t.Errorf("%s: fitTo(%d) = %d, %d; want %d, %d", c.name, c.need, h, ex, c.wantH, c.wantEx)
		}
	}
}
