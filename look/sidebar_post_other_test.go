//go:build !darwin && !windows

package look_test

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// postBlocked says why events posted to the system cannot reach the test window, "" when they can.
func postBlocked(*gowt.App) string { return "" }

// postMouse posts a left-button event at x, y in display points. Linux uses xdotool: Display.Post
// there queues GDK button events the X server never sees, so a GTK sash gets no motion.
func postMouse(app *gowt.App, typ, x, y int32) bool {
	if runtime.GOOS == "linux" {
		xdotool, err := exec.LookPath("xdotool")
		if err != nil {
			return false
		}
		args := map[int32][]string{swt.MouseMove: {"mousemove", fmt.Sprint(x), fmt.Sprint(y)},
			swt.MouseDown: {"mousedown", "1"}, swt.MouseUp: {"mouseup", "1"}}[typ]
		return exec.Command(xdotool, args...).Run() == nil
	}
	e := swt.NewEvent()
	e.Type, e.X, e.Y, e.Button = typ, x, y, 1
	return app.Unwrap().Post(e)
}
