package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/haiodo/gowt/swt"
)

// The checkbox clicked on a tab after its first snapshot: each one makes the example recreate or
// reconfigure its sample widgets.
var clicks = map[string]string{"Button": "SWT.BORDER", "Canvas": "Caret", "Text": "SWT.BORDER", "Label": "SWT.SEPARATOR"}

// snapHooks is the part of the -snap driver that talks to the windowing system.
type snapHooks struct {
	meta      func(path string)
	selectTab func(folder *swt.TabFolder, index int)
	click     func(root *swt.Control, text string)
	shot      func(path string)
}

// snapRun selects each tab, waits for layout and paint, snapshots the window, clicks one checkbox on
// some tabs and snapshots again, then closes the shell. Steps run 500 ms apart from the main loop.
// stepTimeout bounds one step (and the wait for the next timer): a stuck native call or a timer that
// never fires ends the run with the step's name and all goroutine stacks instead of hanging.
// stuckTrace is extra per-OS state the watchdog prints (the win32 dispatch ring).
var stuckTrace = func() string { return "" }

const stepTimeout = 20 * time.Second

func snapRun(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string, h snapHooks) {
	var steps []func()
	var progress atomic.Int64 // unix nanos of the last step start
	var current atomic.Value  // name of the running step
	progress.Store(time.Now().UnixNano())
	current.Store("start")
	go func() {
		for range time.Tick(time.Second) {
			if time.Since(time.Unix(0, progress.Load())) > stepTimeout {
				buf := make([]byte, 1<<18)
				fmt.Fprintf(os.Stderr, "snap: step %q stuck for more than %v\n%s\n%s", current.Load(), stepTimeout, buf[:runtime.Stack(buf, true)], stuckTrace())
				os.Exit(3)
			}
		}
	}()
	named := func(name string, f func()) func() {
		return func() {
			current.Store(name)
			progress.Store(time.Now().UnixNano())
			fmt.Fprintln(os.Stderr, "snap step:", name)
			f()
			current.Store("after " + name)
			progress.Store(time.Now().UnixNano())
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	steps = append(steps, named("meta", func() { h.meta(filepath.Join(dir, "meta.txt")) }))
	for i, item := range folder.GetItems() {
		name := item.GetText()
		steps = append(steps, named("select tab "+name, func() { h.selectTab(folder, i) }), named("shot "+name, func() { h.shot(filepath.Join(dir, name+".png")) }))
		if click, ok := clicks[name]; ok {
			var before *swt.Button
			steps = append(steps, named("click "+name+" "+click, func() {
				before = findButton(item.GetControl(), "One")
				h.click(item.GetControl(), click)
			}), named("shot "+name+" "+click, func() {
				if before != nil {
					fmt.Println("example widgets recreated:", before.IsDisposed() && findButton(item.GetControl(), "One") != nil)
				}
				h.shot(filepath.Join(dir, name+"_"+click+".png"))
			}))
		}
	}
	steps = append(steps, named("close", func() { shell.Close() }))
	// The steps run in the main loop (snapTick), never inside a WM_TIMER callback; the waker only
	// makes Sleep return.
	next, due := 0, time.Now().Add(500*time.Millisecond)
	snapTick = func() {
		if next < len(steps) && time.Now().After(due) {
			step := steps[next]
			next++
			step()
			due = time.Now().Add(500 * time.Millisecond)
		}
	}
	go func() {
		defer func() { recover() }() // Wake panics once the display is disposed
		for !display.IsDisposed() {
			time.Sleep(100 * time.Millisecond)
			display.Wake()
		}
	}()
}

// snapTick is called by main between ReadAndDispatch calls while -snap runs.
var snapTick func()

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
