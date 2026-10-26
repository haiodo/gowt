package swt

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

func init() {
	runtime.LockOSThread()
	uiRun = func(f func()) {
		done := make(chan struct{})
		vtMainq <- func() { defer close(done); f() }
		<-done
	}
}

var vtMainq = make(chan func())

// The toolkit needs the main thread: the tests run on their own goroutine and hand the UI work back.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-vtMainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

// Run with GOWT_GUI_TEST=1, and again with GOWT_VIEW_TABLE=0: the native table must be view-based unless the flag is 0.
func TestViewTableEngaged(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	want := os.Getenv("GOWT_VIEW_TABLE") != "0"
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		vtCheckEngaged(t, want)
	}
	<-done
}

func vtCheckEngaged(t *testing.T, want bool) {
	if viewTableEnabled != want {
		t.Fatalf("viewTableEnabled = %v, want %v", viewTableEnabled, want)
	}
	d := NewDisplay()
	defer d.Dispose()
	shell := NewShell()
	shell.SetSize(600, 600)

	table := NewTable(shell, CHECK|FULL_SELECTION)
	table.SetHeaderVisible(true)
	table.SetBounds(0, 0, 280, 380)
	for i := 0; i < 2; i++ {
		NewTableColumn(table, NONE).SetWidth(120)
	}
	for i := 0; i < 5; i++ {
		NewTableItem(table, NONE).SetTexts([]string{"a", "b"})
	}

	tree := NewTree(shell, CHECK|FULL_SELECTION)
	tree.SetBounds(300, 0, 280, 380)
	NewTreeColumn(tree, NONE).SetWidth(120)
	parent := NewTreeItem(tree, NONE)
	parent.SetTexts([]string{"p"})
	NewTreeItemParentItemStyle(parent, NONE).SetTexts([]string{"c"})
	parent.SetExpanded(true)

	list := NewList(shell, BORDER|V_SCROLL)
	list.SetBounds(0, 400, 280, 150)
	list.SetItems([]string{"Apples", "Oranges"})

	shell.Open()
	for end := time.Now().Add(1500 * time.Millisecond); time.Now().Before(end); {
		if !d.ReadAndDispatch() {
			time.Sleep(10 * time.Millisecond)
		}
	}

	tv := castcocoaNSViewTococoaNSTableView(table.View).Id
	ov := castcocoaNSViewTococoaNSOutlineView(tree.View).Id
	lv := castcocoaNSViewTococoaNSTableView(list.View).Id
	for _, c := range []struct {
		name string
		view int64
		col  int64
	}{{"table column 0", tv, 0}, {"table column 1", tv, 1}, {"tree column 0", ov, 0}, {"list column 0", lv, 0}} {
		got := cocoa.Msg(c.view, "viewAtColumn:row:makeIfNecessary:", c.col, 0, 0) != 0
		if got != want {
			t.Errorf("%s: viewAtColumn non-nil = %v, want %v", c.name, got, want)
		}
	}
	if (vtViewCalls > 0) != want {
		t.Errorf("viewForTableColumn calls = %d, view-based expected: %v", vtViewCalls, want)
	}
}

// GUI test (needs GOWT_GUI_TEST=1, not GOWT_VIEW_TABLE=0): accessibility press toggles a check box, a truncated cell has a tooltip.
// MouseUp on the check box needs a real click: not covered here.
func TestViewTableCheckAccessTooltip(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" || !viewTableEnabled {
		t.Skip("set GOWT_GUI_TEST=1 and unset GOWT_VIEW_TABLE=0")
	}
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		d := NewDisplay()
		defer d.Dispose()
		shell := NewShell()
		shell.SetSize(300, 200)
		table := NewTable(shell, CHECK|FULL_SELECTION)
		table.SetBounds(0, 0, 280, 180)
		NewTableColumn(table, NONE).SetWidth(30)
		NewTableItem(table, NONE).SetTexts([]string{"a rather long cell text that does not fit"})
		shell.Open()
		for end := time.Now().Add(time.Second); time.Now().Before(end); {
			if !d.ReadAndDispatch() {
				time.Sleep(10 * time.Millisecond)
			}
		}
		tv := castcocoaNSViewTococoaNSTableView(table.View).Id
		chk := cocoa.Msg(tv, "viewAtColumn:row:makeIfNecessary:", 0, 0, 1)
		txt := cocoa.Msg(tv, "viewAtColumn:row:makeIfNecessary:", 1, 0, 1)
		if chk == 0 || txt == 0 {
			t.Fatal("no cell views")
		}
		if cocoa.Msg(chk, "accessibilityPerformPress") != 1 || !table.GetItem(0).GetChecked() {
			t.Error("accessibilityPerformPress must check the item")
		}
		table.SetEnabled(false)
		table.GetItem(0).SetChecked(false)
		if cocoa.Msg(chk, "accessibilityPerformPress") != 0 || table.GetItem(0).GetChecked() {
			t.Error("accessibilityPerformPress must not toggle a disabled table")
		}
		table.SetEnabled(true)
		if cocoa.Msg(vtToolTip([]int64{txt, 0, 0, 0, 0, 0}), "length") == 0 {
			t.Error("truncated cell must have a tooltip")
		}
	}
	<-done
}

// GUI test (needs GOWT_GUI_TEST=1): the cells of a Table, Tree and List end a long text with an ellipsis, not clipped.
func TestViewTableCellEllipsis(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" || !viewTableEnabled {
		t.Skip("set GOWT_GUI_TEST=1 and unset GOWT_VIEW_TABLE=0")
	}
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		d := NewDisplay()
		defer d.Dispose()
		shell := NewShell()
		shell.SetSize(600, 300)
		long := "a rather long cell text that does not fit"
		table := NewTable(shell, NONE)
		table.SetBounds(0, 0, 180, 120)
		NewTableColumn(table, NONE).SetWidth(30)
		NewTableItem(table, NONE).SetTexts([]string{long})
		tree := NewTree(shell, NONE)
		tree.SetBounds(200, 0, 180, 120)
		NewTreeColumn(tree, NONE).SetWidth(30)
		NewTreeItem(tree, NONE).SetTexts([]string{long})
		list := NewList(shell, V_SCROLL)
		list.SetBounds(400, 0, 180, 120)
		list.SetItems([]string{long})
		shell.Open()
		for end := time.Now().Add(time.Second); time.Now().Before(end); {
			if !d.ReadAndDispatch() {
				time.Sleep(10 * time.Millisecond)
			}
		}
		for name, cell := range map[string]int64{
			"table": table.dataCell.Id, "tree": tree.dataCell.Id, "list": list.column.DataCell().Id,
		} {
			if got := cocoa.Msg(cell, "lineBreakMode"); got != int64(cocoa.OSNSLineBreakByTruncatingTail) {
				t.Errorf("%s: line break mode %d, want truncating tail", name, got)
			}
		}
	}
	<-done
}
