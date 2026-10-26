//go:build darwin

package cocoa

import "testing"

func TestGraphemeRange(t *testing.T) {
	family := "\U0001F468‍\U0001F469‍\U0001F467" // 8 UTF-16 units
	s := "xy" + family + "z"
	if loc, n := GraphemeRange(s, 9); loc != 2 || n != 8 {
		t.Errorf("family: %d,%d, want 2,8", loc, n)
	}
	if loc, n := GraphemeRange(s, 0); loc != 0 || n != 1 {
		t.Errorf("x: %d,%d, want 0,1", loc, n)
	}
}
