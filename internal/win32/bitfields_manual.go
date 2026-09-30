//go:build windows

// toC/fromC of the C bitfield structs Uniscribe and GetMenuBarInfo use: a Java boolean per bit, packed as
// the C compiler does (LSB first in the declared storage unit).
package win32

import "unsafe"

func bit(v uint32, n uint) bool { return v>>n&1 != 0 }

func setBit(v uint32, n uint, b bool) uint32 {
	if b {
		return v | 1<<n
	}
	return v
}

func (this *SCRIPT_STATE) toC(p unsafe.Pointer) {
	v := uint32(this.UBidiLevel) & 0x1f
	for i, b := range []bool{this.FOverrideDirection, this.FInhibitSymSwap, this.FCharShape, this.FDigitSubstitute, this.FInhibitLigate,
		this.FDisplayZWG, this.FArabicNumContext, this.FGcpClusters, this.FReserved} {
		v = setBit(v, uint(5+i), b)
	}
	*(*uint16)(p) = uint16(v)
}

func (this *SCRIPT_STATE) fromC(p unsafe.Pointer) {
	v := uint32(*(*uint16)(p))
	this.UBidiLevel = int16(v & 0x1f)
	this.FOverrideDirection, this.FInhibitSymSwap, this.FCharShape, this.FDigitSubstitute = bit(v, 5), bit(v, 6), bit(v, 7), bit(v, 8)
	this.FInhibitLigate, this.FDisplayZWG, this.FArabicNumContext, this.FGcpClusters, this.FReserved = bit(v, 9), bit(v, 10), bit(v, 11), bit(v, 12), bit(v, 13)
}

func (this *SCRIPT_ANALYSIS) toC(p unsafe.Pointer) {
	v := uint32(this.EScript) & 0x3ff
	for i, b := range []bool{this.FRTL, this.FLayoutRTL, this.FLinkBefore, this.FLinkAfter, this.FLogicalOrder, this.FNoGlyphIndex} {
		v = setBit(v, uint(10+i), b)
	}
	*(*uint16)(p) = uint16(v)
	if this.S != nil {
		this.S.toC(unsafe.Add(p, 2))
	}
}

func (this *SCRIPT_ANALYSIS) fromC(p unsafe.Pointer) {
	v := uint32(*(*uint16)(p))
	this.EScript = int16(v & 0x3ff)
	this.FRTL, this.FLayoutRTL, this.FLinkBefore, this.FLinkAfter, this.FLogicalOrder, this.FNoGlyphIndex = bit(v, 10), bit(v, 11), bit(v, 12), bit(v, 13), bit(v, 14), bit(v, 15)
	if this.S == nil {
		this.S = NewSCRIPT_STATE()
	}
	this.S.fromC(unsafe.Add(p, 2))
}

func (this *SCRIPT_CONTROL) toC(p unsafe.Pointer) {
	v := uint32(this.UDefaultLanguage) & 0xffff
	for i, b := range []bool{this.FContextDigits, this.FInvertPreBoundDir, this.FInvertPostBoundDir, this.FLinkStringBefore, this.FLinkStringAfter,
		this.FNeutralOverride, this.FNumericOverride, this.FLegacyBidiClass, this.FMergeNeutralItems, this.FUseStandardBidi} {
		v = setBit(v, uint(16+i), b)
	}
	*(*uint32)(p) = v
}

func (this *SCRIPT_CONTROL) fromC(p unsafe.Pointer) {
	v := *(*uint32)(p)
	this.UDefaultLanguage = int32(v & 0xffff)
	this.FContextDigits, this.FInvertPreBoundDir, this.FInvertPostBoundDir, this.FLinkStringBefore, this.FLinkStringAfter = bit(v, 16), bit(v, 17), bit(v, 18), bit(v, 19), bit(v, 20)
	this.FNeutralOverride, this.FNumericOverride, this.FLegacyBidiClass, this.FMergeNeutralItems, this.FUseStandardBidi = bit(v, 21), bit(v, 22), bit(v, 23), bit(v, 24), bit(v, 25)
}

