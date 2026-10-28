package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// A borderless toggle shows its state in the accent colour of its symbol and title.
func setButtonBorderless(b *swt.Button) {
	cocoa.SetButtonBorderless(b.View)
	if b.GetStyle()&swt.TOGGLE != 0 {
		cocoa.SyncButtonTint(b.View)
		onSelect(b, func(*swt.SelectionEvent) { cocoa.SyncButtonTint(b.View) })
	}
}

// syncToggle follows a state set by the program on a borderless toggle.
func syncToggle(b *swt.Button) {
	if b.GetStyle()&swt.TOGGLE != 0 && !cocoa.ButtonBordered(b.View) {
		cocoa.SyncButtonTint(b.View)
	}
}

func setButtonSmall(b *swt.Button) { cocoa.SetControlSmall(b.View) }

// newChevron is a push-on-push-off button with the disclosure bezel: on is expanded.
func newChevron(p *swt.Composite) *swt.Button {
	b := swt.NewButton(p, swt.TOGGLE)
	cocoa.SetButtonDisclosure(b.View)
	return b
}

func setChevron(b *swt.Button, expanded bool) { b.SetSelection(expanded) }

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
