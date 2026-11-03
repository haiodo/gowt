package gowt

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestOverlayThumb(t *testing.T) {
	r := &swt.Rectangle{Width: 100, Height: 100}
	x, y, w, h := overlayThumb(r, true, 0, 200, 50, 50)
	if x != 96 || y != 26 || w != 2 || h != 24 {
		t.Errorf("vertical thumb %v %v %v %v", x, y, w, h)
	}
	x, y, w, h = overlayThumb(r, false, 0, 200, 50, 150)
	if x != 74 || y != 96 || w != 24 || h != 2 {
		t.Errorf("horizontal thumb %v %v %v %v", x, y, w, h)
	}
}

func TestOverlayScroll(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	guiRun(t, func(app *App, w *Window) {
		var text *TextView
		var table *TableView
		var tree *TreeView
		var list *ListView
		var styled *StyledTextView
		rows := make([]string, 30)
		for i := range rows {
			rows[i] = fmt.Sprintf("row %d", i)
		}
		w.Add(VBox(
			Text(Multiline(), Scrollbars(), Size(0, 100), Value(strings.Repeat("line\n", 30)), Ref(&text)),
			Table(Size(0, 100), Init(func(v *TableView) {
				for _, row := range rows {
					v.Row(row)
				}
			}), Ref(&table)),
			Tree(Size(0, 100), Init(func(v *TreeView) {
				root := v.Node("root")
				for _, row := range rows {
					root.Node(row)
				}
				root.SetExpanded(true)
			}), Ref(&tree)),
			List(rows, Size(0, 100), Ref(&list)),
			StyledText(Scrollbars(), Size(0, 100), Value(strings.Repeat("line\n", 30)), Ref(&styled)),
		))
		w.SetSize(400, 600)
		w.Show()
		app.Flush()
		for _, item := range []struct {
			name string
			c    *swt.Control
			s    overlayScrollable
		}{
			{"Text", &text.t.Control, text.t}, {"Table", &table.t.Control, table.t}, {"Tree", &tree.t.Control, tree.t}, {"List", &list.l.Control, list.l}, {"StyledText", &styled.t.Control, styled.t},
		} {
			bar := item.s.GetVerticalBar()
			if bar == nil || bar.GetVisible() {
				t.Errorf("%s keeps the native vertical scrollbar", item.name)
			}
			wheel := swt.NewEvent()
			wheel.Type, wheel.Detail, wheel.Count = swt.MouseWheel, swt.SCROLL_LINE, -3
			item.c.NotifyListeners(swt.MouseWheel, wheel)
			app.Flush()
			if got := bar.GetSelection(); got == 0 {
				t.Errorf("%s did not scroll with the mouse wheel", item.name)
			}
		}
	})
}
