// Command hello opens a Shell with one push button: the Display+Shell+Button path end to end.
package main

import (
	"fmt"
	"runtime"

	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetLayout(swt.NewFillLayout())
	button := swt.NewButton(shell, swt.PUSH)
	button.SetText("Hello")
	button.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Hello clicked")
	}))
	shell.Pack()
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
