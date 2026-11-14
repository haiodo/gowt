package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// A row of a Table takes an in-place editor: Enter commits the typed text, Esc and a focus loss leave it.
func TestTableEdit(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			app.OnError(func(err error) { t.Error(err) })
			w := app.Window("Table editing")
			var table *TableView
			var outside *TextView
			w.Add(VBox(Grow(), Table(Ref(&table), Grow()), Text(Ref(&outside))))
			row := table.Row("original")
			table.Row("other")
			w.SetSize(400, 300)
			w.Show()
			app.After(200*time.Millisecond, func() {
				defer w.Close()
				commits, cancels := 0, 0
				commit := func(value string) {
					commits++
					row.SetTextAt(0, value)
				}
				cancel := func() { cancels++ }
				field := func() *swt.Text {
					for _, child := range table.t.GetChildren() {
						if text, ok := child.Impl().(*swt.Text); ok && !text.IsDisposed() {
							return text
						}
					}
					t.Error("no table editor")
					return nil
				}
				key := func(text *swt.Text, character uint16) {
					e := swt.NewEvent()
					e.Character = character
					text.NotifyListeners(swt.KeyDown, e)
					app.display.RunAsyncMessages(true)
				}
				table.Edit(row, commit, cancel)
				text := field()
				if text == nil {
					return
				}
				// No focus check: the window is not key in every test run.
				bounds, cells := text.GetBounds(), table.t.GetItem(0).GetTextBounds(0)
				if text.GetText() != "original" || text.GetSelectionText() != "original" {
					t.Errorf("text=%q selection=%q", text.GetText(), text.GetSelectionText())
				}
				if bounds.Y != cells.Y || bounds.Height < cells.Height || bounds.Width <= 0 {
					t.Errorf("editor=%v cell=%v", bounds, cells)
				}
				text.SetText("renamed")
				key(text, swt.CR)
				if commits != 1 || cancels != 0 || row.TextAt(0) != "renamed" || !text.IsDisposed() {
					t.Errorf("enter commits=%d cancels=%d text=%q", commits, cancels, row.TextAt(0))
				}
				table.Edit(row, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				text.SetText("discard")
				key(text, swt.ESC)
				if commits != 1 || cancels != 1 || row.TextAt(0) != "renamed" {
					t.Errorf("escape commits=%d cancels=%d text=%q", commits, cancels, row.TextAt(0))
				}
				table.Edit(row, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				text.SetText("lost")
				if !outside.Focus() {
					t.Error("outside text rejected focus")
				}
				app.display.RunAsyncMessages(true)
				if commits != 2 || row.TextAt(0) != "lost" {
					t.Errorf("focus loss commits=%d text=%q", commits, row.TextAt(0))
				}
			})
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
