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

// No Display and no window: a 2x2-point image at scale 2, opaque red on the left half, tinted blue.
func TestViewTableTinted(t *testing.T) {
	pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
	defer cocoa.Msg(pool, "drain")
	size := cocoa.NSSize{Width: 2, Height: 2}
	src := castcocoaNSObjectTococoaNSImage(cocoa.NewNSImage().Alloc()).InitWithSize(size)
	defer src.Release()
	rep := castcocoaNSObjectTococoaNSBitmapImageRep(cocoa.NewNSBitmapImageRep().Alloc())
	rep = rep.InitWithBitmapDataPlanes(0, 4, 4, 8, 4, true, false, cocoa.OSNSDeviceRGBColorSpace_, 0, 16, 32)
	rep.SetSize(size)
	cocoa.NSGraphicsContextStatic_saveGraphicsState()
	cocoa.NSGraphicsContextSetCurrentContext(cocoa.NSGraphicsContextGraphicsContextWithBitmapImageRep(rep))
	cocoa.NSColorColorWithDeviceRed(1, 0, 0, 1).SetFill()
	cocoa.NSBezierPathFillRect(cocoa.NSRect{Width: 1, Height: 2})
	cocoa.NSGraphicsContextStatic_restoreGraphicsState()
	src.AddRepresentation(upcastcocoaNSBitmapImageRepTococoaNSImageRep(rep))
	rep.Release()

	out := vtTinted(src, cocoa.NSColorColorWithDeviceRed(0, 0, 1, 1).Id)
	reps := out.Representations()
	if reps.Count() != 1 {
		t.Fatalf("representations: %d, want 1", reps.Count())
	}
	r := reps.ObjectAtIndex(0).Id
	if w := cocoa.NewNSImageRepOverload1(r).PixelsWide(); w != 4 {
		t.Fatalf("pixels wide: %d, want 4", w)
	}
	left := cocoa.NewNSColorOverload1(cocoa.Msg(r, "colorAtX:y:", 0, 2))
	if left.AlphaComponent() < 0.99 || left.BlueComponent() < 0.99 || left.RedComponent() > 0.01 {
		t.Fatalf("left pixel: rgba %.2f,%.2f,%.2f, want opaque blue", left.RedComponent(), left.BlueComponent(), left.AlphaComponent())
	}
	if right := cocoa.NewNSColorOverload1(cocoa.Msg(r, "colorAtX:y:", 3, 2)); right.AlphaComponent() > 0.01 {
		t.Fatalf("right pixel alpha %.2f, want transparent", right.AlphaComponent())
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
	if cocoa.Msg(v, "isAccessibilityElement") != 1 {
		t.Fatal("cell views must be accessibility elements")
	}
	if cocoa.Msg(v, "respondsToSelector:", cocoa.OSSel_mouseUp_) == 0 || cocoa.Msg(v, "respondsToSelector:", cocoa.OSSel_registerName("accessibilityPerformPress")) == 0 {
		t.Fatal("check view must handle mouseUp: and accessibilityPerformPress")
	}
}
