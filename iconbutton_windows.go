package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// fluent paints the push button: Borderless is the WinUI 3 Subtle button, Circle a round one.
func setButtonBorderless(b *swt.Button) { fluent.Button(b, true, false, false) }

func setButtonCircle(b *swt.Button) { fluent.Button(b, false, true, false) }

// Accessible is a stub in the toolkit on Windows.
func setAccessibleLabel(*swt.Control, string) {}

func setButtonKind(b *swt.Button, kind int) {
	fluent.ButtonKind(b, kind&buttonPrimary != 0, kind&buttonDestructive != 0)
}
