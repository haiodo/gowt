package swt

import "testing"

// Needs a display (the Linux stand): the accent provider must load and survive a reset to no accent.
func TestApplyAccent(t *testing.T) {
	d := NewDisplay()
	defer d.Dispose()
	sh := NewShellDisplayStyle(d, SHELL_TRIM)
	NewLabel(&sh.Composite, NONE).SetText("accent")
	d.applyAccent([]float64{0.2, 0.4, 0.9})
	if accentProvider == 0 {
		t.Fatal("accent provider not created")
	}
	sh.Open()
	for d.ReadAndDispatch() {
	}
	d.applyAccent([]float64{-1, -1, -1})
	for d.ReadAndDispatch() {
	}
}
