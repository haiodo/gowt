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
	shell.SetLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(2, false))

	fill := swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL)
	fill.WidthHint = 150
	swt.NewLabel(shell, swt.NONE).SetText("Name:")
	nameText := swt.NewText(shell, swt.BORDER)
	nameText.SetLayoutData(fill)

	fill2 := swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL)
	fill2.WidthHint = 150
	swt.NewLabel(shell, swt.NONE).SetText("Email:")
	emailText := swt.NewText(shell, swt.BORDER)
	emailText.SetLayoutData(fill2)

	ok := swt.NewButton(shell, swt.PUSH)
	ok.SetText("OK")
	okData := swt.NewGridData()
	okData.HorizontalSpan = 2
	ok.SetLayoutData(okData)
	ok.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Name:", nameText.GetText())
		fmt.Println("Email:", emailText.GetText())
	}))

	menuBar := swt.NewMenuParentStyle(shell, swt.BAR)
	shell.SetMenuBar(menuBar)
	fileItem := swt.NewMenuItem(menuBar, swt.CASCADE)
	fileItem.SetText("File")
	fileMenu := swt.NewMenuParentStyle(shell, swt.DROP_DOWN)
	fileItem.SetMenu(fileMenu)
	quitItem := swt.NewMenuItem(fileMenu, swt.PUSH)
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
