// Package offscreen picks the headless gfx.Renderer of the running OS: CoreGraphics on macOS, cairo
// and Pango on Linux, Direct2D on Windows. It needs no window, no display and no build tags.
//
//	r, err := offscreen.New(offscreen.Options{})
//	raster, err := gfx.RenderSubtree(r, scene, node, gfx.Rect{}, 2, bg)
//	png, err := raster.PNG()
package offscreen

// Options are the settings every backend understands.
type Options struct {
	// LOD skips text, shadows and dashes that are unreadable at the scale drawn.
	LOD bool
	// FontDir is a directory (or one file) of fonts to register for this process; the environment
	// variable GFX_FONT_DIR stands in when it is empty.
	FontDir string
}
