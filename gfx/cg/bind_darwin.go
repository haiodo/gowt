//go:build darwin && (!cgo || !gowtcgo)

package cg

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
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
		reg(&cgGetCTM, "CGContextGetCTM")
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
		reg(&cfDataCreateMutable, "CFDataCreateMutable")
		reg(&cfDataCreate, "CFDataCreate")
		reg(&cfDataLength, "CFDataGetLength")
		reg(&cfDataBytes, "CFDataGetBytePtr")
		reg(&cgConsumerFromData, "CGDataConsumerCreateWithCFData")
		reg(&cgConsumerRelease, "CGDataConsumerRelease")
		reg(&cgPDFContextCreate, "CGPDFContextCreate")
		reg(&cgPDFContextClose, "CGPDFContextClose")
		reg(&cgPDFContextBeginPage, "CGPDFContextBeginPage")
		reg(&cgPDFContextEndPage, "CGPDFContextEndPage")
		reg(&cfRelease, "CFRelease")
		reg(&cfStringCreateWithBytes, "CFStringCreateWithBytes")
		reg(&cfDictionaryCreate, "CFDictionaryCreate")
		reg(&cfAttrStringCreate, "CFAttributedStringCreate")
		reg(&ctFontCreateWithName, "CTFontCreateWithName")
		reg(&ctLineCreateWithAttrStr, "CTLineCreateWithAttributedString")
		reg(&ctLineDraw, "CTLineDraw")
		reg(&cfRetain, "CFRetain")
		reg(&cfArrayCount, "CFArrayGetCount")
		reg(&cfArrayAt, "CFArrayGetValueAtIndex")
		reg(&cfDictGet, "CFDictionaryGetValue")
		reg(&cfNumberGet, "CFNumberGetValue")
		reg(&cfNumberCreate, "CFNumberCreate")
		reg(&ctDescCreate, "CTFontDescriptorCreateWithAttributes")
		reg(&ctDescMatching, "CTFontDescriptorCreateMatchingFontDescriptors")
		reg(&ctDescCopyAttr, "CTFontDescriptorCopyAttribute")
		reg(&ctFontFromDesc, "CTFontCreateWithFontDescriptor")
		reg(&ctUIFont, "CTFontCreateUIFontForLanguage")
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
		kPDFMediaBox = deref(sym("kCGPDFContextMediaBox"))
		kPDFCreator = deref(sym("kCGPDFContextCreator"))
		kFontAttr = deref(sym("kCTFontAttributeName"))
		kFgFromCtx = deref(sym("kCTForegroundColorFromContextAttributeName"))
		kTrue = deref(sym("kCFBooleanTrue"))
		kFamilyAttr = deref(sym("kCTFontFamilyNameAttribute"))
		kTraitsAttr = deref(sym("kCTFontTraitsAttribute"))
		kWeightTrait = deref(sym("kCTFontWeightTrait"))
		kSymbolicTrait = deref(sym("kCTFontSymbolicTrait"))
		kKernAttr = deref(sym("kCTKernAttributeName"))
		kWidthTrait = deref(sym("kCTFontWidthTrait"))
		kKeyCB = sym("kCFTypeDictionaryKeyCallBacks")
		kValCB = sym("kCFTypeDictionaryValueCallBacks")
	})
}
