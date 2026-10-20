//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

var (
	cgEventCreateScrollWheelEvent2 func(source uintptr, units uint32, count uint32, w1, w2, w3 int32) uintptr
	cfRelease                      func(uintptr)
)

const cgScrollEventUnitLine = 1

// SendScrollWheel hands the NSScrollView around view a wheel event of lines (negative: down), for
// tests: an event posted through the window server needs the Accessibility permission.
func SendScrollWheel(view *NSView, lines int32) bool {
	ensureFrameworks()
	if cfRelease == nil {
		if _, err := purego.Dlsym(purego.RTLD_DEFAULT, "CGEventCreateScrollWheelEvent2"); err != nil {
			return false
		}
		purego.RegisterLibFunc(&cgEventCreateScrollWheelEvent2, purego.RTLD_DEFAULT, "CGEventCreateScrollWheelEvent2")
		purego.RegisterLibFunc(&cfRelease, purego.RTLD_DEFAULT, "CFRelease")
	}
	cg := cgEventCreateScrollWheelEvent2(0, cgScrollEventUnitLine, 1, lines, 0, 0)
	if cg == 0 {
		return false
	}
	defer cfRelease(cg)
	target := msg(uintptr(view.Id), "enclosingScrollView")
	if target == 0 {
		target = uintptr(view.Id)
	}
	msg(target, "scrollWheel:", msg(class("NSEvent"), "eventWithCGEvent:", cg))
	return true
}
