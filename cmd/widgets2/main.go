// Command widgets2 shows a TabFolder holding a Combo pair, a Table, and a ScrolledComposite.
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
	shell.SetText("Widgets2")
	shell.SetLayout(&swt.NewFillLayout().Layout)

	tabFolder := swt.NewTabFolder(shell, swt.NONE)

	comboTab := swt.NewCompositeParentStyle(tabFolder, swt.NONE)
	comboTab.SetLayout(&swt.NewRowLayoutType(swt.VERTICAL).Layout)

	readOnlyCombo := swt.NewCombo(comboTab, swt.READ_ONLY|swt.BORDER)
	readOnlyCombo.SetItems([]string{"One", "Two", "Three"})
	readOnlyCombo.Select(0)
	readOnlyCombo.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Combo (read-only) selected:", readOnlyCombo.GetText())
	}))

	editableCombo := swt.NewCombo(comboTab, swt.BORDER)
	editableCombo.SetItems([]string{"Alpha", "Beta", "Gamma"})
	editableCombo.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Combo (editable) selected:", editableCombo.GetText())
	}))

	tabCombo := swt.NewTabItem(tabFolder, swt.NONE)
	tabCombo.SetText("Combo")
	tabCombo.SetControl(comboTab)

	table := swt.NewTable(tabFolder, swt.BORDER|swt.FULL_SELECTION)
	table.SetHeaderVisible(true)
	table.SetLinesVisible(true)
	colName := swt.NewTableColumn(table, swt.NONE)
	colName.SetText("Name")
	colName.SetWidth(120)
	colValue := swt.NewTableColumn(table, swt.NONE)
	colValue.SetText("Value")
	colValue.SetWidth(80)

	rowByWidget := map[*swt.Widget]*swt.TableItem{}
	for i := 1; i <= 5; i++ {
		row := swt.NewTableItem(table, swt.NONE)
		row.SetTexts([]string{fmt.Sprintf("Row %d", i), fmt.Sprintf("%d", i*10)})
		rowByWidget[row.AsWidget()] = row
	}
	table.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Table selected:", rowByWidget[e.Item].GetText())
	}))

	tabTable := swt.NewTabItem(tabFolder, swt.NONE)
	tabTable.SetText("Table")
	tabTable.SetControl(table)

	scrolled := swt.NewScrolledComposite(tabFolder, swt.BORDER|swt.V_SCROLL)
	inner := swt.NewCompositeParentStyle(scrolled, swt.NONE)
	inner.SetLayout(&swt.NewRowLayoutType(swt.VERTICAL).Layout)
	for i := 1; i <= 30; i++ {
		swt.NewLabel(inner, swt.NONE).SetText(fmt.Sprintf("Label %d", i))
	}
	scrolled.SetContent(inner)
	scrolled.SetExpandHorizontal(true)
	scrolled.SetExpandVertical(true)
	size := inner.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	scrolled.SetMinSizeWidthHeight(size.X, size.Y)

	tabScrolled := swt.NewTabItem(tabFolder, swt.NONE)
	tabScrolled.SetText("Scrolled")
	tabScrolled.SetControl(scrolled)

	tabFolder.SetSelectionIndex(0)

	shell.SetSize(420, 380)
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
