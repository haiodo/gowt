// Command images loads 3 of the ControlExample test images through
// getResourceAsStream -> ImageData(InputStream) -> Image(Display, ImageData), and draws them on
// a Canvas with gc.DrawImage.
package main

import (
	"embed"
	"fmt"
	"os"
	"runtime"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

//go:embed testdata
var testdataFS embed.FS

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

type painter func(e *swt.PaintEvent)

func (p painter) PaintControl(e *swt.PaintEvent) { p(e) }

type task struct{ fn func() }

func (t *task) Run() { t.fn() }

func loadImage(display *swt.Display, name string) *swt.Image {
	stream := jrt.GetResourceAsStream(name)
	if stream == nil {
		panic("resource not found: " + name)
	}
	data := swt.NewImageDataStream(stream)
	return swt.NewImageDeviceData(display, data)
}

func main() {
	jrt.RegisterResources(testdataFS)
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("Images")
	shell.SetLayout(swt.NewFillLayout())
	canvas := swt.NewCanvasParentStyle(shell, swt.NONE)
	canvas.SetBackgroundWithColor(display.GetSystemColor(swt.COLOR_WHITE))

	png := loadImage(display, "testdata/controlexample/backgroundImage.png")
	gif := loadImage(display, "testdata/controlexample/closedFolder.gif")
	bmp := loadImage(display, "testdata/controlexample/red.bmp")

	canvas.AddPaintListener(painter(func(e *swt.PaintEvent) {
		gc := e.Gc
		gc.DrawImage(png, 20, 20)
		gc.DrawImage(gif, 120, 20)
		gc.DrawImage(bmp, 160, 20)
	}))
	shell.SetSize(300, 120)
	shell.Open()
	if len(os.Args) > 1 {
		display.TimerExec(500, &task{func() { snapshot(os.Args[1]) }})
		display.TimerExec(1000, &task{func() { shell.Close() }})
	}
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
	fmt.Println("disposed cleanly")
}
