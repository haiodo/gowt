//go:build darwin

package view

import (
	"image"
	"image/draw"
	"structs"

	"github.com/ebitengine/purego/objc"

	"github.com/haiodo/gowt/swt"
)

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// Overlay draws the tile layers into img, the capture of the window's frame view. The view capture
// (cacheDisplayInRect) does not include CALayer contents, so the tiles are drawn from the
// compositor's own images, which needs neither the window server nor screen recording permission.
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
	tiles := v.host.Render(r.W, r.H, scale)
	at := image.Pt(img.Bounds().Min.X+int(r.X*scale+0.5), img.Bounds().Min.Y+int(y*scale+0.5))
	draw.Draw(img, tiles.Bounds().Add(at), tiles, image.Point{}, draw.Over)
	return nil
}