func (this *SCRIPT_LOGATTR) toC(p unsafe.Pointer) {
	var v uint32
	for i, b := range []bool{this.FSoftBreak, this.FWhiteSpace, this.FCharStop, this.FWordStop, this.FInvalid} {
		v = setBit(v, uint(i), b)
	}
	*(*uint8)(p) = uint8(v)
}

func (this *SCRIPT_LOGATTR) fromC(p unsafe.Pointer) {
	v := uint32(*(*uint8)(p))
	this.FSoftBreak, this.FWhiteSpace, this.FCharStop, this.FWordStop, this.FInvalid = bit(v, 0), bit(v, 1), bit(v, 2), bit(v, 3), bit(v, 4)
}

func (this *SCRIPT_PROPERTIES) toC(p unsafe.Pointer) {
	v := uint32(this.Langid) & 0xffff
	for i, b := range []bool{this.FNumeric, this.FComplex, this.FNeedsWordBreaking, this.FNeedsCaretInfo} {
		v = setBit(v, uint(16+i), b)
	}
	v |= uint32(uint8(this.BCharSet)) << 20
	for i, b := range []bool{this.FControl, this.FPrivateUseArea, this.FNeedsCharacterJustify, this.FInvalidGlyph} {
		v = setBit(v, uint(28+i), b)
	}
	var w uint32
	for i, b := range []bool{this.FInvalidLogAttr, this.FCDM, this.FAmbiguousCharSet, this.FClusterSizeVaries, this.FRejectInvalid} {
		w = setBit(w, uint(i), b)
	}
	*(*uint32)(p), *(*uint32)(unsafe.Add(p, 4)) = v, w
}

func (this *SCRIPT_PROPERTIES) fromC(p unsafe.Pointer) {
	v, w := *(*uint32)(p), *(*uint32)(unsafe.Add(p, 4))
	this.Langid = int16(v & 0xffff)
	this.FNumeric, this.FComplex, this.FNeedsWordBreaking, this.FNeedsCaretInfo = bit(v, 16), bit(v, 17), bit(v, 18), bit(v, 19)
	this.BCharSet = int8(v >> 20)
	this.FControl, this.FPrivateUseArea, this.FNeedsCharacterJustify, this.FInvalidGlyph = bit(v, 28), bit(v, 29), bit(v, 30), bit(v, 31)
	this.FInvalidLogAttr, this.FCDM, this.FAmbiguousCharSet, this.FClusterSizeVaries, this.FRejectInvalid = bit(w, 0), bit(w, 1), bit(w, 2), bit(w, 3), bit(w, 4)
}

func (this *MENUBARINFO) toC(p unsafe.Pointer) {
	*(*int32)(p) = this.CbSize
	*(*int32)(unsafe.Add(p, 4)), *(*int32)(unsafe.Add(p, 8)), *(*int32)(unsafe.Add(p, 12)), *(*int32)(unsafe.Add(p, 16)) = this.Left, this.Top, this.Right, this.Bottom
	*(*int64)(unsafe.Add(p, 24)), *(*int64)(unsafe.Add(p, 32)) = this.HMenu, this.HwndMenu
	*(*uint32)(unsafe.Add(p, 40)) = setBit(setBit(0, 0, this.FBarFocused), 1, this.FFocused)
}

func (this *MENUBARINFO) fromC(p unsafe.Pointer) {
	this.CbSize = *(*int32)(p)
	this.Left, this.Top, this.Right, this.Bottom = *(*int32)(unsafe.Add(p, 4)), *(*int32)(unsafe.Add(p, 8)), *(*int32)(unsafe.Add(p, 12)), *(*int32)(unsafe.Add(p, 16))
	this.HMenu, this.HwndMenu = *(*int64)(unsafe.Add(p, 24)), *(*int64)(unsafe.Add(p, 32))
	v := *(*uint32)(unsafe.Add(p, 40))
	this.FBarFocused, this.FFocused = bit(v, 0), bit(v, 1)
}
