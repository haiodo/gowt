package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func TestViewTableIntersects(t *testing.T) {
	a := cocoa.NSRect{X: 0, Y: 0, Width: 10, Height: 10}
	if !vtIntersects(a, cocoa.NSRect{X: 9, Y: 9, Width: 5, Height: 5}) {
		t.Fatal("overlapping rects must intersect")
	}
	if vtIntersects(a, cocoa.NSRect{X: 10, Y: 0, Width: 5, Height: 5}) {
		t.Fatal("rects sharing only an edge must not intersect")
	}
}

// No Display and no window: only registers the runtime classes and creates one view.
func TestViewTableClasses(t *testing.T) {
	cell, check := vtClasses()
	if cell == 0 || check == 0 {
		t.Fatal("classes not registered")
	}
	v := cocoa.Msg(check, "new")
	if cocoa.Msg(v, "isFlipped") != 1 {
		t.Fatal("cell view must be flipped")
	}
	if cocoa.OSObject_getClass(v) != check || cocoa.Msg(v, "isKindOfClass:", cell) == 0 {
		t.Fatal("check view must subclass the cell view")
	}
}
