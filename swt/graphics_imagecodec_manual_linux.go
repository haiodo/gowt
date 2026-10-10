package swt

import "image"

// GTK's NativeImageLoader goes through GdkPixbuf: a PNG with an alpha channel loads as 32-bit
// ImageData, and PNG is always saved as RGBA. The Go codecs only keep what the data needs.
const pixbufLikeAlpha = true

type opaqueFalse struct{ *image.NRGBA }

// Opaque makes image/png write the alpha channel even when every pixel is 255.
func (opaqueFalse) Opaque() bool { return false }

func pngSource(img image.Image) image.Image {
	if n, ok := img.(*image.NRGBA); ok {
		return opaqueFalse{n}
	}
	return img
}
