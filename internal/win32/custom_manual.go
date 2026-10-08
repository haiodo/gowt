//go:build windows

// os_custom.c natives that are not DLL exports.
package win32

import (
	"math"
	"syscall"
	"unsafe"
)

// OS.GID_ROTATE_ANGLE_FROM_ARGUMENT: the macro maps the 16-bit argument to radians.
func custom_GID_ROTATE_ANGLE_FROM_ARGUMENT(arg int64) float64 {
	return float64(arg)/65535.0*4.0*math.Pi - 2.0*math.Pi
}

var procSendMessageW = syscall.NewLazyDLL("user32.dll").NewProc("SendMessageW")

// OS.TreeView_GetItemRect: the TreeView_GetItemRect macro (TVM_GETITEMRECT with the item handle in the RECT).
func custom_TreeView_GetItemRect(hwndTV int64, hitem int64, prc *RECT, fItemRect bool) bool {
	var buf [2]uint64
	buf[0] = uint64(hitem)
	r, _, _ := procSendMessageW.Call(uintptr(hwndTV), 0x1104, boolToUintptr(fItemRect), uintptr(unsafe.Pointer(&buf)))
	if r == 0 {
		return false
	}
	prc.fromC(unsafe.Pointer(&buf))
	return true
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func custom_GetLibraryHandle() int64 {
	h, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	return int64(h)
}

// Dark mode uses undocumented uxtheme ordinals (133 AllowDarkModeForWindow, 135 SetPreferredAppMode, 1903+ only);
// a missing export reports it unavailable.
var (
	uxthemeOrdinals    = syscall.NewLazyDLL("uxtheme.dll")
	procGetProcAddress = kernel32.NewProc("GetProcAddress")
	darkModeProcs      struct{ allow, setMode uintptr }
)

func uxthemeOrdinal(n uintptr) uintptr {
	if uxthemeOrdinals.Load() != nil {
		return 0
	}
	r, _, _ := procGetProcAddress.Call(uxthemeOrdinals.Handle(), n)
	return r
}

func custom_IsDarkModeAvailable() bool {
	if darkModeProcs.setMode == 0 && OsVersionWIN32_BUILD >= 18362 {
		darkModeProcs.allow = uxthemeOrdinal(133)
		darkModeProcs.setMode = uxthemeOrdinal(135)
	}
	return darkModeProcs.allow != 0 && darkModeProcs.setMode != 0
}

func custom_AllowDarkModeForWindow(hWnd int64, allow bool) bool {
	if !custom_IsDarkModeAvailable() {
		return false
	}
	r, _, _ := syscall.SyscallN(darkModeProcs.allow, uintptr(hWnd), boolToUintptr(allow))
	return r&0xff != 0
}

func custom_SetPreferredAppMode(mode int32) int32 {
	if !custom_IsDarkModeAvailable() {
		return 0
	}
	r, _, _ := syscall.SyscallN(darkModeProcs.setMode, uintptr(mode))
	return int32(r)
}

func OsVersionCheckCompatibleWindowsVersion() {}

// OS.setTheme(boolean) took a Display and Colors; the dark-theme tweaks are not ported.
func OSSetTheme(isDarkTheme bool) {}
