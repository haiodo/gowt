// Package shot captures a Shell's window for the snapshot commands and for GOWT_SNAP (gowt.Run).
package shot

import (
	"fmt"
	"structs"

	"github.com/ebitengine/purego/objc"
)

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// WindowPNG writes the window that holds the NSView view (Shell.View.Id) as a PNG. It renders the view
// tree with cacheDisplayInRect:toBitmapImageRep:; screencapture needs Screen Recording permission.
func WindowPNG(view int64, path string) error {
	v := objc.ID(view).Send(objc.RegisterName("window")).Send(objc.RegisterName("contentView")).Send(objc.RegisterName("superview"))
	bounds := objc.Send[nsRect](v, objc.RegisterName("bounds"))
	rep := v.Send(objc.RegisterName("bitmapImageRepForCachingDisplayInRect:"), bounds)
	v.Send(objc.RegisterName("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(objc.RegisterName("dictionary"))
	data := rep.Send(objc.RegisterName("representationUsingType:properties:"), 4, dict)
	nsPath := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), path)
	if !objc.Send[bool](data, objc.RegisterName("writeToFile:atomically:"), nsPath, true) {
		return fmt.Errorf("writing %s failed", path)
	}
	return nil
}
