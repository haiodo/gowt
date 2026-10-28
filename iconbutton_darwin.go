package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// A borderless toggle is the scope-bar button: no border until the pointer is over it, a recessed fill when on.
func setButtonBorderless(b *swt.Button) {
	if b.GetStyle()&swt.TOGGLE != 0 {
		cocoa.SetButtonRecessed(b.View)
		return
	}
	cocoa.SetButtonBorderless(b.View)
}

func setButtonSmall(b *swt.Button) { cocoa.SetControlSmall(b.View) }

// SWT makes a TOGGLE a push-on-push-off NSButton, which AppKit draws on and off itself.
func styleToggle(*swt.Button) {}

func setButtonCircle(b *swt.Button) {
	cocoa.SetButtonCircle(b.View)
	b.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { cocoa.ClipGlassCircle(b.View) }))
}

func setAccessibleLabel(c *swt.Control, s string) { cocoa.SetAccessibilityLabel(c.View, s) }

// The default button (Primary) is drawn by AppKit in the accent colour; Destructive adds the system's destructive action.
func setButtonKind(b *swt.Button, kind int) {
	if kind&buttonDestructive != 0 {
		cocoa.SetButtonDestructive(b.View)
	}
}
