package gowt

import (
	"fmt"
	"os/exec"

	"github.com/haiodo/gowt/swt"
)

func postBlocked(*App) string {
	if _, err := exec.LookPath("xdotool"); err != nil {
		return "xdotool is not installed"
	}
	return ""
}

// pointer converts display points to the coordinates postMouse takes; call it on the UI thread.
func pointer(_ *App, p swt.Point) swt.Point { return p }

// postMouse posts a left-button event at x, y in display points. Display.Post queues GDK events the
// X server never sees, so a GTK drag gets nothing from it.
func postMouse(_ *App, typ, x, y int32) bool {
	args := map[int32][]string{swt.MouseMove: {"mousemove", fmt.Sprint(x), fmt.Sprint(y)},
		swt.MouseDown: {"mousedown", "1"}, swt.MouseUp: {"mouseup", "1"}}[typ]
	return exec.Command("xdotool", args...).Run() == nil
}
