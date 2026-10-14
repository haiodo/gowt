package shot

import (
	"fmt"

	"github.com/ebitengine/purego/objc"
)

// layerPNG renders the layer tree of the layer-backed view v, so layer-only content (colours set in
// updateLayer, as the gowt.Section card) shows too. Backdrops and glass are composited by the window
// server and stay empty.
func layerPNG(v objc.ID, path string) error {
	sel := objc.RegisterName
	bounds := objc.Send[nsRect](v, sel("bounds"))
	rep := v.Send(sel("bitmapImageRepForCachingDisplayInRect:"), bounds)
	ctx := objc.ID(objc.GetClass("NSGraphicsContext")).Send(sel("graphicsContextWithBitmapImageRep:"), rep)
	cg := ctx.Send(sel("CGContext"))
	v.Send(sel("layer")).Send(sel("renderInContext:"), cg)
	ctx.Send(sel("flushGraphics"))
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(sel("dictionary"))
	data := rep.Send(sel("representationUsingType:properties:"), 4, dict)
	nsPath := objc.ID(objc.GetClass("NSString")).Send(sel("stringWithUTF8String:"), path)
	if !objc.Send[bool](data, sel("writeToFile:atomically:"), nsPath, true) {
		return fmt.Errorf("writing %s failed", path)
	}
	return nil
}
