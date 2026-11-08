//go:build darwin && cgo && gowtcgo

package cg

/*
#cgo LDFLAGS: -framework QuartzCore
#include <QuartzCore/QuartzCore.h>
#include <objc/message.h>
#include <objc/runtime.h>
#include <stdlib.h>
#include <stdint.h>
#include <dlfcn.h>

typedef struct { double x, y, w, h; } wtRect;
typedef struct { double x, y; } wtPoint;
typedef struct { double a, b, c, d, tx, ty; } wtAffine;

// objc_msgSend must be cast to the exact argument and return types of the method; the wt* structs
// cross the Go/C boundary by pointer because structs with float fields may not travel by value.
static void wtSendRect(uintptr_t recv, uintptr_t sel, const wtRect *r) {
	((void (*)(id, SEL, CGRect))objc_msgSend)((id)recv, (SEL)sel, (CGRect){{r->x, r->y}, {r->w, r->h}});
}
static void wtSendPoint(uintptr_t recv, uintptr_t sel, const wtPoint *p) {
	((void (*)(id, SEL, CGPoint))objc_msgSend)((id)recv, (SEL)sel, (CGPoint){p->x, p->y});
}
static void wtSendF64(uintptr_t recv, uintptr_t sel, double f) {
	((void (*)(id, SEL, double))objc_msgSend)((id)recv, (SEL)sel, f);
}
static void wtSendF32(uintptr_t recv, uintptr_t sel, float f) {
	((void (*)(id, SEL, float))objc_msgSend)((id)recv, (SEL)sel, f);
}
static void wtSendAffine(uintptr_t recv, uintptr_t sel, const wtAffine *m) {
	((void (*)(id, SEL, CGAffineTransform))objc_msgSend)((id)recv, (SEL)sel,
		(CGAffineTransform){m->a, m->b, m->c, m->d, m->tx, m->ty});
}
static uintptr_t wtSendPtr(uintptr_t recv, uintptr_t sel, uintptr_t a) {
	return (uintptr_t)((id (*)(id, SEL, id))objc_msgSend)((id)recv, (SEL)sel, (id)a);
}
static void wtSendRectRet(uintptr_t recv, uintptr_t sel, wtRect *out) {
	CGRect r = ((CGRect (*)(id, SEL))objc_msgSend)((id)recv, (SEL)sel);
	out->x = r.origin.x; out->y = r.origin.y; out->w = r.size.width; out->h = r.size.height;
}
static void wtSendAffineRet(uintptr_t recv, uintptr_t sel, wtAffine *out) {
	CGAffineTransform m = ((CGAffineTransform (*)(id, SEL))objc_msgSend)((id)recv, (SEL)sel);
	out->a = m.a; out->b = m.b; out->c = m.c; out->d = m.d; out->tx = m.tx; out->ty = m.ty;
}
static float wtSendF32Ret(uintptr_t recv, uintptr_t sel) {
	return ((float (*)(id, SEL))objc_msgSend)((id)recv, (SEL)sel);
}
static double wtSendF64Ret(uintptr_t recv, uintptr_t sel) {
	return ((double (*)(id, SEL))objc_msgSend)((id)recv, (SEL)sel);
}
// The SDK does not declare this private filter constant, so resolve it like the purego backend.
static uintptr_t wtFilterTrilinear(void) {
	void *sym = dlsym(RTLD_DEFAULT, "kCAFilterTrilinear");
	return sym ? (uintptr_t)*(void **)sym : 0;
}
*/
import "C"

import "unsafe"

func hostSel(name string) uintptr {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return uintptr(unsafe.Pointer(C.sel_registerName(cs)))
}

func hostClass(name string) uintptr {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return uintptr(unsafe.Pointer(C.objc_lookUpClass(cs)))
}

func initHost() {
	hostOnce.Do(func() {
		bind()
		s := hostSel
		sels.alloc, sels.init, sels.release = s("alloc"), s("init"), s("release")
		sels.addSublayer, sels.removeFromSuperlayer = s("addSublayer:"), s("removeFromSuperlayer")
		sels.setFrame, sels.setBounds, sels.setPosition, sels.setAnchorPoint = s("setFrame:"), s("setBounds:"), s("setPosition:"), s("setAnchorPoint:")
		sels.setAffineTransform, sels.setOpacity, sels.setHidden = s("setAffineTransform:"), s("setOpacity:"), s("setHidden:")
		sels.setContents, sels.setContentsScale = s("setContents:"), s("setContentsScale:")
		sels.begin, sels.commit, sels.setDisableActions = s("begin"), s("commit"), s("setDisableActions:")
		sels.frame, sels.affineTransform, sels.opacity, sels.contents, sels.isHidden = s("frame"), s("affineTransform"), s("opacity"), s("contents"), s("isHidden")
		sels.setActions = s("setActions:")
		sels.setMinFilter, sels.minFilter = s("setMinificationFilter:"), s("minificationFilter")
		filterTrilinear = uintptr(C.wtFilterTrilinear())
		if filterTrilinear == 0 {
			panic("gfx/cg: dlsym kCAFilterTrilinear")
		}
		clsLayer = hostClass("CALayer")
		clsTransaction = hostClass("CATransaction")

		msgRect = func(recv, sel uintptr, r cgRect) {
			C.wtSendRect(C.uintptr_t(recv), C.uintptr_t(sel), wtRectP(&r))
		}
		msgPoint = func(recv, sel uintptr, p cgPoint) {
			C.wtSendPoint(C.uintptr_t(recv), C.uintptr_t(sel), wtPointP(&p))
		}
		msgF64 = func(recv, sel uintptr, f float64) { C.wtSendF64(C.uintptr_t(recv), C.uintptr_t(sel), C.double(f)) }
		msgF32 = func(recv, sel uintptr, f float32) { C.wtSendF32(C.uintptr_t(recv), C.uintptr_t(sel), C.float(f)) }
		msgAffine = func(recv, sel uintptr, m cgAffine) {
			C.wtSendAffine(C.uintptr_t(recv), C.uintptr_t(sel), wtAffineP(&m))
		}
		msgInt = func(recv, sel, a uintptr) uintptr {
			return uintptr(C.wtSendPtr(C.uintptr_t(recv), C.uintptr_t(sel), C.uintptr_t(a)))
		}
		msgRectRet = func(recv, sel uintptr) cgRect {
			var r C.wtRect
			C.wtSendRectRet(C.uintptr_t(recv), C.uintptr_t(sel), &r)
			return *(*cgRect)(unsafe.Pointer(&r))
		}
		msgAffRet = func(recv, sel uintptr) cgAffine {
			var m C.wtAffine
			C.wtSendAffineRet(C.uintptr_t(recv), C.uintptr_t(sel), &m)
			return *(*cgAffine)(unsafe.Pointer(&m))
		}
		msgF32Ret = func(recv, sel uintptr) float32 {
			return float32(C.wtSendF32Ret(C.uintptr_t(recv), C.uintptr_t(sel)))
		}
		msgF64Ret = func(recv, sel uintptr) float64 {
			return float64(C.wtSendF64Ret(C.uintptr_t(recv), C.uintptr_t(sel)))
		}
	})
}
