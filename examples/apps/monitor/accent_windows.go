package main

import (
	"syscall"
	"unsafe"
)

// systemAccent is the Windows accent colour (Settings > Personalization > Colors), stored by DWM
// as 0xAABBGGRR.
func systemAccent() (r, g, b uint8, ok bool) {
	key, err := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\DWM`)
	if err != nil {
		return 0, 0, 0, false
	}
	var h syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, key, 0, syscall.KEY_READ, &h) != nil {
		return 0, 0, 0, false
	}
	defer syscall.RegCloseKey(h)
	name, _ := syscall.UTF16PtrFromString("AccentColor")
	var typ, v, n uint32 = 0, 0, 4
	if syscall.RegQueryValueEx(h, name, nil, &typ, (*byte)(unsafe.Pointer(&v)), &n) != nil || typ != syscall.REG_DWORD {
		return 0, 0, 0, false
	}
	return uint8(v), uint8(v >> 8), uint8(v >> 16), true
}
