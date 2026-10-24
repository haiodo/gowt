//go:build darwin

package cg

import (
	"testing"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/internal/cocoa"
)

// A scroll event built from a CGEvent: no window, no app.
func TestReadInputScroll(t *testing.T) {
	initInput()
	var create func(src uintptr, units uint32, n uint32, w1, w2, w3 int32) uintptr
	var setInt func(ev uintptr, field uint32, v int64)
	var fromCG func(recv, sel, cg uintptr) uintptr
	purego.RegisterLibFunc(&create, purego.RTLD_DEFAULT, "CGEventCreateScrollWheelEvent2")
	purego.RegisterLibFunc(&setInt, purego.RTLD_DEFAULT, "CGEventSetIntegerValueField")
	purego.RegisterLibFunc(&fromCG, purego.RTLD_DEFAULT, "objc_msgSend")

	cg := create(0, 1, 2, 7, -3, 0) // pixel units
	if cg == 0 {
		t.Skip("no CGEvent")
	}
	setInt(cg, 88, 1)  // kCGScrollWheelEventIsContinuous
	setInt(cg, 99, 2)  // kCGScrollWheelEventScrollPhase: changed (CG numbering)
	setInt(cg, 123, 1) // kCGScrollWheelEventMomentumPhase: begin
	ev := fromCG(clsNSEvent, uintptr(cocoa.OSSel_registerName("eventWithCGEvent:")), cg)
	if ev == 0 {
		t.Skip("no NSEvent")
	}
	e, _, ok := readInput(ev)
	if !ok || e.Kind != InputScroll {
		t.Fatalf("not a scroll: %+v ok=%v", e, ok)
	}
	if !e.Precise || e.DY <= 0 || e.DX >= 0 {
		t.Errorf("deltas %+v", e)
	}
	if e.Phase != 4 || e.Momentum != 1 {
		t.Errorf("phase %d momentum %d, want NSEventPhase changed 4 and began 1", e.Phase, e.Momentum)
	}
}

// The Objective-C runtime must accept the blocks, and stopping twice is harmless.
func TestMonitorInstall(t *testing.T) {
	initInput()
	stop := MonitorInput(0, func(InputEvent) bool { return false })
	stop()
	stop()
	stop = OnWindowChange(0, func() {})
	stop()
}

func TestMouseUpAndSmartMagnify(t *testing.T) {
	initInput()
	var create func(src uintptr, typ uint32, p cgPoint, button uint32) uintptr
	var fromCG func(recv, sel, cg uintptr) uintptr
	purego.RegisterLibFunc(&create, purego.RTLD_DEFAULT, "CGEventCreateMouseEvent")
	purego.RegisterLibFunc(&fromCG, purego.RTLD_DEFAULT, "objc_msgSend")
	cases := []struct {
		typ, cgButton uint32
		want          int
	}{{2, 0, 1}, {4, 1, 3}} // left up, right up
	for _, c := range cases {
		cg := create(0, c.typ, cgPoint{10, 20}, c.cgButton)
		ev := fromCG(clsNSEvent, uintptr(cocoa.OSSel_registerName("eventWithCGEvent:")), cg)
		if ev == 0 {
			t.Skip("no NSEvent")
		}
		e, _, ok := readInput(ev)
		if !ok || e.Kind != InputMouseUp || e.Button != c.want {
			t.Errorf("type %d: %+v ok=%v, want button %d", c.typ, e, ok, c.want)
		}
	}

	view := send0(send0(uintptr(cocoa.OSObjc_lookUpClass("NSView")), sels.alloc), sels.init)
	sel := uintptr(cocoa.OSSel_registerName("smartMagnifyWithEvent:"))
	enableSmartMagnify(view)
	if !nsBool(msgInt(view, isel.respondsTo, sel)) {
		t.Error("the view must respond to smartMagnifyWithEvent: after enableSmartMagnify")
	}
}
