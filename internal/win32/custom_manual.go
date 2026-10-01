//go:build windows

// os_custom.c natives that are not DLL exports, and the manifest SWT loads from its JNI DLL's resources.
package win32

import (
	_ "embed"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// SWT loads this manifest (visual styles, common controls 6) from its JNI DLL's resources; a Go exe has none,
// so it is written next to the temp files and activated for the process before comctl32 loads.
//
//go:embed swt.manifest
var manifest []byte

func init() {
	path := filepath.Join(os.TempDir(), "gowt-swt.manifest")
	if os.WriteFile(path, manifest, 0o644) != nil {
		return
	}
	src, _ := syscall.UTF16PtrFromString(path)
	ctx := struct {
		cbSize                 uint32
		dwFlags                uint32
		lpSource               *uint16
		wProcessorArchitecture uint16
		wLangId                uint16
		lpAssemblyDirectory    uintptr
		lpResourceName         uintptr
		lpApplicationName      uintptr
		hModule                uintptr
	}{lpSource: src}
	ctx.cbSize = uint32(unsafe.Sizeof(ctx))
	h, _, _ := kernel32.NewProc("CreateActCtxW").Call(uintptr(unsafe.Pointer(&ctx)))
	if h == ^uintptr(0) {
		return
	}
	var cookie uintptr
	kernel32.NewProc("ActivateActCtx").Call(h, uintptr(unsafe.Pointer(&cookie)))
}

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
