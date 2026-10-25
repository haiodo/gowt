//go:build darwin

package cg

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

type cgRect struct{ X, Y, W, H float64 }
type cgSize struct{ W, H float64 }
type cgPoint struct{ X, Y float64 }
type cgAffine struct{ A, B, C, D, Tx, Ty float64 }

// The CoreGraphics, CoreFoundation and CoreText entry points the rasterizer calls; contexts,
// colors, images and fonts travel as raw uintptr references.
var (
	cgColorSpaceCreateDeviceRGB func() uintptr
	cgColorSpaceRelease         func(cs uintptr)
	cgBitmapContextCreate       func(data uintptr, w, h, bpc, bpr uintptr, cs uintptr, info uint32) uintptr
	cgBitmapContextGetData      func(ctx uintptr) uintptr
	cgBitmapContextCreateImage  func(ctx uintptr) uintptr
	cgContextRelease            func(ctx uintptr)
	cgImageRelease              func(img uintptr)
	cgImageCreateWithImageInRct func(img uintptr, r cgRect) uintptr

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

	kFontAttr, kFgFromCtx, kTrue, kKeyCB, kValCB uintptr
)

var bindOnce sync.Once

func bind() {
	bindOnce.Do(func() {
		for _, p := range []string{
			"/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
			"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
			"/System/Library/Frameworks/CoreText.framework/CoreText",
		} {
			if _, err := purego.Dlopen(p, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				panic("gfx/cg: dlopen " + p + ": " + err.Error())
			}
		}
		reg := func(fn any, name string) { purego.RegisterLibFunc(fn, purego.RTLD_DEFAULT, name) }
		reg(&cgColorSpaceCreateDeviceRGB, "CGColorSpaceCreateDeviceRGB")
		reg(&cgColorSpaceRelease, "CGColorSpaceRelease")
		reg(&cgBitmapContextCreate, "CGBitmapContextCreate")
		reg(&cgBitmapContextGetData, "CGBitmapContextGetData")
		reg(&cgBitmapContextCreateImage, "CGBitmapContextCreateImage")
		reg(&cgContextRelease, "CGContextRelease")
		reg(&cgImageRelease, "CGImageRelease")
		reg(&cgImageCreateWithImageInRct, "CGImageCreateWithImageInRect")
		reg(&cgSaveGState, "CGContextSaveGState")
		reg(&cgRestoreGSt, "CGContextRestoreGState")
		reg(&cgTranslate, "CGContextTranslateCTM")
		reg(&cgScale, "CGContextScaleCTM")
		reg(&cgRotate, "CGContextRotateCTM")
		reg(&cgBeginPath, "CGContextBeginPath")
		reg(&cgMoveTo, "CGContextMoveToPoint")
		reg(&cgLineTo, "CGContextAddLineToPoint")
		reg(&cgQuadTo, "CGContextAddQuadCurveToPoint")
		reg(&cgCurveTo, "CGContextAddCurveToPoint")
		reg(&cgClosePath, "CGContextClosePath")
		reg(&cgFillPath, "CGContextFillPath")
		reg(&cgStrokePath, "CGContextStrokePath")
		reg(&cgFillColor, "CGContextSetRGBFillColor")
		reg(&cgStrokeColor, "CGContextSetRGBStrokeColor")
		reg(&cgLineWidth, "CGContextSetLineWidth")
		reg(&cgLineCap, "CGContextSetLineCap")
		reg(&cgLineJoin, "CGContextSetLineJoin")
		reg(&cgMiterLimit, "CGContextSetMiterLimit")
		reg(&cgLineDash, "CGContextSetLineDash")
		reg(&cgSetShadow, "CGContextSetShadowWithColor")
		reg(&cgColorCreate, "CGColorCreateGenericRGB")
		reg(&cgColorRelase, "CGColorRelease")
		reg(&cgClipRect, "CGContextClipToRect")
		reg(&cgFillRect, "CGContextFillRect")
		reg(&cgClearRect, "CGContextClearRect")
		reg(&cgDrawImage, "CGContextDrawImage")
		reg(&cgInterpolate, "CGContextSetInterpolationQuality")
		reg(&cgFontSmooth, "CGContextSetShouldSmoothFonts")
		reg(&cgSetTextPos, "CGContextSetTextPosition")
		reg(&cgSetAA, "CGContextSetShouldAntialias")
		reg(&cgConcat, "CGContextConcatCTM")
		reg(&cgClip, "CGContextClip")
		reg(&cgEOClip, "CGContextEOClip")
		reg(&cgEOFillPath, "CGContextEOFillPath")
		reg(&cgSetAlpha, "CGContextSetAlpha")
		reg(&cgBeginLayer, "CGContextBeginTransparencyLayer")
		reg(&cgEndLayer, "CGContextEndTransparencyLayer")
		reg(&cgStrokedPath, "CGContextReplacePathWithStrokedPath")
		reg(&cgGradCreate, "CGGradientCreateWithColorComponents")
		reg(&cgGradRelease, "CGGradientRelease")
		reg(&cgDrawLinear, "CGContextDrawLinearGradient")
		reg(&cgDrawRadial, "CGContextDrawRadialGradient")
		reg(&cfRelease, "CFRelease")
		reg(&cfStringCreateWithBytes, "CFStringCreateWithBytes")
		reg(&cfDictionaryCreate, "CFDictionaryCreate")
		reg(&cfAttrStringCreate, "CFAttributedStringCreate")
		reg(&ctFontCreateWithName, "CTFontCreateWithName")
		reg(&ctLineCreateWithAttrStr, "CTLineCreateWithAttributedString")
		reg(&ctLineDraw, "CTLineDraw")
		reg(&ctLineTypographicBounds, "CTLineGetTypographicBounds")

		sym := func(name string) uintptr {
			a, err := purego.Dlsym(purego.RTLD_DEFAULT, name)
			if err != nil {
				panic("gfx/cg: dlsym " + name + ": " + err.Error())
			}
			return a
		}
		// the constants are pointers stored in the symbol; the callback tables are the symbol itself
		deref := func(a uintptr) uintptr { return *(*uintptr)(*(*unsafe.Pointer)(unsafe.Pointer(&a))) }
		kFontAttr = deref(sym("kCTFontAttributeName"))
		kFgFromCtx = deref(sym("kCTForegroundColorFromContextAttributeName"))
		kTrue = deref(sym("kCFBooleanTrue"))
		kKeyCB = sym("kCFTypeDictionaryKeyCallBacks")
		kValCB = sym("kCFTypeDictionaryValueCallBacks")
	})
}
