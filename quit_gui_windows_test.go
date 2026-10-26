package gowt

import "github.com/haiodo/gowt/internal/win32"

const (
	systemQuitAvailable = true
	dialogDismissable   = true
)

// systemQuit sends WM_QUERYENDSESSION to every window of this thread, as the system does on logout
// and shutdown; end also sends WM_ENDSESSION. It reports whether no window said no.
func systemQuit(w *Window, end bool) bool {
	tid := win32.OSGetWindowThreadProcessId(w.shell.Handle, nil)
	var wins []int64
	for h := win32.OSGetWindow(w.shell.Handle, win32.OSGW_HWNDFIRST); h != 0; h = win32.OSGetWindow(h, win32.OSGW_HWNDNEXT) {
		if win32.OSGetWindowThreadProcessId(h, nil) == tid {
			wins = append(wins, h)
		}
	}
	ok := true
	for _, h := range wins {
		if win32.OSSendMessageOverload4(h, win32.OSWM_QUERYENDSESSION, 0, 0) == 0 {
			ok = false
		}
	}
	if ok && end {
		for _, h := range wins {
			win32.OSSendMessageOverload4(h, win32.OSWM_ENDSESSION, 1, 0)
		}
	}
	return ok
}

// dismissDialog presses No in the message box that the window owns.
func dismissDialog(w *Window) {
	if p := win32.OSGetLastActivePopup(w.shell.Handle); p != 0 && p != w.shell.Handle {
		win32.OSPostMessage(p, win32.OSWM_COMMAND, int64(win32.OSIDNO), 0)
	}
}
