package gowt

import "github.com/haiodo/gowt/internal/cocoa"

const (
	systemQuitAvailable = true
	dialogDismissable   = false
)

// systemQuit is NSApp terminate:, the path of Cmd+Q, the Dock and logout. A yes answer would end
// the test process, so only a refusal is tried.
func systemQuit(_ *Window, allowed bool) bool {
	if allowed {
		return true
	}
	cocoa.NSApplicationSharedApplication().Terminate(nil)
	return false
}

func dismissDialog(*Window) {}
