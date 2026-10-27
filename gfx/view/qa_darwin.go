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

	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// Overlay replaces the canvas rectangle of img with the canvas background, then the tiles from the
// compositor's images, then the native child views (Overlay and Edit controls) captured one by one.
// cacheDisplayInRect has the layers on some macOS versions and not on others: tiles drawn over a
// capture that has them double translucent pixels and cover the children.
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
	kids := visibleChildren(view)
	// The children stay shown: hiding a field resigns its first responder and ends the edit.
	v.host.SetHidden(true)
	back, err := capture(view)
	v.host.SetHidden(false)
	if err != nil {
		return err
	}
	tiles := v.host.Render(r.W, r.H, scale)
	if back.Bounds().Size() != tiles.Bounds().Size() {
		return fmt.Errorf("view capture is %v, tiles %v", back.Bounds().Size(), tiles.Bounds().Size())
	}
	draw.Draw(back, back.Bounds(), tiles, image.Point{}, draw.Over)
	flipped := objc.Send[bool](view, sel("isFlipped"))
	for _, k := range kids {
		kimg, err := capture(k)
		if err != nil {
			return err
		}
		kr := objc.Send[nsRect](view, sel("convertRect:fromView:"), objc.Send[nsRect](k, sel("bounds")), k)
		ky := kr.Y
		if !flipped {
			ky = r.H - kr.Y - kr.H
		}
		kat := image.Pt(int(kr.X*scale+0.5), int(ky*scale+0.5))
		draw.Draw(back, kimg.Bounds().Add(kat), kimg, image.Point{}, draw.Over)
	}
	at := image.Pt(img.Bounds().Min.X+int(r.X*scale+0.5), img.Bounds().Min.Y+int(y*scale+0.5))
	draw.Draw(img, back.Bounds().Add(at), back, image.Point{}, draw.Src)
	return nil
}

// visibleChildren are the shown native subviews of the canvas.
func visibleChildren(view objc.ID) []objc.ID {
	sel := objc.RegisterName
	subs := view.Send(sel("subviews"))
	var out []objc.ID
	for i, n := uint64(0), objc.Send[uint64](subs, sel("count")); i < n; i++ {
		if k := subs.Send(sel("objectAtIndex:"), i); !objc.Send[bool](k, sel("isHidden")) {
			out = append(out, k)
		}
	}
	return out
}

// capture draws view as cacheDisplayInRect does and returns the bitmap.
func capture(view objc.ID) (*image.NRGBA, error) {
	if shot.Fast { // a recorder: no PNG round trip per frame
		if img, err := shot.ViewImage(view); err == nil {
			return img, nil
		}
	}
	sel := objc.RegisterName
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
