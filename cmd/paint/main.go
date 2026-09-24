// Command paint draws on a Canvas from a PaintListener: the drawRect: -> SWT.Paint -> GC path.
package main

import (
	"runtime"

	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

type painter func(e *swt.PaintEvent)

func (p painter) PaintControl(e *swt.PaintEvent) { p(e) }

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("Paint")
	shell.SetLayout(&swt.NewFillLayout().Layout)
	canvas := swt.NewCanvasParentStyle(shell, swt.NONE)
	canvas.SetBackgroundColorOnControl(display.GetSystemColor(swt.COLOR_WHITE))
	canvas.AddPaintListener(painter(func(e *swt.PaintEvent) {
		gc := e.Gc
		gc.SetBackground(display.GetSystemColor(swt.COLOR_DARK_GREEN))
		gc.FillRectangle(20, 20, 120, 80)
		gc.SetForeground(display.GetSystemColor(swt.COLOR_RED))
		gc.SetLineWidth(3)
		gc.DrawLine(20, 130, 280, 130)
		gc.SetForeground(display.GetSystemColor(swt.COLOR_BLUE))
		gc.DrawOval(160, 20, 120, 80)
		gc.SetForeground(display.GetSystemColor(swt.COLOR_BLACK))
		gc.DrawString("Hello, GC", 20, 150)
	}))
	shell.SetSize(300, 220)
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
