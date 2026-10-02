//go:build windows

package win32

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
)

// With GOWT_TRACE_DISPATCH=1 every message about to be dispatched goes into a ring of the last 16;
// DispatchTrace prints it (the snapshot watchdog does when a step is stuck).
var (
	traceOn   = os.Getenv("GOWT_TRACE_DISPATCH") != ""
	traceMu   sync.Mutex
	traceRing [16]string
	traceN    int
)

var msgNames = map[int32]string{0x1: "WM_CREATE", 0x2: "WM_DESTROY", 0x3: "WM_MOVE", 0x5: "WM_SIZE", 0x6: "WM_ACTIVATE", 0x7: "WM_SETFOCUS", 0x8: "WM_KILLFOCUS",
	0xF: "WM_PAINT", 0x10: "WM_CLOSE", 0x14: "WM_ERASEBKGND", 0x18: "WM_SHOWWINDOW", 0x46: "WM_WINDOWPOSCHANGING", 0x47: "WM_WINDOWPOSCHANGED", 0x4E: "WM_NOTIFY",
	0x85: "WM_NCPAINT", 0x100: "WM_KEYDOWN", 0x111: "WM_COMMAND", 0x113: "WM_TIMER", 0x200: "WM_MOUSEMOVE", 0x201: "WM_LBUTTONDOWN", 0x202: "WM_LBUTTONUP",
	0x2A1: "WM_MOUSEHOVER", 0x2A3: "WM_MOUSELEAVE", 0x31F: "WM_DWMCOMPOSITIONCHANGED"}

func traceDispatch(m *MSG) {
	if !traceOn || m == nil {
		return
	}
	name, ok := msgNames[m.Message]
	if !ok {
		name = fmt.Sprintf("0x%X", m.Message)
	}
	var cls [64]uint16
	n := OSGetClassName(m.Hwnd, cls[:], 64)
	line := fmt.Sprintf("%s hwnd=%#x class=%q wParam=%#x lParam=%#x", name, m.Hwnd, syscall.UTF16ToString(cls[:n]), m.WParam, m.LParam)
	traceMu.Lock()
	traceRing[traceN%len(traceRing)] = line
	traceN++
	traceMu.Unlock()
}

// DispatchTrace is the recorded messages, oldest first (empty without GOWT_TRACE_DISPATCH).
func DispatchTrace() string {
	traceMu.Lock()
	defer traceMu.Unlock()
	var b strings.Builder
	for i := max(0, traceN-len(traceRing)); i < traceN; i++ {
		b.WriteString(traceRing[i%len(traceRing)] + "\n")
	}
	return b.String()
}
