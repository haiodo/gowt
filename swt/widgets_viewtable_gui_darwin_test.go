package swt

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

func init() { runtime.LockOSThread() }

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

// Run with GOWT_GUI_TEST=1, and again with GOWT_VIEW_TABLE=1: the native table must be view-based exactly when the flag is set.
func TestViewTableEngaged(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	want := os.Getenv("GOWT_VIEW_TABLE") == "1"
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
	shell.SetSize(600, 400)

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

	shell.Open()
	for end := time.Now().Add(1500 * time.Millisecond); time.Now().Before(end); {
		if !d.ReadAndDispatch() {
			time.Sleep(10 * time.Millisecond)
		}
	}

	tv := castcocoaNSViewTococoaNSTableView(table.View).Id
	ov := castcocoaNSViewTococoaNSOutlineView(tree.View).Id
	for _, c := range []struct {
		name string
		view int64
		col  int64
	}{{"table column 0", tv, 0}, {"table column 1", tv, 1}, {"tree column 0", ov, 0}} {
		got := cocoa.Msg(c.view, "viewAtColumn:row:makeIfNecessary:", c.col, 0, 0) != 0
		if got != want {
			t.Errorf("%s: viewAtColumn non-nil = %v, want %v", c.name, got, want)
		}
	}
	if (vtViewCalls > 0) != want {
		t.Errorf("viewForTableColumn calls = %d, view-based expected: %v", vtViewCalls, want)
	}
}
