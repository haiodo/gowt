//go:build windows

// Package fluent gives native Win32 controls the Windows 11 (WinUI 3) look: text boxes, flat borders,
// subtle and standard buttons, a rounded input surface and the Explorer selection of tables. The
// controls stay SWT widgets; the frame is painted in their non-client area from a window subclass.
package fluent

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	pGetWindowRect        = user32.NewProc("GetWindowRect")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pClientToScreen       = user32.NewProc("ClientToScreen")
	pGetWindowDC          = user32.NewProc("GetWindowDC")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pGetWindowLongPtr     = user32.NewProc("GetWindowLongPtrW")
	pSetWindowLongPtr     = user32.NewProc("SetWindowLongPtrW")
	pSetWindowPos         = user32.NewProc("SetWindowPos")
	pRedrawWindow         = user32.NewProc("RedrawWindow")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pGetDpiForWindow      = user32.NewProc("GetDpiForWindow")
	pGetSystemMetricsDpi  = user32.NewProc("GetSystemMetricsForDpi")
	pSendMessage          = user32.NewProc("SendMessageW")
	pGetFocus             = user32.NewProc("GetFocus")
	pTrackMouseEvent      = user32.NewProc("TrackMouseEvent")
	pExcludeClipRect      = gdi32.NewProc("ExcludeClipRect")
	pSelectObject         = gdi32.NewProc("SelectObject")
	pGetTextMetrics       = gdi32.NewProc("GetTextMetricsW")
	pSetWindowSubclass    = comctl32.NewProc("SetWindowSubclass")
	pDefSubclassProc      = comctl32.NewProc("DefSubclassProc")
	pRemoveWindowSubclass = comctl32.NewProc("RemoveWindowSubclass")
	pRegGetValue          = advapi32.NewProc("RegGetValueW")
)

const (
	wmNCDestroy  = 0x82
	wmNCCalcSize = 0x83
	wmNCPaint    = 0x85
	wmSetFocus   = 0x7
	wmKillFocus  = 0x8
	wmEnable     = 0xA
	wmGetFont    = 0x31
	wmMouseWheel = 0x20A
	wmMouseMove  = 0x200
	wmMouseLeave = 0x2A3
	wmPaint      = 0xF
	wmEraseBkgnd = 0x14

	wsBorder  = 0x800000
	wsVScroll = 0x200000
	wsExEdge  = 0x200 // WS_EX_CLIENTEDGE

	swpNoSize       = 0x1
	swpNoMove       = 0x2
	swpNoZOrder     = 0x4
	swpNoActivate   = 0x10
	swpFrameChanged = 0x20

	rdwInvalidate = 0x1
	rdwFrame      = 0x400

	emLineScroll = 0xB6
	bmGetState   = 0xF2
	bstPushed    = 0x4
	bstFocus     = 0x8

	tmeLeave = 0x2
)

type rect struct{ left, top, right, bottom int32 }
type point struct{ x, y int32 }

func call(p *syscall.LazyProc, a ...uintptr) uintptr {
	r, _, _ := p.Call(a...)
	return r
}

// ptr turns a message parameter into a pointer without go vet's unsafe.Pointer(uintptr) report.
func ptr(p uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&p)) }

// GWL_STYLE is -16 and GWL_EXSTYLE -20.
func style(hwnd uintptr) uintptr   { return call(pGetWindowLongPtr, hwnd, ^uintptr(15)) }
func exStyle(hwnd uintptr) uintptr { return call(pGetWindowLongPtr, hwnd, ^uintptr(19)) }

func setStyle(hwnd, v uintptr)   { call(pSetWindowLongPtr, hwnd, ^uintptr(15), v) }
func setExStyle(hwnd, v uintptr) { call(pSetWindowLongPtr, hwnd, ^uintptr(19), v) }

// frameChanged makes Windows ask the control for its non-client area again.
func frameChanged(hwnd uintptr) {
	call(pSetWindowPos, hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func redrawFrame(hwnd uintptr) { call(pRedrawWindow, hwnd, 0, 0, rdwFrame|rdwInvalidate) }

func dpiOf(hwnd uintptr) int {
	if d := int(call(pGetDpiForWindow, hwnd)); d > 0 {
		return d
	}
	return 96
}

// px converts device-independent pixels (96 per inch) to device pixels at dpi.
func px(dips float64, dpi int) int32 {
	v := dips * float64(dpi) / 96
	if v < 1 && dips > 0 {
		return 1
	}
	return int32(v + 0.5)
}

func focused(hwnd uintptr) bool { return call(pGetFocus) == hwnd }

// textHeight is the height in pixels of a line in the font of the window.
func textHeight(hwnd uintptr) int32 {
	hdc := call(pGetDC, hwnd)
	if hdc == 0 {
		return 0
	}
	defer call(pReleaseDC, hwnd, hdc)
	old := call(pSelectObject, hdc, call(pSendMessage, hwnd, wmGetFont, 0, 0))
	defer call(pSelectObject, hdc, old)
	var tm [64]byte // TEXTMETRICW, tmHeight first
	if call(pGetTextMetrics, hdc, uintptr(unsafe.Pointer(&tm[0]))) == 0 {
		return 0
	}
	return *(*int32)(unsafe.Pointer(&tm[0]))
}
