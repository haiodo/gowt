package gowt

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/swt"
)

// Rows far from the visible part are hidden natively and come back when scrolled to; SWT keeps them
// visible, and the row with the focus is never hidden.
func TestScrollHidesFarRows(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			var field *TextView
			var sv *ScrollView
			w := app.Window("clip")
			w.SetSize(600, 400)
			w.Add(Scroll(Grow(), Ref(&sv), Text(GrowX(), Ref(&field))))
			sv.Add(feedRows(0, 200, "none")...)
			w.Show()
			app.Flush()
			w.Unwrap().ForceActive()
			field.Unwrap().SetFocus()
			kids := sv.Content().Unwrap().GetChildren()
			shown := func(k *swt.Control) bool { return !k.TopView().IsHidden() }
			for _, k := range kids {
				if !k.GetVisible() {
					t.Fatal("a row clipped off screen must stay visible to SWT")
				}
			}
			if !shown(kids[0]) || !shown(kids[3]) || shown(kids[150]) {
				t.Errorf("at the top: first rows shown %v %v, row 150 shown %v", shown(kids[0]), shown(kids[3]), shown(kids[150]))
			}
			sv.SetOrigin(0, int(sv.Unwrap().GetMinHeight()))
			last := kids[len(kids)-1]
			if !shown(last) || shown(kids[50]) {
				t.Errorf("at the bottom: last row shown %v, row 50 shown %v", shown(last), shown(kids[50]))
			}
			// Over SSH the window may not become key and then nothing has the focus.
			if !field.Unwrap().IsFocusControl() {
				t.Log("the field did not get the focus: the focus case is not checked")
			} else if !shown(kids[0]) {
				t.Error("the row with the focus was hidden")
			}
			sv.SetOrigin(0, 0)
			if !shown(kids[3]) || shown(last) {
				t.Errorf("back at the top: row 3 shown %v, last row shown %v", shown(kids[3]), shown(last))
			}
			w.Close()
		})
		if err != nil {
			t.Error(err)
		}
	})
}
