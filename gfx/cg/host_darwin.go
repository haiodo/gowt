//go:build darwin && (!cgo || !gowtcgo)

package cg

import (
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/internal/cocoa"
)

func hostSel(name string) uintptr { return uintptr(cocoa.OSSel_registerName(name)) }

func initHost() {
	hostOnce.Do(func() {
		bind()
		if _, err := purego.Dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
			panic("gfx/cg: dlopen QuartzCore: " + err.Error())
		}
		for _, f := range []any{&msgRect, &msgPoint, &msgF64, &msgF32, &msgAffine, &msgInt, &msgRectRet, &msgAffRet, &msgF32Ret, &msgF64Ret} {
			purego.RegisterLibFunc(f, purego.RTLD_DEFAULT, "objc_msgSend")
		}
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
		a, err := purego.Dlsym(purego.RTLD_DEFAULT, "kCAFilterTrilinear")
		if err != nil {
			panic("gfx/cg: dlsym kCAFilterTrilinear: " + err.Error())
		}
		filterTrilinear = *(*uintptr)(*(*unsafe.Pointer)(unsafe.Pointer(&a)))
		clsLayer = uintptr(cocoa.OSObjc_lookUpClass("CALayer"))
		clsTransaction = uintptr(cocoa.OSObjc_lookUpClass("CATransaction"))
	})
}
