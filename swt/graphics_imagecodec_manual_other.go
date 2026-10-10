//go:build !linux

package swt

import "image"

const pixbufLikeAlpha = false

func pngSource(img image.Image) image.Image { return img }
