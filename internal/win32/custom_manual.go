//go:build windows

// os_custom.c natives that are not DLL exports, and the manifest SWT loads from its JNI DLL's resources.
package win32

import "syscall"

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
