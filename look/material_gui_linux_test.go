package look_test

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// materialProblem wants the window background of the theme on the pane and in the StyledText on it.
func materialProblem(w *gowt.Window, pane *gowt.Panel, white gowt.RGB) string {
	c := pane.AsComposite()
	sys := c.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_BACKGROUND).GetRGB()
	want := gowt.RGB{R: uint8(sys.Red), G: uint8(sys.Green), B: uint8(sys.Blue)}
	bg := c.GetBackground().GetRGB()
	if got := (gowt.RGB{R: uint8(bg.Red), G: uint8(bg.Green), B: uint8(bg.Blue)}); got != want {
		return fmt.Sprintf("pane background %v, want the window background %v", got, want)
	}
	return paneColourProblem(w, want, white)
}
