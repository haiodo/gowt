//go:build darwin

package swt

import "github.com/haiodo/gowt/internal/cocoa"

// kVK_Tab, the virtual key code of the Tab key.
const keyCodeTab = 48

// tabFromWindow sends Tab to the first tab group and Shift+Tab to the last while the window is its own
// first responder: SWT traverses only from a focused control, and AppKit did nothing with that Tab on
// macOS 26. tooling/port.sh calls it from windowSendEvent_.
func (this *Shell) tabFromWindow(nsEvent *cocoa.NSEvent) bool {
	return this.tabKeyFromWindow(nsEvent.KeyCode(), nsEvent.ModifierFlags())
}

func (this *Shell) tabKeyFromWindow(keyCode int16, mods int64) bool {
	if keyCode != keyCodeTab || mods&int64(cocoa.OSNSEventModifierFlagCommand|cocoa.OSNSEventModifierFlagControl|cocoa.OSNSAlternateKeyMask) != 0 {
		return false
	}
	w := this.View.Window()
	if w == nil || w.FirstResponder() == nil || w.FirstResponder().Id != w.Id {
		return false
	}
	return this.TraverseGroup(mods&int64(cocoa.OSNSEventModifierFlagShift) == 0)
}
