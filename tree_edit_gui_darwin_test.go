package gowt

import (
	"math"
	"os"
	"structs"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// The editor of an inline tree rename is as tall as its row; AppKit puts the text of a borderless
// field at the top of the frame, so the value must be drawn on the line in the middle of the row.
func TestTreeEditTextCentred(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		err := Run(func(app *App) {
			app.OnError(func(err error) { t.Error(err) })
			w := app.Window("Tree editing")
			var tree *TreeView
			w.Add(Tree(Ref(&tree), Grow()))
			w.SetSize(400, 300)
			node := tree.Node("Bars")
			w.Show()
			app.After(200*time.Millisecond, func() {
				defer w.Close()
				tree.Edit(node, nil, nil)
				var field *swt.Text
				for _, child := range tree.t.GetChildren() {
					if text, ok := child.Impl().(*swt.Text); ok && !text.IsDisposed() {
						field = text
					}
				}
				if field == nil {
					t.Fatal("no tree editor")
				}
				row, bounds := node.it.GetBoundsIndex(0), field.GetBounds()
				if bounds.Y != row.Y || bounds.Height != row.Height {
					t.Errorf("editor %v, row %v: the field must hug the row", bounds, row)
				}
				// The dark pixels of the value, in field points: the row's own text is covered by the
				// editor, so what is left is the editing text. No field editor means the edit has not
				// started (a race), not a geometry defect - say so instead of measuring a settled cell.
				view := field.View
				if cocoa.Msg(view.Id, "currentEditor") == 0 {
					t.Fatal("no field editor yet: the tree edit has not started")
				}
				scale := cocoa.BackingScale(view)
				area := view.Bounds()
				rep := view.BitmapImageRepForCachingDisplayInRect(area)
				view.CacheDisplayInRect(area, rep)
				top, bottom := area.Height, 0.0
				for y := int64(0); y < rep.PixelsHigh(); y++ {
					for x := int64(0); x < rep.PixelsWide(); x++ {
						c := rep.ColorAtX(x, y)
						if c.AlphaComponent() > 0.5 && c.RedComponent() < 0.3 && c.GreenComponent() < 0.3 && c.BlueComponent() < 0.3 {
							top = math.Min(top, float64(y)/scale)
							bottom = math.Max(bottom, float64(y)/scale)
						}
					}
				}
				if bottom <= top {
					t.Fatal("no text in the tree editor")
				}
				if centre := (top + bottom) / 2; math.Abs(centre-float64(bounds.Height)/2) > 1.5 {
					t.Errorf("text rows %v..%v (centre %v) in a row of %v: not centred", top, bottom, centre, bounds.Height)
				}
			})
		})
		if err != nil {
			t.Error(err)
		}
	})
}

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
				var field *swt.Text
				for _, child := range tree.t.GetChildren() {
					if text, ok := child.Impl().(*swt.Text); ok && !text.IsDisposed() {
						field = text
					}
				}
				if field == nil {
					t.Fatal("no tree editor")
				}
				// The row views of a macOS tree are subviews of the outline view; the editor
				// must sit above them, or it draws behind the row it edits.
				sv := field.View.Superview()
				subs := sv.Subviews()
				if top := subs.ObjectAtIndex(subs.Count() - 1); top.Id != field.View.Id {
					t.Error("editor is not above the tree row views")
				}
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
