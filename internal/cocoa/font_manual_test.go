//go:build darwin

package cocoa

import "testing"

func TestNSFontWeightOf(t *testing.T) {
	cases := []struct {
		w    int32
		want float64
	}{{400, 0}, {500, 0.23}, {600, 0.3}, {700, 0.4}, {0, -0.8}, {1000, 0.62}, {449, 0}, {450, 0.23}}
	for _, c := range cases {
		if got := NSFontWeightOf(c.w); got != c.want {
			t.Errorf("NSFontWeightOf(%d) = %v, want %v", c.w, got, c.want)
		}
	}
}
