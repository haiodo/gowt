// Command stack: a SashForm splits left (buttons) from right (a StackLayout of Groups);
// clicking a button switches the visible panel.
package main

import (
	"runtime"

	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("Stack")
	shell.SetLayout(swt.NewFillLayout())

	form := swt.NewSashForm(shell, swt.HORIZONTAL)

	left := swt.NewCompositeParentStyle(form, swt.NONE)
	left.SetLayout(swt.NewFillLayout())

	right := swt.NewCompositeParentStyle(form, swt.NONE)
	stackLayout := swt.NewStackLayout()
	right.SetLayout(stackLayout)

	groupA := swt.NewGroup(right, swt.NONE)
	groupA.SetText("Panel A")
	groupA.SetLayout(swt.NewFillLayout())
	swt.NewLabel(groupA, swt.NONE).SetText("This is panel A")

	groupB := swt.NewGroup(right, swt.NONE)
	groupB.SetText("Panel B")
	groupB.SetLayout(swt.NewFillLayout())
	swt.NewLabel(groupB, swt.NONE).SetText("This is panel B")

	groupC := swt.NewGroup(right, swt.NONE)
	groupC.SetText("Panel C")
	groupC.SetLayout(swt.NewFillLayout())
	swt.NewLabel(groupC, swt.NONE).SetText("This is panel C")

	stackLayout.TopControl = &groupA.Control
	form.SetWeights([]int32{1, 3})

	show := func(g *swt.Group) {
		stackLayout.TopControl = &g.Control
		right.Layout()
	}

	buttonA := swt.NewButton(left, swt.PUSH)
	buttonA.SetText("A")
	buttonA.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		show(groupA)
	}))

	buttonB := swt.NewButton(left, swt.PUSH)
	buttonB.SetText("B")
	buttonB.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		show(groupB)
	}))

	buttonC := swt.NewButton(left, swt.PUSH)
	buttonC.SetText("C")
	buttonC.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		show(groupC)
	}))

	shell.SetSize(500, 300)
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
