package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

func setButtonBorderless(b *swt.Button) { gtk.AddStyleClass(b.Handle, "flat") }

func setButtonCircle(b *swt.Button) { gtk.AddStyleClass(b.Handle, "circular") }

// Small has set the font already.
func setButtonSmall(*swt.Button) {}

// SWT makes a TOGGLE a GtkToggleButton, which the theme draws on and off itself.
func styleToggle(*swt.Button) {}

// segmentCSS joins the toggle buttons of a Segmented as a "linked" GtkBox joins its buttons. SWT puts
// each button in a container of its own, so the theme's ".linked > button" rules never match.
const segmentCSS = `
button.gowt-segment-first { border-top-right-radius: 0; border-bottom-right-radius: 0; border-right-style: none; }
button.gowt-segment-middle { border-radius: 0; border-right-style: none; }
button.gowt-segment-last { border-top-left-radius: 0; border-bottom-left-radius: 0; }`

var segmentCSSLoaded bool

func styleSegment(b *swt.Button, i, n int) {
	if !segmentCSSLoaded {
		gtk.AddGlobalCSS(segmentCSS)
		segmentCSSLoaded = true
	}
	switch {
	case n == 1:
	case i == 0:
		gtk.AddStyleClass(b.Handle, "gowt-segment-first")
	case i == n-1:
		gtk.AddStyleClass(b.Handle, "gowt-segment-last")
	default:
		gtk.AddStyleClass(b.Handle, "gowt-segment-middle")
	}
}

func setButtonKind(b *swt.Button, kind int) {
	if kind&buttonPrimary != 0 {
		gtk.AddStyleClass(b.Handle, "suggested-action")
	}
	if kind&buttonDestructive != 0 {
		gtk.AddStyleClass(b.Handle, "destructive-action")
	}
}

func setAccessibleLabel(c *swt.Control, s string) { gtk.SetAccessibleName(c.Handle, s) }
