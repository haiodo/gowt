package gowt

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

const (
	dwmaWindowCornerPreference         = 33
	dwmaSystemBackdropType             = 38
	dwmbbNone, dwmbbMica, dwmbbAcrylic = 1, 2, 3
	dwmcpDoNotRound, dwmcpRound        = 1, 2

	buildWin11      = 22000
	buildWin11_22H2 = 22621
)

var (
	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrame        = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procRtlGetVersion         = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
)

func windowsBuild() uint32 {
	var v [36]uint32 // OSVERSIONINFOW: size, major, minor, build, ...
	v[0] = 284
	if r, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&v[0]))); r != 0 {
		return 0
	}
	return v[3]
}

func dwmSet(h int64, attr, value int32) {
	procDwmSetWindowAttribute.Call(uintptr(h), uintptr(attr), uintptr(unsafe.Pointer(&value)), 4)
}

// setBackdrop needs Windows 11 22H2 (build 22621); older builds keep the opaque window.
// Translucent is Acrylic, Glass is Mica. The material shows only where the client area is painted black
// (DWM treats RGB 0,0,0 over an extended frame as transparent), so the shell background becomes black:
// widgets with their own opaque background are unaffected, but text drawn in pure black turns transparent -
// use a near-black foreground on the material.
func (w *Window) setBackdrop(b Backdrop) {
	if windowsBuild() < buildWin11_22H2 {
		return
	}
	h := w.shell.Handle
	kind, margin := int32(dwmbbNone), int32(0)
	switch b {
	case BackdropTranslucent:
		kind, margin = dwmbbAcrylic, -1
	case BackdropGlass:
		kind, margin = dwmbbMica, -1
	}
	dwmSet(h, dwmaSystemBackdropType, kind)
	m := [4]int32{margin, margin, margin, margin}
	procDwmExtendFrame.Call(uintptr(h), uintptr(unsafe.Pointer(&m)))
	if margin == 0 {
		w.shell.SetBackgroundWithColor(nil)
		w.shell.SetBackgroundMode(swt.INHERIT_NONE)
		return
	}
	w.shell.SetBackgroundWithColor(RGB{}.color())
	w.shell.SetBackgroundMode(swt.INHERIT_DEFAULT)
}

func (w *Window) setRoundedCorners(on bool) {
	if windowsBuild() < buildWin11 {
		return
	}
	pref := int32(dwmcpDoNotRound)
	if on {
		pref = dwmcpRound
	}
	dwmSet(w.shell.Handle, dwmaWindowCornerPreference, pref)
}
