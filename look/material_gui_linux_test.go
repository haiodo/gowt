package look_test

import (
	"fmt"
	"os"
	"testing"

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

// The inspector gives its pane a background of its own: an alpha-0 StyledText over a pane that only
// inherited the shell background drew the theme's white base instead.
func TestInspectorPaneBackground(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	inspectorWindow(t, false, func(_ *gowt.App, _ *gowt.Window, _, _, insp *gowt.Panel) {
		c := insp.Unwrap()
		sys := c.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_BACKGROUND).GetRGB()
		want := gowt.RGB{R: uint8(sys.Red), G: uint8(sys.Green), B: uint8(sys.Blue)}
		bg := c.GetBackground()
		if bg == nil {
			t.Fatalf("inspector has no background of its own, want %v", want)
		}
		rgb := bg.GetRGB()
		if got := (gowt.RGB{R: uint8(rgb.Red), G: uint8(rgb.Green), B: uint8(rgb.Blue)}); got != want {
			t.Errorf("inspector background %v, want %v", got, want)
		}
	})
}
