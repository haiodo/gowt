//go:build gowtqa

package swt

import (
	"os"

	"github.com/haiodo/gowt/internal/cocoa"
)

// qaNoActivate: a QA run (package qa) must not take the focus from the user's work.
func qaNoActivate() bool { return os.Getenv("GOWT_QA_SOCKET") != "" }

// qaFocusShell is the window QAFocus last gave a first responder. An inactive app has no key window,
// and Display.GetFocusControl reads the first responder of that window.
var qaFocusShell *Shell

// keyWindowQA is the key window, else the window QAFocus used. tooling/port.sh calls it from GetFocusControl.
func (this *Display) keyWindowQA() *cocoa.NSWindow {
	if w := this.application.KeyWindow(); w != nil {
		return w
	}
	if s := qaFocusShell; s != nil && !s.IsDisposed() {
		return s.window
	}
	return nil
}

// QAFocus makes c the first responder of its window without activating the app or ordering the window.
func QAFocus(c *Control) bool {
	c.CheckWidget()
	if !qaNoActivate() {
		return c.SetFocus()
	}
	w := c.View.Window()
	fv := c.FocusView()
	if w == nil || c.style&NO_FOCUS != 0 || !c.IsEnabled() || !c.IsVisible() || !fv.CanBecomeKeyView() {
		return false
	}
	qaFocusShell = c.GetShell()
	return w.MakeFirstResponder(upcastcocoaNSViewTococoaNSResponder(fv))
}
