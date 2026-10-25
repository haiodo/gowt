package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func setButtonBorderless(b *swt.Button) { cocoa.SetButtonBorderless(b.View) }

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
