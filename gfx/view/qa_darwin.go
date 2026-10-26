//go:build darwin

package view

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"structs"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/haiodo/gowt/swt"
)

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// Overlay replaces the canvas rectangle of img with the view captured without its layers plus the
// tiles from the compositor's images: cacheDisplayInRect has the layers on some macOS versions and
// not on others, and drawing tiles over a capture that has them doubles translucent pixels.
func (h qaHook) Overlay(shell *swt.Shell, img *image.NRGBA) error {
	v := h.v
	c := v.cv.Unwrap()
	if v.gone.Load() || c.IsDisposed() || c.GetShell() != shell || !c.IsVisible() {
		return nil
	}
	sel := objc.RegisterName
	view := objc.ID(v.nsview)
	root := view.Send(sel("window")).Send(sel("contentView")).Send(sel("superview"))
	rb := objc.Send[nsRect](root, sel("bounds"))
	r := objc.Send[nsRect](view, sel("convertRect:toView:"), objc.Send[nsRect](view, sel("bounds")), root)
	if rb.W <= 0 {
		return nil
	}
	scale := float64(img.Bounds().Dx()) / rb.W
	y := r.Y
	if !objc.Send[bool](root, sel("isFlipped")) {
		y = rb.H - r.Y - r.H
	}
	back, err := v.captureWithoutLayers(view)
	if err != nil {
		return err
	}
	tiles := v.host.Render(r.W, r.H, scale)
	if back.Bounds().Size() != tiles.Bounds().Size() {
		return fmt.Errorf("view capture is %v, tiles %v", back.Bounds().Size(), tiles.Bounds().Size())
	}
	draw.Draw(back, back.Bounds(), tiles, image.Point{}, draw.Over)
	at := image.Pt(img.Bounds().Min.X+int(r.X*scale+0.5), img.Bounds().Min.Y+int(y*scale+0.5))
	draw.Draw(img, back.Bounds().Add(at), back, image.Point{}, draw.Src)
	return nil
}

// captureWithoutLayers draws the canvas view as cacheDisplayInRect does, with the tile layers hidden.
func (v *View) captureWithoutLayers(view objc.ID) (*image.NRGBA, error) {
	sel := objc.RegisterName
	v.host.SetHidden(true)
	defer v.host.SetHidden(false)
	bounds := objc.Send[nsRect](view, sel("bounds"))
	rep := view.Send(sel("bitmapImageRepForCachingDisplayInRect:"), bounds)
	view.Send(sel("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(sel("dictionary"))
	data := rep.Send(sel("representationUsingType:properties:"), 4, dict) // NSBitmapImageFileTypePNG
	if data == 0 {
		return nil, fmt.Errorf("the view capture has no PNG")
	}
	b := unsafe.Slice(objc.Send[*byte](data, sel("bytes")), objc.Send[uint64](data, sel("length")))
	src, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	out := image.NewNRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(out, out.Bounds(), src, src.Bounds().Min, draw.Src)
	return out, nil
}
