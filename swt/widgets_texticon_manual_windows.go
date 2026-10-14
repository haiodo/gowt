package swt

import (
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/win32"
)

func (this *Display) GetTextSearchIcon(size int32) int64 {
	return this.textIcon(this.sizeToSearchIconHandle, size, false)
}

func (this *Display) GetTextCancelIcon(size int32) int64 {
	return this.textIcon(this.sizeToCancelIconHandle, size, true)
}

// textIcon caches per size like SWT; on failure it caches 0, so nothing is drawn and creation is not retried every paint.
func (this *Display) textIcon(cache *jrt.Map, size int32, cancel bool) int64 {
	if !cache.ContainsKey(size) {
		cache.Put(size, newTextIcon(size, cancel, this.textUseDarkthemeIcons))
	}
	return jrt.Cast[int64](cache.Get(size))
}

func newTextIcon(size int32, cancel, dark bool) int64 {
	if size <= 0 || size > 512 {
		return 0
	}
	pix := textIconBGRA(int(size), cancel, dark)
	hdr := win32.NewBITMAPINFOHEADER()
	hdr.BiSize = win32.BITMAPINFOHEADERSizeof
	hdr.BiWidth = size
	hdr.BiHeight = -size
	hdr.BiPlanes = int16(1)
	hdr.BiBitCount = int16(32)
	hdr.BiCompression = win32.OSBI_RGB
	bmi := make([]int8, win32.BITMAPINFOHEADERSizeof)
	win32.OSMoveMemoryOverload5(bmi, hdr, win32.BITMAPINFOHEADERSizeof)
	pBits := make([]int64, 1)
	color := win32.OSCreateDIBSection(0, bmi, win32.OSDIB_RGB_COLORS, pBits, 0, 0)
	if color == 0 || pBits[0] == 0 {
		return 0
	}
	src := make([]int8, len(pix))
	for i, b := range pix {
		src[i] = int8(b)
	}
	win32.OSMoveMemoryOverload11(pBits[0], src, int32(len(src)))
	// The AND mask is required by CreateIconIndirect; the alpha of the color bitmap decides the shape.
	mask := win32.OSCreateBitmap(size, size, 1, 1, make([]int8, ((size+15)/16)*2*size))
	if mask == 0 {
		win32.OSDeleteObject(color)
		return 0
	}
	info := win32.NewICONINFO()
	info.FIcon = true
	info.HbmMask = mask
	info.HbmColor = color
	icon := win32.OSCreateIconIndirect(info)
	win32.OSDeleteObject(color)
	win32.OSDeleteObject(mask)
	return icon
}
