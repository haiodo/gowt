//go:build darwin && cgo && gowtcgo

package cg

/*
#cgo LDFLAGS: -framework CoreFoundation -framework CoreGraphics -framework CoreText
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <CoreText/CoreText.h>
#include <stdint.h>

// CGFloat is a double on 64-bit macOS, so these mirror the Go cg* types field for field.
// Structs with float fields may not cross the Go/C boundary by value, so they travel by pointer.
typedef struct { double x, y, w, h; } wtRect;
typedef struct { double w, h; } wtSize;
typedef struct { double x, y; } wtPoint;
typedef struct { double a, b, c, d, tx, ty; } wtAffine;

static CGRect wtCGRect(const wtRect *r) { return (CGRect){{r->x, r->y}, {r->w, r->h}}; }
static CGSize wtCGSize(const wtSize *s) { return (CGSize){s->w, s->h}; }
static CGPoint wtCGPoint(const wtPoint *p) { return (CGPoint){p->x, p->y}; }
static CGAffineTransform wtCGAffine(const wtAffine *m) {
	return (CGAffineTransform){m->a, m->b, m->c, m->d, m->tx, m->ty};
}

static uintptr_t wtColorSpaceCreateDeviceRGB(void) { return (uintptr_t)CGColorSpaceCreateDeviceRGB(); }
static void wtColorSpaceRelease(uintptr_t cs) { CGColorSpaceRelease((CGColorSpaceRef)cs); }
static uintptr_t wtBitmapContextCreate(uintptr_t data, uintptr_t w, uintptr_t h, uintptr_t bpc, uintptr_t bpr, uintptr_t cs, uint32_t info) {
	return (uintptr_t)CGBitmapContextCreate((void *)data, w, h, bpc, bpr, (CGColorSpaceRef)cs, (CGBitmapInfo)info);
}
static uintptr_t wtBitmapContextGetData(uintptr_t ctx) { return (uintptr_t)CGBitmapContextGetData((CGContextRef)ctx); }
static uintptr_t wtBitmapContextCreateImage(uintptr_t ctx) { return (uintptr_t)CGBitmapContextCreateImage((CGContextRef)ctx); }
static void wtContextRelease(uintptr_t ctx) { CGContextRelease((CGContextRef)ctx); }
static void wtImageRelease(uintptr_t img) { CGImageRelease((CGImageRef)img); }
static uintptr_t wtImageInRect(uintptr_t img, const wtRect *r) {
	return (uintptr_t)CGImageCreateWithImageInRect((CGImageRef)img, wtCGRect(r));
}

static void wtSaveGState(uintptr_t ctx) { CGContextSaveGState((CGContextRef)ctx); }
static void wtRestoreGState(uintptr_t ctx) { CGContextRestoreGState((CGContextRef)ctx); }
static void wtTranslateCTM(uintptr_t ctx, double x, double y) { CGContextTranslateCTM((CGContextRef)ctx, x, y); }
static void wtScaleCTM(uintptr_t ctx, double x, double y) { CGContextScaleCTM((CGContextRef)ctx, x, y); }
static void wtRotateCTM(uintptr_t ctx, double a) { CGContextRotateCTM((CGContextRef)ctx, a); }
static void wtBeginPath(uintptr_t ctx) { CGContextBeginPath((CGContextRef)ctx); }
static void wtMoveToPoint(uintptr_t ctx, double x, double y) { CGContextMoveToPoint((CGContextRef)ctx, x, y); }
static void wtAddLineToPoint(uintptr_t ctx, double x, double y) { CGContextAddLineToPoint((CGContextRef)ctx, x, y); }
static void wtAddQuadCurveToPoint(uintptr_t ctx, double cx, double cy, double x, double y) {
	CGContextAddQuadCurveToPoint((CGContextRef)ctx, cx, cy, x, y);
}
static void wtAddCurveToPoint(uintptr_t ctx, double c1x, double c1y, double c2x, double c2y, double x, double y) {
	CGContextAddCurveToPoint((CGContextRef)ctx, c1x, c1y, c2x, c2y, x, y);
}
static void wtClosePath(uintptr_t ctx) { CGContextClosePath((CGContextRef)ctx); }
static void wtFillPath(uintptr_t ctx) { CGContextFillPath((CGContextRef)ctx); }
static void wtStrokePath(uintptr_t ctx) { CGContextStrokePath((CGContextRef)ctx); }
static void wtSetRGBFillColor(uintptr_t ctx, double r, double g, double b, double a) {
	CGContextSetRGBFillColor((CGContextRef)ctx, r, g, b, a);
}
static void wtSetRGBStrokeColor(uintptr_t ctx, double r, double g, double b, double a) {
	CGContextSetRGBStrokeColor((CGContextRef)ctx, r, g, b, a);
}
static void wtSetLineWidth(uintptr_t ctx, double w) { CGContextSetLineWidth((CGContextRef)ctx, w); }
static void wtSetLineCap(uintptr_t ctx, int32_t c) { CGContextSetLineCap((CGContextRef)ctx, (CGLineCap)c); }
static void wtSetLineJoin(uintptr_t ctx, int32_t j) { CGContextSetLineJoin((CGContextRef)ctx, (CGLineJoin)j); }
static void wtSetMiterLimit(uintptr_t ctx, double m) { CGContextSetMiterLimit((CGContextRef)ctx, m); }
static void wtSetLineDash(uintptr_t ctx, double phase, const double *lengths, uintptr_t n) {
	CGContextSetLineDash((CGContextRef)ctx, phase, lengths, n);
}
static void wtSetShadowWithColor(uintptr_t ctx, const wtSize *off, double blur, uintptr_t color) {
	CGContextSetShadowWithColor((CGContextRef)ctx, wtCGSize(off), blur, (CGColorRef)color);
}
static void wtGetCTM(uintptr_t ctx, wtAffine *out) {
	CGAffineTransform m = CGContextGetCTM((CGContextRef)ctx);
	out->a = m.a; out->b = m.b; out->c = m.c; out->d = m.d; out->tx = m.tx; out->ty = m.ty;
}
static uintptr_t wtColorCreateGenericRGB(double r, double g, double b, double a) {
	return (uintptr_t)CGColorCreateGenericRGB(r, g, b, a);
}
static void wtColorRelease(uintptr_t c) { CGColorRelease((CGColorRef)c); }
static void wtClipToRect(uintptr_t ctx, const wtRect *r) { CGContextClipToRect((CGContextRef)ctx, wtCGRect(r)); }
static void wtFillRect(uintptr_t ctx, const wtRect *r) { CGContextFillRect((CGContextRef)ctx, wtCGRect(r)); }
static void wtClearRect(uintptr_t ctx, const wtRect *r) { CGContextClearRect((CGContextRef)ctx, wtCGRect(r)); }
static void wtDrawImage(uintptr_t ctx, const wtRect *r, uintptr_t img) {
	CGContextDrawImage((CGContextRef)ctx, wtCGRect(r), (CGImageRef)img);
}
static void wtSetInterpolationQuality(uintptr_t ctx, int32_t q) {
	CGContextSetInterpolationQuality((CGContextRef)ctx, (CGInterpolationQuality)q);
}
static void wtSetShouldSmoothFonts(uintptr_t ctx, uint8_t on) {
	CGContextSetShouldSmoothFonts((CGContextRef)ctx, on != 0);
}
static void wtSetTextPosition(uintptr_t ctx, double x, double y) { CGContextSetTextPosition((CGContextRef)ctx, x, y); }
static void wtSetShouldAntialias(uintptr_t ctx, uint8_t on) {
	CGContextSetShouldAntialias((CGContextRef)ctx, on != 0);
}
static void wtConcatCTM(uintptr_t ctx, const wtAffine *m) { CGContextConcatCTM((CGContextRef)ctx, wtCGAffine(m)); }
static void wtClip(uintptr_t ctx) { CGContextClip((CGContextRef)ctx); }
static void wtEOClip(uintptr_t ctx) { CGContextEOClip((CGContextRef)ctx); }
static void wtEOFillPath(uintptr_t ctx) { CGContextEOFillPath((CGContextRef)ctx); }
static void wtSetAlpha(uintptr_t ctx, double a) { CGContextSetAlpha((CGContextRef)ctx, a); }
static void wtBeginTransparencyLayer(uintptr_t ctx, uintptr_t aux) {
	CGContextBeginTransparencyLayer((CGContextRef)ctx, (CFDictionaryRef)aux);
}
static void wtEndTransparencyLayer(uintptr_t ctx) { CGContextEndTransparencyLayer((CGContextRef)ctx); }
static void wtReplacePathWithStrokedPath(uintptr_t ctx) { CGContextReplacePathWithStrokedPath((CGContextRef)ctx); }
static uintptr_t wtGradientCreate(uintptr_t cs, const double *comps, const double *locs, uintptr_t n) {
	return (uintptr_t)CGGradientCreateWithColorComponents((CGColorSpaceRef)cs, comps, locs, n);
}
static void wtGradientRelease(uintptr_t g) { CGGradientRelease((CGGradientRef)g); }
static void wtDrawLinearGradient(uintptr_t ctx, uintptr_t g, const wtPoint *a, const wtPoint *b, uint32_t opts) {
	CGContextDrawLinearGradient((CGContextRef)ctx, (CGGradientRef)g, wtCGPoint(a), wtCGPoint(b), (CGGradientDrawingOptions)opts);
}
static void wtDrawRadialGradient(uintptr_t ctx, uintptr_t g, const wtPoint *c0, double r0, const wtPoint *c1, double r1, uint32_t opts) {
	CGContextDrawRadialGradient((CGContextRef)ctx, (CGGradientRef)g, wtCGPoint(c0), r0, wtCGPoint(c1), r1, (CGGradientDrawingOptions)opts);
}

static uintptr_t wtCFDataCreateMutable(uintptr_t alloc, int capacity) {
	return (uintptr_t)CFDataCreateMutable((CFAllocatorRef)alloc, (CFIndex)capacity);
}
static uintptr_t wtCFDataCreate(uintptr_t alloc, const void *b, int n) {
	return (uintptr_t)CFDataCreate((CFAllocatorRef)alloc, (const UInt8 *)b, (CFIndex)n);
}
static int wtCFDataGetLength(uintptr_t d) { return (int)CFDataGetLength((CFDataRef)d); }
static uintptr_t wtCFDataGetBytePtr(uintptr_t d) { return (uintptr_t)CFDataGetBytePtr((CFDataRef)d); }
static uintptr_t wtDataConsumerCreateWithCFData(uintptr_t d) {
	return (uintptr_t)CGDataConsumerCreateWithCFData((CFMutableDataRef)d);
}
static void wtDataConsumerRelease(uintptr_t c) { CGDataConsumerRelease((CGDataConsumerRef)c); }
static uintptr_t wtPDFContextCreate(uintptr_t consumer, const wtRect *mediaBox, uintptr_t aux) {
	if (mediaBox) {
		CGRect r = wtCGRect(mediaBox);
		return (uintptr_t)CGPDFContextCreate((CGDataConsumerRef)consumer, &r, (CFDictionaryRef)aux);
	}
	return (uintptr_t)CGPDFContextCreate((CGDataConsumerRef)consumer, NULL, (CFDictionaryRef)aux);
}
static void wtPDFContextClose(uintptr_t ctx) { CGPDFContextClose((CGContextRef)ctx); }
static void wtPDFContextBeginPage(uintptr_t ctx, uintptr_t info) {
	CGPDFContextBeginPage((CGContextRef)ctx, (CFDictionaryRef)info);
}
static void wtPDFContextEndPage(uintptr_t ctx) { CGPDFContextEndPage((CGContextRef)ctx); }

static void wtCFRelease(uintptr_t o) { CFRelease((CFTypeRef)o); }
static uintptr_t wtCFStringCreateWithBytes(uintptr_t alloc, const void *b, int n, uint32_t enc, uint8_t external) {
	return (uintptr_t)CFStringCreateWithBytes((CFAllocatorRef)alloc, (const UInt8 *)b, (CFIndex)n, (CFStringEncoding)enc, external != 0);
}
static uintptr_t wtCFDictionaryCreate(uintptr_t alloc, const uintptr_t *keys, const uintptr_t *vals, int n, uintptr_t kcb, uintptr_t vcb) {
	return (uintptr_t)CFDictionaryCreate((CFAllocatorRef)alloc, (const void **)keys, (const void **)vals, (CFIndex)n,
		(const CFDictionaryKeyCallBacks *)kcb, (const CFDictionaryValueCallBacks *)vcb);
}
static uintptr_t wtCFAttributedStringCreate(uintptr_t alloc, uintptr_t s, uintptr_t attrs) {
	return (uintptr_t)CFAttributedStringCreate((CFAllocatorRef)alloc, (CFStringRef)s, (CFDictionaryRef)attrs);
}
static uintptr_t wtCTFontCreateWithName(uintptr_t name, double size, uintptr_t m) {
	return (uintptr_t)CTFontCreateWithName((CFStringRef)name, (CGFloat)size, (const CGAffineTransform *)m);
}
static uintptr_t wtCTLineCreateWithAttributedString(uintptr_t as) {
	return (uintptr_t)CTLineCreateWithAttributedString((CFAttributedStringRef)as);
}
static void wtCTLineDraw(uintptr_t line, uintptr_t ctx) { CTLineDraw((CTLineRef)line, (CGContextRef)ctx); }
static double wtCTLineGetTypographicBounds(uintptr_t line, double *asc, double *desc, double *lead) {
	return (double)CTLineGetTypographicBounds((CTLineRef)line, (CGFloat *)asc, (CGFloat *)desc, (CGFloat *)lead);
}

static uintptr_t wtCFRetain(uintptr_t o) { return (uintptr_t)CFRetain((CFTypeRef)o); }
static int wtCFArrayGetCount(uintptr_t a) { return (int)CFArrayGetCount((CFArrayRef)a); }
static uintptr_t wtCFArrayGetValueAtIndex(uintptr_t a, int i) {
	return (uintptr_t)CFArrayGetValueAtIndex((CFArrayRef)a, (CFIndex)i);
}
static uintptr_t wtCFDictionaryGetValue(uintptr_t d, uintptr_t k) {
	return (uintptr_t)CFDictionaryGetValue((CFDictionaryRef)d, (const void *)k);
}
static uint8_t wtCFNumberGetValue(uintptr_t n, int typ, void *p) {
	return CFNumberGetValue((CFNumberRef)n, (CFNumberType)typ, p) != 0;
}
static uintptr_t wtCFNumberCreate(uintptr_t alloc, int typ, const void *p) {
	return (uintptr_t)CFNumberCreate((CFAllocatorRef)alloc, (CFNumberType)typ, p);
}
static uintptr_t wtCTFontDescriptorCreateWithAttributes(uintptr_t attrs) {
	return (uintptr_t)CTFontDescriptorCreateWithAttributes((CFDictionaryRef)attrs);
}
static uintptr_t wtCTFontDescriptorCreateMatchingFontDescriptors(uintptr_t d, uintptr_t mandatory) {
	return (uintptr_t)CTFontDescriptorCreateMatchingFontDescriptors((CTFontDescriptorRef)d, (CFSetRef)mandatory);
}
static uintptr_t wtCTFontDescriptorCopyAttribute(uintptr_t d, uintptr_t attr) {
	return (uintptr_t)CTFontDescriptorCopyAttribute((CTFontDescriptorRef)d, (CFStringRef)attr);
}
static uintptr_t wtCTFontCreateWithFontDescriptor(uintptr_t d, double size, uintptr_t m) {
	return (uintptr_t)CTFontCreateWithFontDescriptor((CTFontDescriptorRef)d, (CGFloat)size, (const CGAffineTransform *)m);
}
static uintptr_t wtCTFontCreateUIFontForLanguage(int32_t typ, double size, uintptr_t lang) {
	return (uintptr_t)CTFontCreateUIFontForLanguage((CTFontUIFontType)typ, (CGFloat)size, (CFStringRef)lang);
}

static uintptr_t wtPDFMediaBox(void) { return (uintptr_t)kCGPDFContextMediaBox; }
static uintptr_t wtPDFCreator(void) { return (uintptr_t)kCGPDFContextCreator; }
static uintptr_t wtFontAttr(void) { return (uintptr_t)kCTFontAttributeName; }
static uintptr_t wtFgFromCtx(void) { return (uintptr_t)kCTForegroundColorFromContextAttributeName; }
static uintptr_t wtTrue(void) { return (uintptr_t)kCFBooleanTrue; }
static uintptr_t wtFamilyAttr(void) { return (uintptr_t)kCTFontFamilyNameAttribute; }
static uintptr_t wtTraitsAttr(void) { return (uintptr_t)kCTFontTraitsAttribute; }
static uintptr_t wtWeightTrait(void) { return (uintptr_t)kCTFontWeightTrait; }
static uintptr_t wtSymbolicTrait(void) { return (uintptr_t)kCTFontSymbolicTrait; }
static uintptr_t wtKernAttr(void) { return (uintptr_t)kCTKernAttributeName; }
static uintptr_t wtWidthTrait(void) { return (uintptr_t)kCTFontWidthTrait; }
static uintptr_t wtKeyCB(void) { return (uintptr_t)&kCFTypeDictionaryKeyCallBacks; }
static uintptr_t wtValCB(void) { return (uintptr_t)&kCFTypeDictionaryValueCallBacks; }
*/
import "C"

