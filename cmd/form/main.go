// Command form: a two-column GridLayout form (Name/Email + OK) with a File > Quit menu bar.
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
	shell.SetText("Form")
	shell.SetLayout(&swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(2, false).Layout)

	fill := swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL)
	fill.WidthHint = 150
	swt.NewLabel(&shell.Composite, swt.SWTNONE).SetText("Name:")
	nameText := swt.NewText(&shell.Composite, swt.SWTBORDER)
	nameText.SetLayoutData(fill)

	fill2 := swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL)
	fill2.WidthHint = 150
	swt.NewLabel(&shell.Composite, swt.SWTNONE).SetText("Email:")
	emailText := swt.NewText(&shell.Composite, swt.SWTBORDER)
	emailText.SetLayoutData(fill2)

	ok := swt.NewButton(&shell.Composite, swt.SWTPUSH)
	ok.SetText("OK")
	okData := swt.NewGridData()
	okData.HorizontalSpan = 2
	ok.SetLayoutData(okData)
	ok.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Name:", nameText.GetText())
		fmt.Println("Email:", emailText.GetText())
	}))

	menuBar := swt.NewMenuParentStyle(&shell.Decorations, swt.SWTBAR)
	shell.SetMenuBar(menuBar)
	fileItem := swt.NewMenuItem(menuBar, swt.SWTCASCADE)
	fileItem.SetText("File")
	fileMenu := swt.NewMenuParentStyle(&shell.Decorations, swt.SWTDROP_DOWN)
	fileItem.SetMenu(fileMenu)
	quitItem := swt.NewMenuItem(fileMenu, swt.SWTPUSH)
	quitItem.SetText("Quit")
	quitItem.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		shell.Close()
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
