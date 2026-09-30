// Command controlexample runs SWT's ControlExample (examples/controlexample, translated by j2go)
// with the tabs translated so far. With -snap <dir> it selects every tab, writes <dir>/<tab>.png,
// clicks one style/option checkbox on some tabs (<dir>/<tab>_<checkbox>.png) and exits.
package main

import (
	"flag"
	"runtime"

	"github.com/haiodo/gowt/examples/controlexample"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	snapDir := flag.String("snap", "", "write a PNG of every tab into this directory, then exit")
	flag.Parse()

	display := swt.NewDisplay()
	shell := swt.NewShellDisplayStyle(display, swt.SHELL_TRIM)
	shell.SetLayout(swt.NewFillLayout())
	instance := controlexample.NewControlExample(shell)
	shell.SetText(controlexample.ControlExampleGetResourceString("window.title"))
	controlexample.ControlExampleSetShellSize(shell)
	shell.Open()
	if *snapDir != "" {
		snapAll(display, shell, instance.TabFolder(), *snapDir)
	}
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	instance.Dispose()
	display.Dispose()
}
