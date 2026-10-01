package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/haiodo/gowt/swt"
)

type task struct{ fn func() }

func (t *task) Run() { t.fn() }

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
// some tabs and snapshots again, then closes the shell. Steps are timers 500 ms apart.
func snapRun(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string, h snapHooks) {
	var steps []func()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	steps = append(steps, func() { h.meta(filepath.Join(dir, "meta.txt")) })
	for i, item := range folder.GetItems() {
		name := item.GetText()
		steps = append(steps, func() { h.selectTab(folder, i) }, func() { h.shot(filepath.Join(dir, name+".png")) })
		if click, ok := clicks[name]; ok {
			var before *swt.Button
			steps = append(steps, func() {
				before = findButton(item.GetControl(), "One")
				h.click(item.GetControl(), click)
			}, func() {
				if before != nil {
					fmt.Println("example widgets recreated:", before.IsDisposed() && findButton(item.GetControl(), "One") != nil)
				}
				h.shot(filepath.Join(dir, name+"_"+click+".png"))
			})
		}
	}
	steps = append(steps, func() { shell.Close() })
	for i, step := range steps {
		display.TimerExec(int32(500*(i+1)), &task{step})
	}
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
