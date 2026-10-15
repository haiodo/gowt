package shot

import (
	"image/png"
	"os"
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

func init() {
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL); err != nil {
		panic(err)
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", purego.RTLD_GLOBAL); err != nil {
		panic(err)
	}
}

// TestLayerPNGUpright renders a layer tree with a red sublayer over the bottom half; the PNG must show
// it at the bottom, at the backing scale of the rep.
func TestLayerPNGUpright(t *testing.T) {
	sel := objc.RegisterName
	cls := func(n string) objc.ID { return objc.ID(objc.GetClass(n)) }
	root := cls("NSView").Send(sel("alloc")).Send(sel("initWithFrame:"), nsRect{W: 100, H: 100})
	rl := cls("CALayer").Send(sel("layer"))
	rl.Send(sel("setFrame:"), nsRect{W: 100, H: 100})
	sub := cls("CALayer").Send(sel("layer"))
	sub.Send(sel("setFrame:"), nsRect{W: 100, H: 50})
	sub.Send(sel("setBackgroundColor:"), cls("NSColor").Send(sel("redColor")).Send(sel("CGColor")))
	rl.Send(sel("addSublayer:"), sub)
	root.Send(sel("setLayer:"), rl)
	root.Send(sel("setWantsLayer:"), true)
	path := t.TempDir() + "/layer.png"
	if err := layerPNG(root, path); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(path)
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := im.Bounds()
	top, bottom := im.At(b.Dx()/2, 2), im.At(b.Dx()/2, b.Dy()-3)
	if _, _, _, a := top.RGBA(); a != 0 {
		t.Errorf("top half painted: %v", top)
	}
	if r, g, _, _ := bottom.RGBA(); r>>8 < 200 || g>>8 > 50 {
		t.Errorf("bottom half is %v, want red", bottom)
	}
}
