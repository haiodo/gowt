//go:build darwin

package view

// Temporary focus tracing for the Edit focus bug (GOWT_TRACE_FOCUS=1): AppKit's responder calls on the
// SWT window, text field, canvas and field editor classes are wrapped and logged to stderr.

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

var (
	focusTracing = os.Getenv("GOWT_TRACE_FOCUS") != ""
	ftOnce       sync.Once
	ft0          = time.Now()
)

func focusTrace(format string, a ...any) {
	if focusTracing {
		fmt.Fprintf(os.Stderr, "focus %8.1fms %s\n", float64(time.Since(ft0).Microseconds())/1000, fmt.Sprintf(format, a...))
	}
}

func ftSel(s string) uintptr { return uintptr(objc.RegisterName(s)) }

func ftSend(id uintptr, s string) uintptr { return uintptr(objc.ID(id).Send(objc.RegisterName(s))) }

func ftClassName(id uintptr) string {
	return cocoa.NewNSStringOverload1(int64(ftSend(id, "className"))).GetString()
}

// ftDesc names an object: its class and the SWT widget behind it; a field editor names its delegate.
func ftDesc(id uintptr) string {
	if id == 0 {
		return "nil"
	}
	s := ftClassName(id)
	if w := swt.DisplayGetWidget(int64(id)); w != nil {
		return fmt.Sprintf("%s[%s %p]", s, reflect.TypeOf(w.Impl()).Elem().Name(), w.Impl())
	}
	if objc.Send[uintptr](objc.ID(id), objc.RegisterName("respondsToSelector:"), objc.RegisterName("isFieldEditor"))&0xff != 0 &&
		ftSend(id, "isFieldEditor")&0xff != 0 {
		return fmt.Sprintf("%s(field editor of %s)", s, ftDesc(ftSend(id, "delegate")))
	}
	return s
}

// ftFirst describes the first responder of the window w.
func ftFirst(w uintptr) string {
	if w == 0 {
		return "no window"
	}
	return ftDesc(ftSend(w, "firstResponder"))
}

// traceFirstResponder logs the first responder of the view's window and of the key window.
func (v *View) traceFirstResponder(label string) {
	if !focusTracing {
		return
	}
	w := ftSend(v.nsview, "window")
	key := ftSend(ftSend(uintptr(objc.ID(objc.GetClass("NSApplication"))), "sharedApplication"), "keyWindow")
	focusTrace("%s: firstResponder %s; key window is the view's: %v, its firstResponder %s", label, ftFirst(w), key == w, ftFirst(key))
}

func ftReplace(cls, sel uintptr, imp int64, types string) uintptr {
	orig := cocoa.OSClass_getMethodImplementation(int64(cls), int64(sel))
	if !cocoa.OSClass_addMethod(int64(cls), int64(sel), imp, types) {
		cocoa.OSMethod_setImplementation(cocoa.OSClass_getInstanceMethod(int64(cls), int64(sel)), imp)
	}
	return uintptr(orig)
}

func ftClass(name string) uintptr { return uintptr(objc.GetClass(name)) }

// ftHookBool wraps a BOOL method without arguments.
func ftHookBool(cls uintptr, name string) {
	if cls == 0 {
		return
	}
	var orig func(self, sel uintptr) uintptr
	imp := cocoa.NewCallbackN(2, func(a []int64) int64 {
		focusTrace("%s %s begin", name, ftDesc(uintptr(a[0])))
		r := orig(uintptr(a[0]), uintptr(a[1])) & 0xff
		focusTrace("%s %s -> %v", name, ftDesc(uintptr(a[0])), r != 0)
		return int64(r)
	})
	purego.RegisterFunc(&orig, ftReplace(cls, ftSel(name), imp, "c@:"))
}

// ftHookMakeFirst wraps makeFirstResponder: of a window class.
func ftHookMakeFirst(cls uintptr) {
	if cls == 0 {
		return
	}
	var orig func(self, sel, r uintptr) uintptr
	imp := cocoa.NewCallbackN(3, func(a []int64) int64 {
		w := uintptr(a[0])
		focusTrace("makeFirstResponder(%s) from %s", ftDesc(uintptr(a[2])), ftFirst(w))
		r := orig(w, uintptr(a[1]), uintptr(a[2])) & 0xff
		focusTrace("makeFirstResponder(%s) -> %v, firstResponder %s", ftDesc(uintptr(a[2])), r != 0, ftFirst(w))
		return int64(r)
	})
	purego.RegisterFunc(&orig, ftReplace(cls, ftSel("makeFirstResponder:"), imp, "c@:@"))
}

// ftHookVoid1 wraps a void method with one object argument.
func ftHookVoid1(cls uintptr, name string) {
	if cls == 0 {
		return
	}
	var orig func(self, sel, x uintptr)
	imp := cocoa.NewCallbackN(3, func(a []int64) int64 {
		focusTrace("%s %s begin", name, ftDesc(uintptr(a[0])))
		orig(uintptr(a[0]), uintptr(a[1]), uintptr(a[2]))
		focusTrace("%s %s end", name, ftDesc(uintptr(a[0])))
		return 0
	})
	purego.RegisterFunc(&orig, ftReplace(cls, ftSel(name), imp, "v@:@"))
}

// ftHookSendEvent wraps sendEvent: of the application or a window class and logs key events.
func ftHookSendEvent(cls uintptr, who string) {
	if cls == 0 {
		return
	}
	var orig func(self, sel, ev uintptr)
	imp := cocoa.NewCallbackN(3, func(a []int64) int64 {
		ev := uintptr(a[2])
		t := ftSend(ev, "type")
		if t != 10 && t != 11 {
			orig(uintptr(a[0]), uintptr(a[1]), ev)
			return 0
		}
		kind := map[uintptr]string{10: "keyDown", 11: "keyUp"}[t]
		chars := cocoa.NewNSStringOverload1(int64(ftSend(ev, "characters"))).GetString()
		w := ftSend(ev, "window")
		focusTrace("%s sendEvent %s %q begin, firstResponder %s", who, kind, chars, ftFirst(w))
		orig(uintptr(a[0]), uintptr(a[1]), ev)
		focusTrace("%s sendEvent %s %q end, firstResponder %s", who, kind, chars, ftFirst(w))
		return 0
	})
	purego.RegisterFunc(&orig, ftReplace(cls, ftSel("sendEvent:"), imp, "v@:@"))
}

func installFocusTrace() {
	if !focusTracing {
		return
	}
	ftOnce.Do(func() {
		app := ftSend(uintptr(objc.ID(objc.GetClass("NSApplication"))), "sharedApplication")
		ftHookSendEvent(uintptr(cocoa.OSObject_getClass(int64(app))), "app")
		for _, c := range []string{"SWTWindow", "SWTPanel"} {
			ftHookMakeFirst(ftClass(c))
			ftHookSendEvent(ftClass(c), "window")
		}
		for _, c := range []string{"SWTTextField", "SWTCanvasView", "SWTEditorView"} {
			ftHookBool(ftClass(c), "becomeFirstResponder")
			ftHookBool(ftClass(c), "resignFirstResponder")
		}
		ftHookVoid1(ftClass("SWTTextField"), "textDidEndEditing:")
		ftHookBool(ftClass("SWTTextField"), "abortEditing")
		focusTrace("tracing installed")
	})
}
