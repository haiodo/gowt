package icons

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"runtime"
	"testing"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

// No Display and no window: markTemplate sets the flag on the NSImage the swt.Image keeps.
func TestMarkTemplate(t *testing.T) {
	runtime.LockOSThread() // the pool belongs to one thread; it exits with the test
	pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
	defer cocoa.Msg(pool, "drain")
	ns := cocoa.NewNSImage()
	ns.Id = cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSImage"), "alloc"), "init")
	defer ns.Release()
	img := &swt.Image{Handle: ns}
	if isTemplate(img) {
		t.Fatal("a new NSImage is a template")
	}
	markTemplate(img)
	if !isTemplate(img) {
		t.Fatal("markTemplate left the NSImage a non-template")
	}
}

// The key tells Get(name) from Get(name, Color(c)) with the same resolved colour.
func TestIconKeyTemplate(t *testing.T) {
	c := gowt.RGB{R: 30, G: 30, B: 30}
	if (iconKey{name: "add", size: 16, c: c, template: true}) == (iconKey{name: "add", size: 16, c: c}) {
		t.Fatal("a template and a coloured icon share a cache key")
	}
}

// Needs a Display, no window: which images are templates.
func TestGetTemplate(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	done := make(chan struct{})
	mainq <- func() {
		defer close(done)
		err = gowt.Run(func(app *gowt.App) {
			if !isTemplate(Get(app, "file").Unwrap()) {
				t.Error("Get without Color is not a template")
			}
			if isTemplate(Get(app, "file", Color(gowt.RGB{R: 30, G: 30, B: 30})).Unwrap()) {
				t.Error("Get with Color is a template")
			}
			// A zoom change swaps the representations of the NSImage, not the NSImage.
			ic := Get(app, "file").Unwrap()
			ic.GetOrCreateRepresentation(200)
			ic.GetOrCreateRepresentation(100)
			if !isTemplate(ic) {
				t.Error("Get is not a template after a zoom change")
			}
			doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="4"/></svg>`)
			if isTemplate(FromSVG(app, doc, 4, gowt.RGB{}).Unwrap()) {
				t.Error("FromSVG is a template")
			}
			if isTemplate(app.ImageFromProvider(svg.NewImageDataProvider(doc, 4, 4, "#000000")).Unwrap()) {
				t.Error("ImageFromProvider is a template")
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
				t.Error(err)
			}
			img, err := app.ImageFrom(&buf)
			if err != nil {
				t.Error(err)
			} else if isTemplate(img.Unwrap()) {
				t.Error("ImageFrom is a template")
			}
		})
	}
	<-done
	if err != nil {
		t.Fatal(err)
	}
}
