// Command jfacedemo builds a form from JFace's fluent widget factories and layout factories only:
// labels, texts and a group, a table under a TableColumnLayout, and a row of buttons. With a file
// argument it writes a PNG of the window and closes it.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/haiodo/gowt/jface"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

type task struct{ fn func() }

func (t *task) Run() { t.fn() }

func fill(grabV bool) *swt.GridData {
	return jface.GridDataFactoryFillDefaults().Grab(true, grabV).Create()
}

func main() {
	display := swt.NewDisplay()
	shell := jface.WidgetFactoryShell(swt.SHELL_TRIM).Text("JFace factories").
		Layout(jface.GridLayoutFactorySwtDefaults().Create()).CreateDisplay(display)

	person := jface.WidgetFactoryGroup(swt.NONE).Text("Person").
		Layout(jface.GridLayoutFactorySwtDefaults().NumColumns(2).Create()).
		LayoutData(fill(false)).Create(shell)
	jface.WidgetFactoryLabel(swt.NONE).Text("Name:").Create(person)
	name := jface.WidgetFactoryText(swt.BORDER).Message("Full name").LayoutData(fill(false)).Create(person)
	jface.WidgetFactoryLabel(swt.NONE).Text("Email:").Create(person)
	email := jface.WidgetFactoryText(swt.BORDER).Message("name@example.org").LayoutData(fill(false)).Create(person)
	jface.WidgetFactoryLabel(swt.NONE).Text("Notes:").
		LayoutData(jface.GridDataFactoryCreate(swt.BEGINNING).Create()).Create(person)
	jface.WidgetFactoryText(swt.BORDER | swt.MULTI | swt.WRAP | swt.V_SCROLL).
		LayoutData(jface.GridDataFactoryFillDefaults().Grab(true, false).Hint(swt.DEFAULT, 60).Create()).Create(person)

	layout := jface.NewTableColumnLayout()
	tableHolder := jface.WidgetFactoryComposite(swt.NONE).Layout(layout).LayoutData(fill(true)).Create(shell)
	table := jface.WidgetFactoryTable(swt.BORDER | swt.FULL_SELECTION).HeaderVisible(true).LinesVisible(true).Create(tableHolder)
	for _, c := range []struct {
		title string
		data  *jface.ColumnWeightData
	}{{"Name", jface.NewColumnWeightDataWeightMinimumWidth(3, 80)}, {"Email", jface.NewColumnWeightDataWeightMinimumWidth(4, 100)},
		{"Role", jface.NewColumnWeightData(2)}} {
		col := jface.WidgetFactoryTableColumn(swt.NONE).Text(c.title).Create(table)
		layout.SetColumnData(col, c.data)
	}
	for _, r := range [][]string{{"Ada Lovelace", "ada@example.org", "Analyst"}, {"Alan Turing", "alan@example.org", "Cryptanalyst"}} {
		swt.NewTableItem(table, swt.NONE).SetTexts(r)
	}

	buttons := jface.WidgetFactoryComposite(swt.NONE).
		Layout(jface.GridLayoutFactoryFillDefaults().NumColumns(3).EqualWidth(true).Create()).LayoutData(fill(false)).Create(shell)
	jface.WidgetFactoryButton(swt.PUSH).Text("Add").LayoutData(fill(false)).OnSelect(func(e *swt.SelectionEvent) {
		swt.NewTableItem(table, swt.NONE).SetTexts([]string{name.GetText(), email.GetText(), "Member"})
		name.SetText("")
		email.SetText("")
	}).Create(buttons)
	jface.WidgetFactoryButton(swt.PUSH).Text("Clear").LayoutData(fill(false)).OnSelect(func(e *swt.SelectionEvent) {
		table.RemoveAll()
	}).Create(buttons)
	jface.WidgetFactoryButton(swt.PUSH).Text("Close").LayoutData(fill(false)).OnSelect(func(e *swt.SelectionEvent) {
		shell.Close()
	}).Create(buttons)

	shell.SetSize(420, 460)
	shell.Open()
	if len(os.Args) > 1 {
		display.TimerExec(700, &task{func() { snapshot(os.Args[1]) }})
		display.TimerExec(1200, &task{func() { shell.Close() }})
	}
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
	fmt.Println("disposed cleanly")
}
