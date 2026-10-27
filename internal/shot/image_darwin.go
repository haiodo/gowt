package shot

import (
	"errors"
	"image"
	"os"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// ErrNoImage means CaptureImage cannot serve this capture; use Capture.
var ErrNoImage = errors.New("shot: no in-memory capture")

// CaptureImage is Capture without the PNG round trip: the bitmap the view renders into is copied
// to an image. Only the default view capture is served (not GOWT_SNAP_SCREEN or GOWT_SNAP_LAYERS).
func CaptureImage(view int64) (*image.NRGBA, Frame, error) {
	if os.Getenv("GOWT_SNAP_SCREEN") == "1" || os.Getenv("GOWT_SNAP_LAYERS") == "1" {
		return nil, Frame{}, ErrNoImage
	}
	sel := objc.RegisterName
	win := objc.ID(view).Send(sel("window"))
	v := win.Send(sel("contentView")).Send(sel("superview"))
	ClearFocus(view)
	img, err := ViewImage(v)
	if err != nil {
		return nil, Frame{}, err
	}
	return img, frameOf(win), nil
}

// ViewImage renders the view tree of v as cacheDisplayInRect does and copies the bitmap, with no PNG
// in between.
func ViewImage(v objc.ID) (*image.NRGBA, error) {
	sel := objc.RegisterName
	bounds := objc.Send[nsRect](v, sel("bounds"))
	rep := v.Send(sel("bitmapImageRepForCachingDisplayInRect:"), bounds)
	v.Send(sel("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	w, h := objc.Send[int](rep, sel("pixelsWide")), objc.Send[int](rep, sel("pixelsHigh"))
	stride := objc.Send[int](rep, sel("bytesPerRow"))
	format := objc.Send[uint](rep, sel("bitmapFormat"))
	data := objc.Send[unsafe.Pointer](rep, sel("bitmapData"))
	// Only 8-bit RGBA with the alpha last is read; anything else goes the PNG way.
	if data == nil || w <= 0 || h <= 0 || stride < 4*w || objc.Send[int](rep, sel("samplesPerPixel")) != 4 ||
		objc.Send[int](rep, sel("bitsPerSample")) != 8 || format&1 != 0 {
		return nil, ErrNoImage
	}
	src := unsafe.Slice((*byte)(data), stride*h)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	premul := format&2 == 0 // bit 1 is NSBitmapFormatAlphaNonpremultiplied
	for y := 0; y < h; y++ {
		dst := img.Pix[y*img.Stride : y*img.Stride+4*w]
		copy(dst, src[y*stride:y*stride+4*w])
		if !premul {
			continue
		}
		for i := 0; i < len(dst); i += 4 {
			if a := uint32(dst[i+3]); a != 0 && a != 255 {
				dst[i], dst[i+1], dst[i+2] = byte(uint32(dst[i])*255/a), byte(uint32(dst[i+1])*255/a), byte(uint32(dst[i+2])*255/a)
			}
		}
	}
	return img, nil
}
