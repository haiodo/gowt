package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// fluent paints the push button: Borderless is the WinUI 3 Subtle button, Circle a round one.
func setButtonBorderless(b *swt.Button) { fluent.Button(b, true, false, false) }

func setButtonCircle(b *swt.Button) { fluent.Button(b, false, true, false) }

// Small has set the font already; fluent sizes the button from it.
func setButtonSmall(*swt.Button) {}

// fluent paints a toggle as the WinUI ToggleButton, with the accent fill while it is on.
func styleToggle(b *swt.Button) { fluent.Button(b, false, false, false) }

// A segment is as tall as a push button under controlHeight.
func styleSegment(b *swt.Button, i, n int) {
	fluent.Segment(b, i, n)
	if d, ok := b.GetLayoutData().(*swt.GridData); ok && !ChosenDensity().Legacy() {
		d.HeightHint = int32(Spacing.ControlHeight)
	}
}

// Accessible is a stub in the toolkit on Windows.
func setAccessibleLabel(*swt.Control, string) {}

func setButtonKind(b *swt.Button, kind int) {
	fluent.ButtonKind(b, kind&buttonPrimary != 0, kind&buttonDestructive != 0)
}
