//go:build windows

// Hand-written: reach the MSAA IAccessible of a window, so tests can query the same object a
// screen reader gets through WM_GETOBJECT (oleacc.dll).
package win32

import (
	"syscall"
	"unsafe"
)

var proc_AccessibleObjectFromWindow = newProc("AccessibleObjectFromWindow")

// AccessibleObjectFromWindow returns the IAccessible of a window object (OBJID_CLIENT for the
// client area) and the native HRESULT.
func AccessibleObjectFromWindow(hwnd int64, idObject int32) (*IAccessible, int32) {
	var ppvObject [1]int64
	var iid [2]uint64
	COMIIDIAccessible.toC(unsafe.Pointer(&iid[0]))
	r, _, _ := syscall.SyscallN(proc_AccessibleObjectFromWindow.addr(), uintptr(hwnd),
		uintptr(idObject), uintptr(unsafe.Pointer(&iid[0])), uintptr(unsafe.Pointer(&ppvObject[0])))
	hr := int32(r)
	if hr != COMS_OK || ppvObject[0] == 0 {
		return nil, hr
	}
	return NewIAccessible(ppvObject[0]), hr
}

// AccBSTR reads and frees a BSTR returned through an MSAA out-parameter (Get_accName and friends).
func AccBSTR(bstr int64) string {
	if bstr == 0 {
		return ""
	}
	defer COMSysFreeString(bstr)
	n := int(COMSysStringByteLen(bstr))
	if n <= 0 {
		return ""
	}
	buf := make([]uint16, (n+1)/2)
	OSMoveMemoryOverload2(buf, bstr, int32(n))
	return syscall.UTF16ToString(buf)
}
