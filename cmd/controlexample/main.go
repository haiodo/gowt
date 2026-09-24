// Command controlexample runs SWT's ControlExample (examples/controlexample, translated by j2go)
// with the tabs translated so far. With -snap <dir> it selects every tab, writes <dir>/<tab>.png,
// clicks one style/option checkbox on some tabs (<dir>/<tab>_<checkbox>.png) and exits.
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"structs"

	"github.com/ebitengine/purego/objc"
	"github.com/haiodo/gowt/examples/controlexample"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

type task struct{ fn func() }

func (t *task) Run() { t.fn() }

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

// The checkbox clicked on a tab after its first snapshot: each one makes the example recreate or
// reconfigure its sample widgets.
var clicks = map[string]string{"Button": "SWT.BORDER", "Canvas": "Caret", "Text": "SWT.BORDER", "Label": "SWT.SEPARATOR"}

// snapAll selects each tab the way a click does (NSTabView selectTabViewItemAtIndex:, which
// runs TabFolder's own delegate path), waits for layout and paint, then snapshots the window.
func snapAll(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string) {
	var steps []func()
	for i, item := range folder.GetItems() {
		name := item.GetText()
		steps = append(steps, func() { selectTab(folder, i) }, func() { snapshot(filepath.Join(dir, name+".png")) })
		if click, ok := clicks[name]; ok {
			var before *swt.Button
			steps = append(steps, func() {
				before = findButton(item.GetControl(), "One")
				clickButton(item.GetControl(), click)
			}, func() {
				if before != nil {
					fmt.Println("example widgets recreated:", before.IsDisposed() && findButton(item.GetControl(), "One") != nil)
				}
				snapshot(filepath.Join(dir, name+"_"+click+".png"))
			})
		}
	}
	steps = append(steps, func() { shell.Close() })
	for i, step := range steps {
		display.TimerExec(int32(500*(i+1)), &task{step})
	}
}

func selectTab(folder *swt.TabFolder, index int) {
	objc.ID(folder.View.Id).Send(objc.RegisterName("selectTabViewItemAtIndex:"), index)
}

// clickButton finds the Button labelled text under root and sends it performClick:, the real
// NSButton target/action path.
func clickButton(root *swt.Control, text string) {
	b := findButton(root, text)
	if b == nil {
		fmt.Println("button not found:", text)
		return
	}
	objc.ID(b.View.Id).Send(objc.RegisterName("performClick:"), 0)
	fmt.Printf("clicked %s: selection=%v\n", text, b.GetSelection())
}

func findButton(c *swt.Control, text string) *swt.Button {
	if b, ok := c.Impl().(*swt.Button); ok && b.GetText() == text {
		return b
	}
	composite, ok := c.Impl().(interface{ AsComposite() *swt.Composite })
	if !ok {
		return nil
	}
	for _, child := range composite.AsComposite().GetChildren() {
		if b := findButton(child, text); b != nil {
			return b
		}
	}
	return nil
}

type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

func str(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), s)
}

// snapshot renders the Shell via cacheDisplayInRect:toBitmapImageRep: - screencapture needs
// Screen Recording permission this machine does not grant.
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
	fmt.Println("snapshot", path, ok)
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
