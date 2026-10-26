// Package shot captures a Shell's window for the snapshot commands and for GOWT_SNAP (gowt.Run).
package shot

import (
	"fmt"
	"os"
	"structs"

	"github.com/ebitengine/purego/objc"
)

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// Capture writes the window that holds the NSView view (Shell.View.Id) as a PNG. It renders the view
// tree with cacheDisplayInRect:toBitmapImageRep:; screencapture needs Screen Recording permission.
// GOWT_SNAP_LAYERS=1 renders the layer tree instead (layerPNG); the reference images keep the default.
// It returns where the window lies on the screen, in points, y down.
func Capture(view int64, path string) (Frame, error) {
	win := objc.ID(view).Send(objc.RegisterName("window"))
	v := win.Send(objc.RegisterName("contentView")).Send(objc.RegisterName("superview"))
	if os.Getenv("GOWT_SNAP_NOFOCUS") == "1" {
		// The window takes the first responder, and no view or cell may draw a ring in the capture itself:
		// the ring of the old responder was still in the shots with makeFirstResponder:nil alone.
		win.Send(objc.RegisterName("makeFirstResponder:"), objc.ID(0))
		noFocusRings(v)
	}
	if os.Getenv("GOWT_SNAP_SCREEN") == "1" {
		return frameOf(win), screenPNG(v, path)
	}
	if os.Getenv("GOWT_SNAP_LAYERS") == "1" && v.Send(objc.RegisterName("layer")) != 0 {
		return frameOf(win), layerPNG(v, path)
	}
	bounds := objc.Send[nsRect](v, objc.RegisterName("bounds"))
	rep := v.Send(objc.RegisterName("bitmapImageRepForCachingDisplayInRect:"), bounds)
	v.Send(objc.RegisterName("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(objc.RegisterName("dictionary"))
	data := rep.Send(objc.RegisterName("representationUsingType:properties:"), 4, dict)
	nsPath := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), path)
	if !objc.Send[bool](data, objc.RegisterName("writeToFile:atomically:"), nsPath, true) {
		return Frame{}, fmt.Errorf("writing %s failed", path)
	}
	return frameOf(win), nil
}

// frameOf is the frame of an NSWindow with the origin at the top left of the primary screen.
func frameOf(win objc.ID) Frame {
	sel := objc.RegisterName
	fr := objc.Send[nsRect](win, sel("frame"))
	screens := objc.ID(objc.GetClass("NSScreen")).Send(sel("screens"))
	primary := objc.Send[nsRect](screens.Send(sel("objectAtIndex:"), 0), sel("frame"))
	return Frame{fr.X, primary.H - fr.Y - fr.H, fr.W, fr.H}
}

// noFocusRings sets NSFocusRingTypeNone on v, its cell and every view below it. The process ends with
// the capture, so the change is not undone.
func noFocusRings(v objc.ID) {
	sel := objc.RegisterName
	const focusRingTypeNone = 1
	v.Send(sel("setFocusRingType:"), focusRingTypeNone)
	if objc.Send[bool](v, sel("respondsToSelector:"), sel("cell")) {
		if cell := v.Send(sel("cell")); cell != 0 && objc.Send[bool](cell, sel("respondsToSelector:"), sel("setFocusRingType:")) {
			cell.Send(sel("setFocusRingType:"), focusRingTypeNone)
		}
	}
	subs := v.Send(sel("subviews"))
	for i, n := uint64(0), objc.Send[uint64](subs, sel("count")); i < n; i++ {
		noFocusRings(subs.Send(sel("objectAtIndex:"), i))
	}
}