import (
	"sync"
	"unsafe"
)

var bindOnce sync.Once

// wtRectP, wtSizeP, wtPointP, wtAffineP cast a Go cg* value to its C twin: the layouts
// match field for field and only a pointer crosses the boundary.
func wtRectP(r *cgRect) *C.wtRect       { return (*C.wtRect)(unsafe.Pointer(r)) }
func wtSizeP(s *cgSize) *C.wtSize       { return (*C.wtSize)(unsafe.Pointer(s)) }
func wtPointP(p *cgPoint) *C.wtPoint    { return (*C.wtPoint)(unsafe.Pointer(p)) }
func wtAffineP(m *cgAffine) *C.wtAffine { return (*C.wtAffine)(unsafe.Pointer(m)) }

func wtBool(b bool) C.uint8_t {
	if b {
		return 1
	}
	return 0
}

func bind() {
	bindOnce.Do(func() {
		cgColorSpaceCreateDeviceRGB = func() uintptr { return uintptr(C.wtColorSpaceCreateDeviceRGB()) }
		cgColorSpaceRelease = func(cs uintptr) { C.wtColorSpaceRelease(C.uintptr_t(cs)) }
		cgBitmapContextCreate = func(data uintptr, w, h, bpc, bpr uintptr, cs uintptr, info uint32) uintptr {
			return uintptr(C.wtBitmapContextCreate(C.uintptr_t(data), C.uintptr_t(w), C.uintptr_t(h), C.uintptr_t(bpc), C.uintptr_t(bpr), C.uintptr_t(cs), C.uint32_t(info)))
		}
		cgBitmapContextGetData = func(ctx uintptr) uintptr { return uintptr(C.wtBitmapContextGetData(C.uintptr_t(ctx))) }
		cgBitmapContextCreateImage = func(ctx uintptr) uintptr {
			return uintptr(C.wtBitmapContextCreateImage(C.uintptr_t(ctx)))
		}
		cgContextRelease = func(ctx uintptr) { C.wtContextRelease(C.uintptr_t(ctx)) }
		cgImageRelease = func(img uintptr) { C.wtImageRelease(C.uintptr_t(img)) }
		cgImageCreateWithImageInRct = func(img uintptr, r cgRect) uintptr {
			return uintptr(C.wtImageInRect(C.uintptr_t(img), wtRectP(&r)))
		}

		cgSaveGState = func(ctx uintptr) { C.wtSaveGState(C.uintptr_t(ctx)) }
		cgRestoreGSt = func(ctx uintptr) { C.wtRestoreGState(C.uintptr_t(ctx)) }
		cgTranslate = func(ctx uintptr, x, y float64) { C.wtTranslateCTM(C.uintptr_t(ctx), C.double(x), C.double(y)) }
		cgScale = func(ctx uintptr, x, y float64) { C.wtScaleCTM(C.uintptr_t(ctx), C.double(x), C.double(y)) }
		cgRotate = func(ctx uintptr, a float64) { C.wtRotateCTM(C.uintptr_t(ctx), C.double(a)) }
		cgBeginPath = func(ctx uintptr) { C.wtBeginPath(C.uintptr_t(ctx)) }
		cgMoveTo = func(ctx uintptr, x, y float64) { C.wtMoveToPoint(C.uintptr_t(ctx), C.double(x), C.double(y)) }
		cgLineTo = func(ctx uintptr, x, y float64) { C.wtAddLineToPoint(C.uintptr_t(ctx), C.double(x), C.double(y)) }
		cgQuadTo = func(ctx uintptr, cx, cy, x, y float64) {
			C.wtAddQuadCurveToPoint(C.uintptr_t(ctx), C.double(cx), C.double(cy), C.double(x), C.double(y))
		}
		cgCurveTo = func(ctx uintptr, c1x, c1y, c2x, c2y, x, y float64) {
			C.wtAddCurveToPoint(C.uintptr_t(ctx), C.double(c1x), C.double(c1y), C.double(c2x), C.double(c2y), C.double(x), C.double(y))
		}
		cgClosePath = func(ctx uintptr) { C.wtClosePath(C.uintptr_t(ctx)) }
		cgFillPath = func(ctx uintptr) { C.wtFillPath(C.uintptr_t(ctx)) }
		cgStrokePath = func(ctx uintptr) { C.wtStrokePath(C.uintptr_t(ctx)) }
		cgFillColor = func(ctx uintptr, r, g, b, a float64) {
			C.wtSetRGBFillColor(C.uintptr_t(ctx), C.double(r), C.double(g), C.double(b), C.double(a))
		}
		cgStrokeColor = func(ctx uintptr, r, g, b, a float64) {
			C.wtSetRGBStrokeColor(C.uintptr_t(ctx), C.double(r), C.double(g), C.double(b), C.double(a))
		}
		cgLineWidth = func(ctx uintptr, w float64) { C.wtSetLineWidth(C.uintptr_t(ctx), C.double(w)) }
		cgLineCap = func(ctx uintptr, c int32) { C.wtSetLineCap(C.uintptr_t(ctx), C.int32_t(c)) }
		cgLineJoin = func(ctx uintptr, j int32) { C.wtSetLineJoin(C.uintptr_t(ctx), C.int32_t(j)) }
		cgMiterLimit = func(ctx uintptr, m float64) { C.wtSetMiterLimit(C.uintptr_t(ctx), C.double(m)) }
		cgLineDash = func(ctx uintptr, phase float64, lengths *float64, n uintptr) {
			C.wtSetLineDash(C.uintptr_t(ctx), C.double(phase), (*C.double)(unsafe.Pointer(lengths)), C.uintptr_t(n))
		}
		cgSetShadow = func(ctx uintptr, off cgSize, blur float64, color uintptr) {
			C.wtSetShadowWithColor(C.uintptr_t(ctx), wtSizeP(&off), C.double(blur), C.uintptr_t(color))
		}
		cgGetCTM = func(ctx uintptr) cgAffine {
			var m C.wtAffine
			C.wtGetCTM(C.uintptr_t(ctx), &m)
			return *(*cgAffine)(unsafe.Pointer(&m))
		}
		cgColorCreate = func(r, g, b, a float64) uintptr {
			return uintptr(C.wtColorCreateGenericRGB(C.double(r), C.double(g), C.double(b), C.double(a)))
		}
		cgColorRelase = func(c uintptr) { C.wtColorRelease(C.uintptr_t(c)) }
		cgClipRect = func(ctx uintptr, r cgRect) { C.wtClipToRect(C.uintptr_t(ctx), wtRectP(&r)) }
		cgFillRect = func(ctx uintptr, r cgRect) { C.wtFillRect(C.uintptr_t(ctx), wtRectP(&r)) }
		cgClearRect = func(ctx uintptr, r cgRect) { C.wtClearRect(C.uintptr_t(ctx), wtRectP(&r)) }
		cgDrawImage = func(ctx uintptr, r cgRect, img uintptr) {
			C.wtDrawImage(C.uintptr_t(ctx), wtRectP(&r), C.uintptr_t(img))
		}
		cgInterpolate = func(ctx uintptr, q int32) { C.wtSetInterpolationQuality(C.uintptr_t(ctx), C.int32_t(q)) }
		cgFontSmooth = func(ctx uintptr, on bool) { C.wtSetShouldSmoothFonts(C.uintptr_t(ctx), wtBool(on)) }
		cgSetTextPos = func(ctx uintptr, x, y float64) { C.wtSetTextPosition(C.uintptr_t(ctx), C.double(x), C.double(y)) }
		cgSetAA = func(ctx uintptr, on bool) { C.wtSetShouldAntialias(C.uintptr_t(ctx), wtBool(on)) }
		cgConcat = func(ctx uintptr, m cgAffine) { C.wtConcatCTM(C.uintptr_t(ctx), wtAffineP(&m)) }
		cgClip = func(ctx uintptr) { C.wtClip(C.uintptr_t(ctx)) }
		cgEOClip = func(ctx uintptr) { C.wtEOClip(C.uintptr_t(ctx)) }
		cgEOFillPath = func(ctx uintptr) { C.wtEOFillPath(C.uintptr_t(ctx)) }
		cgSetAlpha = func(ctx uintptr, a float64) { C.wtSetAlpha(C.uintptr_t(ctx), C.double(a)) }
		cgBeginLayer = func(ctx uintptr, aux uintptr) { C.wtBeginTransparencyLayer(C.uintptr_t(ctx), C.uintptr_t(aux)) }
		cgEndLayer = func(ctx uintptr) { C.wtEndTransparencyLayer(C.uintptr_t(ctx)) }
		cgStrokedPath = func(ctx uintptr) { C.wtReplacePathWithStrokedPath(C.uintptr_t(ctx)) }
		cgGradCreate = func(cs uintptr, comps *float64, locs *float64, n uintptr) uintptr {
			return uintptr(C.wtGradientCreate(C.uintptr_t(cs), (*C.double)(unsafe.Pointer(comps)), (*C.double)(unsafe.Pointer(locs)), C.uintptr_t(n)))
		}
		cgGradRelease = func(g uintptr) { C.wtGradientRelease(C.uintptr_t(g)) }
		cgDrawLinear = func(ctx uintptr, g uintptr, a, b cgPoint, opts uint32) {
			C.wtDrawLinearGradient(C.uintptr_t(ctx), C.uintptr_t(g), wtPointP(&a), wtPointP(&b), C.uint32_t(opts))
		}
		cgDrawRadial = func(ctx uintptr, g uintptr, c0 cgPoint, r0 float64, c1 cgPoint, r1 float64, opts uint32) {
			C.wtDrawRadialGradient(C.uintptr_t(ctx), C.uintptr_t(g), wtPointP(&c0), C.double(r0), wtPointP(&c1), C.double(r1), C.uint32_t(opts))
		}

		cfDataCreateMutable = func(alloc uintptr, capacity int) uintptr {
			return uintptr(C.wtCFDataCreateMutable(C.uintptr_t(alloc), C.int(capacity)))
		}
		cfDataCreate = func(alloc uintptr, b unsafe.Pointer, n int) uintptr {
			return uintptr(C.wtCFDataCreate(C.uintptr_t(alloc), b, C.int(n)))
		}
		cfDataLength = func(d uintptr) int { return int(C.wtCFDataGetLength(C.uintptr_t(d))) }
		cfDataBytes = func(d uintptr) uintptr { return uintptr(C.wtCFDataGetBytePtr(C.uintptr_t(d))) }
		cgConsumerFromData = func(d uintptr) uintptr { return uintptr(C.wtDataConsumerCreateWithCFData(C.uintptr_t(d))) }
		cgConsumerRelease = func(c uintptr) { C.wtDataConsumerRelease(C.uintptr_t(c)) }
		cgPDFContextCreate = func(consumer uintptr, mediaBox *cgRect, aux uintptr) uintptr {
			var mb *C.wtRect
			if mediaBox != nil {
				mb = wtRectP(mediaBox)
			}
			return uintptr(C.wtPDFContextCreate(C.uintptr_t(consumer), mb, C.uintptr_t(aux)))
		}
		cgPDFContextClose = func(ctx uintptr) { C.wtPDFContextClose(C.uintptr_t(ctx)) }
		cgPDFContextBeginPage = func(ctx uintptr, info uintptr) {
			C.wtPDFContextBeginPage(C.uintptr_t(ctx), C.uintptr_t(info))
		}
		cgPDFContextEndPage = func(ctx uintptr) { C.wtPDFContextEndPage(C.uintptr_t(ctx)) }

		cfRelease = func(o uintptr) { C.wtCFRelease(C.uintptr_t(o)) }
		cfStringCreateWithBytes = func(alloc uintptr, b *byte, n int, enc uint32, external bool) uintptr {
			return uintptr(C.wtCFStringCreateWithBytes(C.uintptr_t(alloc), unsafe.Pointer(b), C.int(n), C.uint32_t(enc), wtBool(external)))
		}
		cfDictionaryCreate = func(alloc uintptr, keys, vals *uintptr, n int, kcb, vcb uintptr) uintptr {
			return uintptr(C.wtCFDictionaryCreate(C.uintptr_t(alloc), (*C.uintptr_t)(unsafe.Pointer(keys)), (*C.uintptr_t)(unsafe.Pointer(vals)), C.int(n), C.uintptr_t(kcb), C.uintptr_t(vcb)))
		}
		cfAttrStringCreate = func(alloc uintptr, s uintptr, attrs uintptr) uintptr {
			return uintptr(C.wtCFAttributedStringCreate(C.uintptr_t(alloc), C.uintptr_t(s), C.uintptr_t(attrs)))
		}
		ctFontCreateWithName = func(name uintptr, size float64, m uintptr) uintptr {
			return uintptr(C.wtCTFontCreateWithName(C.uintptr_t(name), C.double(size), C.uintptr_t(m)))
		}
		ctLineCreateWithAttrStr = func(as uintptr) uintptr {
			return uintptr(C.wtCTLineCreateWithAttributedString(C.uintptr_t(as)))
		}
		ctLineDraw = func(line uintptr, ctx uintptr) { C.wtCTLineDraw(C.uintptr_t(line), C.uintptr_t(ctx)) }
		ctLineTypographicBounds = func(line uintptr, asc, desc, lead *float64) float64 {
			return float64(C.wtCTLineGetTypographicBounds(C.uintptr_t(line), (*C.double)(unsafe.Pointer(asc)), (*C.double)(unsafe.Pointer(desc)), (*C.double)(unsafe.Pointer(lead))))
		}

		cfRetain = func(o uintptr) uintptr { return uintptr(C.wtCFRetain(C.uintptr_t(o))) }
		cfArrayCount = func(a uintptr) int { return int(C.wtCFArrayGetCount(C.uintptr_t(a))) }
		cfArrayAt = func(a uintptr, i int) uintptr {
			return uintptr(C.wtCFArrayGetValueAtIndex(C.uintptr_t(a), C.int(i)))
		}
		cfDictGet = func(d, k uintptr) uintptr { return uintptr(C.wtCFDictionaryGetValue(C.uintptr_t(d), C.uintptr_t(k))) }
		cfNumberGet = func(n uintptr, typ int, p unsafe.Pointer) bool {
			return C.wtCFNumberGetValue(C.uintptr_t(n), C.int(typ), p) != 0
		}
		cfNumberCreate = func(alloc uintptr, typ int, p unsafe.Pointer) uintptr {
			return uintptr(C.wtCFNumberCreate(C.uintptr_t(alloc), C.int(typ), p))
		}
		ctDescCreate = func(attrs uintptr) uintptr {
			return uintptr(C.wtCTFontDescriptorCreateWithAttributes(C.uintptr_t(attrs)))
		}
		ctDescMatching = func(d, mandatory uintptr) uintptr {
			return uintptr(C.wtCTFontDescriptorCreateMatchingFontDescriptors(C.uintptr_t(d), C.uintptr_t(mandatory)))
		}
		ctDescCopyAttr = func(d, attr uintptr) uintptr {
			return uintptr(C.wtCTFontDescriptorCopyAttribute(C.uintptr_t(d), C.uintptr_t(attr)))
		}
		ctFontFromDesc = func(d uintptr, size float64, m uintptr) uintptr {
			return uintptr(C.wtCTFontCreateWithFontDescriptor(C.uintptr_t(d), C.double(size), C.uintptr_t(m)))
		}
		ctUIFont = func(typ int32, size float64, lang uintptr) uintptr {
			return uintptr(C.wtCTFontCreateUIFontForLanguage(C.int32_t(typ), C.double(size), C.uintptr_t(lang)))
		}

		// the constants come from the linked frameworks directly
		kPDFMediaBox = uintptr(C.wtPDFMediaBox())
		kPDFCreator = uintptr(C.wtPDFCreator())
		kFontAttr = uintptr(C.wtFontAttr())
		kFgFromCtx = uintptr(C.wtFgFromCtx())
		kTrue = uintptr(C.wtTrue())
		kFamilyAttr = uintptr(C.wtFamilyAttr())
		kTraitsAttr = uintptr(C.wtTraitsAttr())
		kWeightTrait = uintptr(C.wtWeightTrait())
		kSymbolicTrait = uintptr(C.wtSymbolicTrait())
		kKernAttr = uintptr(C.wtKernAttr())
		kWidthTrait = uintptr(C.wtWidthTrait())
		kKeyCB = uintptr(C.wtKeyCB())
		kValCB = uintptr(C.wtValCB())
	})
}
