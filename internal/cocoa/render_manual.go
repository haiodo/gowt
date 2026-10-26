//go:build darwin

// Hand-written: a view drawn into a bitmap, for tests.
package cocoa

import (
	"image"
	"unsafe"
)

// RenderView draws v and its subviews into a bitmap: what the views draw, not what the window
// server composes over them.
func RenderView(v *NSView) *image.NRGBA {
	rep := v.BitmapImageRepForCachingDisplayInRect(v.Bounds())
	if rep == nil {
		return nil
	}
	v.CacheDisplayInRect(v.Bounds(), rep)
	w, h := int(msg(uintptr(rep.Id), "pixelsWide")), int(msg(uintptr(rep.Id), "pixelsHigh"))
	stride, spp := int(rep.BytesPerRow()), int(rep.SamplesPerPixel())
	if rep.BitmapData() == 0 || spp < 4 {
		return nil
	}
	data := uintptr(rep.BitmapData()) // memory of AppKit, not of the Go heap
	src := unsafe.Slice(*(**byte)(unsafe.Pointer(&data)), stride*h)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+4*w], src[y*stride:y*stride+4*w])
	}
	return img
}
