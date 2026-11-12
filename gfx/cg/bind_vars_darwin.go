//go:build darwin

package cg

import "unsafe"

type cgRect struct{ X, Y, W, H float64 }
type cgSize struct{ W, H float64 }
type cgPoint struct{ X, Y float64 }
type cgAffine struct{ A, B, C, D, Tx, Ty float64 }

// The CoreGraphics, CoreFoundation and CoreText entry points the rasterizer calls; contexts,
// colors, images and fonts travel as raw uintptr references. bind in bind_darwin.go (purego)
// or bind_cgo_darwin.go (cgo) fills them.
var (
	cgColorSpaceCreateDeviceRGB  func() uintptr
	cgColorSpaceRelease          func(cs uintptr)
	cgBitmapContextCreate        func(data uintptr, w, h, bpc, bpr uintptr, cs uintptr, info uint32) uintptr
	cgBitmapContextGetData       func(ctx uintptr) uintptr
	cgBitmapContextCreateImage   func(ctx uintptr) uintptr
	cgContextRelease             func(ctx uintptr)
	cgImageRelease               func(img uintptr)
	cgImageCreateWithImageInRct  func(img uintptr, r cgRect) uintptr
	cgDataProviderCreateWithData func(info, data, size, release uintptr) uintptr
	cgDataProviderRelease        func(provider uintptr)
	cgImageCreate                func(w, h, bpc, bpp, bpr, cs, info, provider uintptr) uintptr

	cgSaveGState  func(ctx uintptr)
	cgRestoreGSt  func(ctx uintptr)
	cgTranslate   func(ctx uintptr, x, y float64)
	cgScale       func(ctx uintptr, x, y float64)
	cgRotate      func(ctx uintptr, a float64)
	cgBeginPath   func(ctx uintptr)
	cgMoveTo      func(ctx uintptr, x, y float64)
	cgLineTo      func(ctx uintptr, x, y float64)
	cgQuadTo      func(ctx uintptr, cx, cy, x, y float64)
	cgCurveTo     func(ctx uintptr, c1x, c1y, c2x, c2y, x, y float64)
	cgClosePath   func(ctx uintptr)
	cgFillPath    func(ctx uintptr)
	cgStrokePath  func(ctx uintptr)
	cgFillColor   func(ctx uintptr, r, g, b, a float64)
	cgStrokeColor func(ctx uintptr, r, g, b, a float64)
	cgLineWidth   func(ctx uintptr, w float64)
	cgLineCap     func(ctx uintptr, c int32)
	cgLineJoin    func(ctx uintptr, j int32)
	cgMiterLimit  func(ctx uintptr, m float64)
	cgLineDash    func(ctx uintptr, phase float64, lengths *float64, n uintptr)
	cgSetShadow   func(ctx uintptr, off cgSize, blur float64, color uintptr)
	cgGetCTM      func(ctx uintptr) cgAffine
	cgColorCreate func(r, g, b, a float64) uintptr
	cgColorRelase func(c uintptr)
	cgClipRect    func(ctx uintptr, r cgRect)
	cgFillRect    func(ctx uintptr, r cgRect)
	cgClearRect   func(ctx uintptr, r cgRect)
	cgDrawImage   func(ctx uintptr, r cgRect, img uintptr)
	cgInterpolate func(ctx uintptr, q int32)
	cgFontSmooth  func(ctx uintptr, on bool)
	cgSetTextPos  func(ctx uintptr, x, y float64)
	cgSetAA       func(ctx uintptr, on bool)
	cgConcat      func(ctx uintptr, m cgAffine)
	cgClip        func(ctx uintptr)
	cgEOClip      func(ctx uintptr)
	cgEOFillPath  func(ctx uintptr)
	cgSetAlpha    func(ctx uintptr, a float64)
	cgBeginLayer  func(ctx uintptr, aux uintptr)
	cgEndLayer    func(ctx uintptr)
	cgStrokedPath func(ctx uintptr)
	cgGradCreate  func(cs uintptr, comps *float64, locs *float64, n uintptr) uintptr
	cgGradRelease func(g uintptr)
	cgDrawLinear  func(ctx uintptr, g uintptr, a, b cgPoint, opts uint32)
	cgDrawRadial  func(ctx uintptr, g uintptr, c0 cgPoint, r0 float64, c1 cgPoint, r1 float64, opts uint32)

	cfRelease               func(o uintptr)
	cfStringCreateWithBytes func(alloc uintptr, b *byte, n int, enc uint32, external bool) uintptr
	cfDictionaryCreate      func(alloc uintptr, keys, vals *uintptr, n int, kcb, vcb uintptr) uintptr
	cfAttrStringCreate      func(alloc uintptr, s uintptr, attrs uintptr) uintptr
	ctFontCreateWithName    func(name uintptr, size float64, m uintptr) uintptr
	ctLineCreateWithAttrStr func(as uintptr) uintptr
	ctLineDraw              func(line uintptr, ctx uintptr)
	ctLineTypographicBounds func(line uintptr, asc, desc, lead *float64) float64

	cfRetain       func(o uintptr) uintptr
	cfArrayCount   func(a uintptr) int
	cfArrayAt      func(a uintptr, i int) uintptr
	cfDictGet      func(d, k uintptr) uintptr
	cfNumberGet    func(n uintptr, typ int, p unsafe.Pointer) bool
	cfNumberCreate func(alloc uintptr, typ int, p unsafe.Pointer) uintptr
	ctDescCreate   func(attrs uintptr) uintptr
	ctDescMatching func(d, mandatory uintptr) uintptr
	ctDescCopyAttr func(d, attr uintptr) uintptr
	ctFontFromDesc func(d uintptr, size float64, m uintptr) uintptr
	ctUIFont       func(typ int32, size float64, lang uintptr) uintptr

	cfDataCreateMutable   func(alloc uintptr, capacity int) uintptr
	cfDataCreate          func(alloc uintptr, b unsafe.Pointer, n int) uintptr
	cfDataLength          func(d uintptr) int
	cfDataBytes           func(d uintptr) uintptr
	cgConsumerFromData    func(d uintptr) uintptr
	cgConsumerRelease     func(c uintptr)
	cgPDFContextCreate    func(consumer uintptr, mediaBox *cgRect, aux uintptr) uintptr
	cgPDFContextClose     func(ctx uintptr)
	cgPDFContextBeginPage func(ctx uintptr, info uintptr)
	cgPDFContextEndPage   func(ctx uintptr)

	kPDFMediaBox, kPDFCreator                    uintptr
	kFontAttr, kFgFromCtx, kTrue, kKeyCB, kValCB uintptr
	kFamilyAttr, kTraitsAttr, kWeightTrait       uintptr
	kSymbolicTrait, kKernAttr, kWidthTrait       uintptr
)
