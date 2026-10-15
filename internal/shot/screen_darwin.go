package shot

import (
	"fmt"
	"math"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	windowImage  func(bounds nsRect, list, window, image uint32) uintptr
	imageRelease func(img uintptr)
)

// screenPNG writes the window of v as the window server composites it, glass and backdrops included
// (GOWT_SNAP_SCREEN=1); it may need Screen Recording permission.
func screenPNG(v objc.ID, path string) error {
	sel := objc.RegisterName
	if windowImage == nil {
		cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_GLOBAL)
		if err != nil {
			return err
		}
		purego.RegisterLibFunc(&windowImage, cg, "CGWindowListCreateImage")
		purego.RegisterLibFunc(&imageRelease, cg, "CGImageRelease")
	}
	const (
		includingWindow = 1 << 3 // kCGWindowListOptionIncludingWindow
		bestResolution  = 1 << 3 // kCGWindowImageBestResolution
	)
	wid := objc.Send[int64](v.Send(sel("window")), sel("windowNumber"))
	null := nsRect{X: math.Inf(1), Y: math.Inf(1)} // CGRectNull: the window's own bounds
	img := windowImage(null, includingWindow, uint32(wid), bestResolution)
	if img == 0 {
		return fmt.Errorf("CGWindowListCreateImage returned no image (Screen Recording permission?)")
	}
	defer imageRelease(img)
	rep := objc.ID(objc.GetClass("NSBitmapImageRep")).Send(sel("alloc")).Send(sel("initWithCGImage:"), img)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(sel("dictionary"))
	data := rep.Send(sel("representationUsingType:properties:"), 4, dict)
	nsPath := objc.ID(objc.GetClass("NSString")).Send(sel("stringWithUTF8String:"), path)
	if !objc.Send[bool](data, sel("writeToFile:atomically:"), nsPath, true) {
		return fmt.Errorf("writing %s failed", path)
	}
	return nil
}
