package look_test

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/fluent"
)

// materialProblem wants the Fluent window background (Layer) on the pane and in the StyledText on it.
func materialProblem(w *gowt.Window, pane *gowt.Panel, white gowt.RGB) string {
	l := fluent.Current(pane.AsComposite().GetDisplay()).Layer()
	want := gowt.RGB{R: l.R, G: l.G, B: l.B}
	bg := pane.AsComposite().GetBackground().GetRGB()
	if got := (gowt.RGB{R: uint8(bg.Red), G: uint8(bg.Green), B: uint8(bg.Blue)}); got != want {
		return fmt.Sprintf("pane background %v, want Layer %v", got, want)
	}
	return paneColourProblem(w, want, white)
}
