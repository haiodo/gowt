//go:build windows

// Hand-written UI Automation access for properties MSAA does not expose reliably on every control.
package win32

import "unsafe"

var (
	clsidCUIAutomation = COMIIDFromStringLpsz("{FF48DBA4-60EF-4201-AA87-54103EEF594E}")
	iidIUIAutomation   = COMIIDFromStringLpsz("{30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}")
)

const (
	uiaGetFocusedElement        = 8
	uiaGetCurrentPropertyValue  = 10
	uiaHasKeyboardFocusProperty = 30008
)

// UIAHasKeyboardFocus reports the HasKeyboardFocus property of the element UIA reports as focused.
func UIAHasKeyboardFocus() (focused bool, ok bool) {
	var automation [1]int64
	if COMCoCreateInstance(clsidCUIAutomation, 0, COMCLSCTX_INPROC_SERVER, iidIUIAutomation, automation[:]) != COMS_OK {
		return false, false
	}
	defer COMVtblCall(2, automation[0])
	var element int64
	if COMVtblCallOverload2(uiaGetFocusedElement, automation[0], int64(uintptr(unsafe.Pointer(&element)))) != COMS_OK || element == 0 {
		return false, false
	}
	defer COMVtblCall(2, element)
	var value [2]uint64
	if COMVtblCallOverload14(uiaGetCurrentPropertyValue, element, uiaHasKeyboardFocusProperty, int64(uintptr(unsafe.Pointer(&value[0])))) != COMS_OK {
		return false, false
	}
	if *(*int16)(unsafe.Pointer(&value[0])) != COMVT_BOOL {
		return false, false
	}
	return *(*int16)(unsafe.Add(unsafe.Pointer(&value[0]), 8)) != 0, true
}
