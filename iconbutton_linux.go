package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

func setButtonBorderless(b *swt.Button) { gtk.AddStyleClass(b.Handle, "flat") }

func setButtonCircle(b *swt.Button) { gtk.AddStyleClass(b.Handle, "circular") }

func setButtonKind(b *swt.Button, kind int) {
	if kind&buttonPrimary != 0 {
		gtk.AddStyleClass(b.Handle, "suggested-action")
	}
	if kind&buttonDestructive != 0 {
		gtk.AddStyleClass(b.Handle, "destructive-action")
	}
}

func setAccessibleLabel(c *swt.Control, s string) { gtk.SetAccessibleName(c.Handle, s) }
