// Command images loads 3 of the ControlExample test images through
// getResourceAsStream -> ImageData(InputStream) -> Image(Display, ImageData), and draws them on
// a Canvas with gc.DrawImage.
package main

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"structs"

	"github.com/ebitengine/purego/objc"
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
	shell.SetLayout(&swt.NewFillLayout().Layout)
	canvas := swt.NewCanvasParentStyle(shell, swt.NONE)
	canvas.SetBackgroundColorOnControlColor(display.GetSystemColor(swt.COLOR_WHITE))

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
	display.TimerExec(500, &task{func() { snapshot(os.Args[1]) }})
	display.TimerExec(1000, &task{func() { shell.Close() }})
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
	fmt.Println("disposed cleanly")
}

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

func str(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), s)
}

// snapshot renders the Shell's whole view hierarchy to a PNG via
// cacheDisplayInRect:toBitmapImageRep: - the same in-process approach cmd/paint's own snapshot
// used, since screencapture has no Screen Recording permission on this machine.
func snapshot(path string) {
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	win := shellWindow(app)
	view := win.Send(objc.RegisterName("contentView")).Send(objc.RegisterName("superview"))
	bounds := objc.Send[nsRect](view, objc.RegisterName("bounds"))
	rep := view.Send(objc.RegisterName("bitmapImageRepForCachingDisplayInRect:"), bounds)
	view.Send(objc.RegisterName("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(objc.RegisterName("dictionary"))
	data := rep.Send(objc.RegisterName("representationUsingType:properties:"), 4, dict)
	ok := objc.Send[bool](data, objc.RegisterName("writeToFile:atomically:"), str(path), true)
	fmt.Println("snapshot written:", ok)
}

// The SWTWindow among NSApp's windows (keyWindow depends on whether the app got activated).
func shellWindow(app objc.ID) objc.ID {
	wins := app.Send(objc.RegisterName("windows"))
	n := objc.Send[int](wins, objc.RegisterName("count"))
	for i := 0; i < n; i++ {
		w := wins.Send(objc.RegisterName("objectAtIndex:"), i)
		if objc.Send[bool](w, objc.RegisterName("isKindOfClass:"), objc.GetClass("SWTWindow")) {
			return w
		}
	}
	panic("no SWTWindow")
}
