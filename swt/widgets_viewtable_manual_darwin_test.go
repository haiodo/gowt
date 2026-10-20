package swt

import (
	"runtime"
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

// vtHalfImage makes a 2x2-point image at scale 2 filled with r, g, b from the left over width points.
func vtHalfImage(r, g, b, width float64) *cocoa.NSImage {
	size := cocoa.NSSize{Width: 2, Height: 2}
	src := castcocoaNSObjectTococoaNSImage(cocoa.NewNSImage().Alloc()).InitWithSize(size)
	rep := castcocoaNSObjectTococoaNSBitmapImageRep(cocoa.NewNSBitmapImageRep().Alloc())
	rep = rep.InitWithBitmapDataPlanes(0, 4, 4, 8, 4, true, false, cocoa.OSNSDeviceRGBColorSpace_, 0, 16, 32)
	rep.SetSize(size)
	cocoa.NSGraphicsContextStatic_saveGraphicsState()
	cocoa.NSGraphicsContextSetCurrentContext(cocoa.NSGraphicsContextGraphicsContextWithBitmapImageRep(rep))
	cocoa.NSColorColorWithDeviceRed(r, g, b, 1).SetFill()
	cocoa.NSBezierPathFillRect(cocoa.NSRect{Width: width, Height: 2})
	cocoa.NSGraphicsContextStatic_restoreGraphicsState()
	src.AddRepresentation(upcastcocoaNSBitmapImageRepTococoaNSImageRep(rep))
	rep.Release()
	return src
}

// No Display and no window: a dark grey glyph on the left half is tinted blue.
func TestViewTableTinted(t *testing.T) {
	runtime.LockOSThread() // the pool and the graphics state stack belong to one thread; it exits with the test
	pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
	defer cocoa.Msg(pool, "drain")
	src := vtHalfImage(30.0/255, 30.0/255, 30.0/255, 1)
	defer src.Release()

	out := vtTinted(src, 0, cocoa.NSColorColorWithDeviceRed(0, 0, 1, 1).Id)
	if out.Id == src.Id {
		t.Fatal("a grey glyph came back untinted")
	}
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

// No Display and no window: a colour image with alpha (an avatar) and an opaque grey one are not glyphs.
func TestViewTableTintedKeepsColour(t *testing.T) {
	runtime.LockOSThread() // the pool and the graphics state stack belong to one thread; it exits with the test
	pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
	defer cocoa.Msg(pool, "drain")
	for name, src := range map[string]*cocoa.NSImage{
		"blue with alpha": vtHalfImage(0x2F/255.0, 0x6F/255.0, 0xEF/255.0, 1),
		"opaque grey":     vtHalfImage(0.5, 0.5, 0.5, 2),
	} {
		if out := vtTinted(src, 0, cocoa.NSColorColorWithDeviceRed(0, 0, 1, 1).Id); out.Id != src.Id {
			t.Errorf("%s: tinted, want the image as it is", name)
		}
		src.Release()
	}
}

// No Display and no window: the grey glyph of TestViewTableTinted in the second column keeps its pixels.
func TestViewTableTintedLeadingColumnOnly(t *testing.T) {
	runtime.LockOSThread() // the pool and the graphics state stack belong to one thread; it exits with the test
	pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
	defer cocoa.Msg(pool, "drain")
	src := vtHalfImage(30.0/255, 30.0/255, 30.0/255, 1)
	defer src.Release()
	if out := vtTinted(src, 1, cocoa.NSColorColorWithDeviceRed(0, 0, 1, 1).Id); out.Id != src.Id {
		t.Fatal("a glyph in column 1 was tinted, want only column 0")
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
