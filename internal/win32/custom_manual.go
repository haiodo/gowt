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

// Dark mode needs undocumented uxtheme ordinals: reported unavailable.
func custom_IsDarkModeAvailable() bool { return false }

func custom_AllowDarkModeForWindow(hWnd int64, allow bool) bool { return false }

func custom_SetPreferredAppMode(mode int32) int32 { return 0 }

func OsVersionCheckCompatibleWindowsVersion() {}

// OS.setTheme(boolean) took a Display and Colors; the dark-theme tweaks are not ported.
func OSSetTheme(isDarkTheme bool) {}
