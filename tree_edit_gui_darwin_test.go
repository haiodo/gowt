package gowt

import (
	"os"
	"structs"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/haiodo/gowt/internal/cocoa"
)

func TestTreeEditNativeKeys(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			app.OnError(func(err error) { t.Error(err) })
			w := app.Window("Native tree editing")
			var tree *TreeView
			w.Add(Tree(Ref(&tree), Grow()))
			node := tree.Node("original")
			w.Show()
			app.After(200*time.Millisecond, func() {
				defer w.Close()
				commits, cancels := 0, 0
				commit := func(value string) { commits++; node.SetText(value) }
				cancel := func() { cancels++ }
				tree.Edit(node, commit, cancel)
				win := tree.t.View.Window()
				key := func(character string, keyCode uint16) {
					s := cocoa.NSStringStringWith(character)
					for _, typ := range []uint64{10, 11} {
						e := objc.ID(objc.GetClass("NSEvent")).Send(objc.RegisterName("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"),
							typ, struct {
								_    structs.HostLayout
								X, Y float64
							}{}, uint64(0), 0.0, win.WindowNumber(), objc.ID(0), objc.ID(s.Id), objc.ID(s.Id), false, keyCode)
						cocoa.NSApplicationSharedApplication().PostEvent(cocoa.NewNSEventOverload1(int64(e)), false)
					}
					for app.display.ReadAndDispatch() {
					}
				}
				key("x", 7)
				key("\r", 36)
				if commits != 1 || cancels != 0 || node.Text() != "x" {
					t.Errorf("native Enter commits=%d cancels=%d text=%q", commits, cancels, node.Text())
				}
				tree.Edit(node, commit, cancel)
				key("y", 16)
				key("\x1b", 53)
				if commits != 1 || cancels != 1 || node.Text() != "x" {
					t.Errorf("native Esc commits=%d cancels=%d text=%q", commits, cancels, node.Text())
				}
			})
		})
		if err != nil {
			t.Error(err)
		}
	})
}
