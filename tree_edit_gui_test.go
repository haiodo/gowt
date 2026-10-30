package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

func TestTreeEdit(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			app.OnError(func(err error) { t.Error(err) })
			w := app.Window("Tree editing")
			var tree *TreeView
			var outside *TextView
			w.Add(VBox(Tree(Ref(&tree), Grow()), Text(Ref(&outside))))
			parent := tree.Node("parent")
			node := parent.Node("original")
			w.Show()
			app.After(200*time.Millisecond, func() {
				defer w.Close()
				commits, cancels := 0, 0
				commit := func(value string) { commits++; node.SetText(value) }
				cancel := func() { cancels++ }
				field := func() *swt.Text {
					for _, child := range tree.t.GetChildren() {
						if text, ok := child.Impl().(*swt.Text); ok && !text.IsDisposed() {
							return text
						}
					}
					t.Error("no tree editor")
					return nil
				}
				key := func(text *swt.Text, character uint16) {
					e := swt.NewEvent()
					e.Character = character
					text.NotifyListeners(swt.KeyDown, e)
					app.display.RunAsyncMessages(true)
				}
				tree.Edit(node, commit, cancel)
				text := field()
				if text == nil {
					return
				}
				selection, bounds, row := text.GetSelection(), text.GetBounds(), node.it.GetBoundsIndex(0)
				if !parent.Expanded() || text.GetText() != "original" || text.GetSelectionText() != "original" || selection.Y != 8 {
					t.Errorf("initial text=%q selection=%v expanded=%v", text.GetText(), selection, parent.Expanded())
				}
				if !text.IsFocusControl() || bounds.Y != row.Y || bounds.Height < row.Height || bounds.Width <= 0 {
					t.Errorf("focus=%v editor=%v row=%v", text.IsFocusControl(), bounds, row)
				}
				text.SetText("renamed")
				key(text, swt.CR)
				if commits != 1 || cancels != 0 || node.Text() != "renamed" || !text.IsDisposed() {
					t.Errorf("enter commits=%d cancels=%d text=%q", commits, cancels, node.Text())
				}
				tree.Edit(node, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				text.SetText("discard")
				key(text, swt.ESC)
				if commits != 1 || cancels != 1 || node.Text() != "renamed" {
					t.Errorf("escape commits=%d cancels=%d text=%q", commits, cancels, node.Text())
				}
				tree.Edit(node, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				text.SetText("")
				if !outside.Focus() {
					t.Error("outside text rejected focus")
				}
				app.display.RunAsyncMessages(true)
				if commits != 2 || node.Text() != "" {
					t.Errorf("focus loss commits=%d text=%q", commits, node.Text())
				}
				tree.Edit(node, func(value string) {
					commits++
					tree.Edit(node, commit, cancel)
				}, cancel)
				text = field()
				if text == nil {
					return
				}
				key(text, swt.CR)
				if commits != 3 || tree.cancelEdit == nil {
					t.Errorf("reentrant commit commits=%d active=%v", commits, tree.cancelEdit != nil)
				}
				text = field()
				if text == nil {
					return
				}
				text.Dispose()
				if cancels != 2 || tree.cancelEdit != nil {
					t.Errorf("dispose cancels=%d active=%v", cancels, tree.cancelEdit != nil)
				}
				tree.Edit(node, commit, cancel)
				parent.SetExpanded(false)
				if cancels != 3 || tree.cancelEdit != nil {
					t.Errorf("ancestor collapse cancels=%d active=%v", cancels, tree.cancelEdit != nil)
				}
				tree.Edit(node, commit, cancel)
				tree.Edit(node, commit, cancel)
				if cancels != 4 {
					t.Errorf("replacement cancels=%d", cancels)
				}
				node.Remove()
				if cancels != 5 || tree.cancelEdit != nil {
					t.Errorf("remove cancels=%d active=%v", cancels, tree.cancelEdit != nil)
				}
				node = parent.Node("reopen")
				reopened := false
				tree.Edit(node, nil, func() {
					reopened = true
					tree.Edit(node, nil, nil)
				})
				parent.SetExpanded(false)
				if !reopened || !parent.Expanded() || tree.cancelEdit == nil {
					t.Error("collapse cancellation could not reopen editing")
				}
				tree.Clear()
				if tree.cancelEdit != nil {
					t.Error("clear left an active editor")
				}
				node = tree.Node("pending")
				tree.Edit(node, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				e := swt.NewEvent()
				e.Character = swt.CR
				text.NotifyListeners(swt.KeyDown, e)
				node.Remove()
				node = tree.Node("replacement")
				tree.Edit(node, nil, nil)
				app.display.RunAsyncMessages(true)
				if cancels != 6 || commits != 3 || tree.cancelEdit == nil {
					t.Errorf("pending removal commits=%d cancels=%d active=%v", commits, cancels, tree.cancelEdit != nil)
				}
				tree.cancelEdit()
				tree.Edit(node, commit, cancel)
				text = field()
				if text == nil {
					return
				}
				text.SetText("committed before next edit")
				text.NotifyListeners(swt.FocusOut, swt.NewEvent())
				next := tree.Node("next")
				tree.Edit(next, nil, nil)
				app.display.RunAsyncMessages(true)
				text = field()
				if text == nil {
					return
				}
				if commits != 4 || text.GetText() != "next" {
					t.Errorf("pending replacement commits=%d text=%q", commits, text.GetText())
				}
				tree.cancelEdit()
				tree.Column("Name", 160)
				tree.Column("Details", 1200)
				app.Flush()
				tree.t.ShowColumn(tree.t.GetColumnIndex(1))
				if tree.t.GetClientArea().X == 0 && hscroll(tree.t.GetHorizontalBar()) == 0 {
					t.Error("test did not scroll horizontally")
				}
				tree.Edit(next, nil, nil)
				text = field()
				if text == nil {
					return
				}
				bounds, area := text.GetBounds(), tree.t.GetClientArea()
				if bounds.Width <= 0 || bounds.Height <= 0 || bounds.X < area.X || bounds.X+bounds.Width > area.X+area.Width || bounds.Y < area.Y || bounds.Y+bounds.Height > area.Y+area.Height {
					t.Errorf("scrolled editor=%v client=%v", bounds, area)
				}
			})
		})
		if err != nil {
			t.Error(err)
		}
	})
}
