package gowt

import "testing"

func TestPopoverPlace(t *testing.T) {
	screen := popRect{0, 0, 1000, 800}
	tests := []struct {
		name         string
		anchor       popRect
		w, h         int
		edge         PopoverEdge
		wantX, wantY int
	}{
		{"below, centred", popRect{400, 100, 100, 20}, 200, 100, PopoverBelow, 350, 126},
		{"above", popRect{400, 400, 100, 20}, 200, 100, PopoverAbove, 350, 294},
		{"right", popRect{100, 100, 100, 20}, 200, 100, PopoverRight, 206, 60},
		{"left", popRect{500, 100, 100, 20}, 200, 100, PopoverLeft, 294, 60},
		{"below flips above at the bottom", popRect{400, 760, 100, 20}, 200, 100, PopoverBelow, 350, 654},
		{"above flips below at the top", popRect{400, 10, 100, 20}, 200, 100, PopoverAbove, 350, 36},
		{"right flips left at the right edge", popRect{900, 100, 90, 20}, 200, 100, PopoverRight, 694, 60},
		{"slides in from the left edge", popRect{0, 100, 20, 20}, 200, 100, PopoverBelow, 0, 126},
		{"slides in from the right edge", popRect{990, 100, 10, 20}, 200, 100, PopoverBelow, 800, 126},
		{"point anchor", popRect{500, 300, 0, 0}, 200, 100, PopoverBelow, 400, 306},
		{"larger than the screen sticks to the corner", popRect{500, 300, 0, 0}, 1200, 900, PopoverBelow, 0, 0},
	}
	for _, tc := range tests {
		x, y := popoverPlace(tc.anchor, screen, tc.w, tc.h, tc.edge)
		if x != tc.wantX || y != tc.wantY {
			t.Errorf("%s: got (%d, %d), want (%d, %d)", tc.name, x, y, tc.wantX, tc.wantY)
		}
	}
}

func TestPopoverPlaceSecondMonitor(t *testing.T) {
	screen := popRect{-1000, 0, 1000, 800}
	x, y := popoverPlace(popRect{-30, 100, 20, 20}, screen, 200, 100, PopoverBelow)
	if x != -200 || y != 126 {
		t.Errorf("got (%d, %d), want (-200, 126)", x, y)
	}
}
