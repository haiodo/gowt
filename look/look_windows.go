package look

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/internal/win32"
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

func windowsBuild() uint32 { return uint32(win32.OsVersionWIN32_BUILD) }

func dwmSet(h int64, attr, value int32) {
	procDwmSetWindowAttribute.Call(uintptr(h), uintptr(attr), uintptr(unsafe.Pointer(&value)), 4)
}

// backdropOnKey marks a shell whose background was set for the material.
const backdropOnKey = "gowt.look.backdropOn"

// setBackdrop needs Windows 11 22H2 (build 22621); older builds keep the opaque window.
// Translucent is Acrylic, Glass is Mica. The material shows only where the client area is painted black
// (DWM treats RGB 0,0,0 over an extended frame as transparent), so the shell background becomes black:
// widgets with their own opaque background are unaffected, but text drawn in pure black turns transparent -
// use a near-black foreground on the material.
func setBackdrop(s *swt.Shell, b Backdrop) {
	if windowsBuild() < buildWin11_22H2 {
		return
	}
	h := s.Handle
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
		if on, _ := s.GetDataKey(backdropOnKey).(bool); on {
			s.SetDataKeyValue(backdropOnKey, false)
			s.SetBackgroundWithColor(nil)
			s.SetBackgroundMode(swt.INHERIT_NONE)
		}
		return
	}
	s.SetDataKeyValue(backdropOnKey, true)
	s.SetBackgroundWithColor(swt.NewColorRedGreenBlue(0, 0, 0))
	s.SetBackgroundMode(swt.INHERIT_DEFAULT)
}

func setRoundedCorners(s *swt.Shell, on bool) {
	if windowsBuild() < buildWin11 {
		return
	}
	pref := int32(dwmcpDoNotRound)
	if on {
		pref = dwmcpRound
	}
	dwmSet(s.Handle, dwmaWindowCornerPreference, pref)
}

func setDarkContent(d *swt.Display, on bool) {
	d.SetData("org.eclipse.swt.internal.win32.useDarkModeExplorerTheme", on)
}

func classic() {}

func setFullSizeContent(*swt.Shell, bool) {}

func setGlass(*swt.Composite, bool) {}

func glassButton(*swt.Control) {}
