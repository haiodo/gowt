package gowt

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// dndFixture is a window with two trees and a canvas; log collects what the handlers saw.
type dndFixture struct {
	w        *Window
	src, dst *TreeView
	cv       *CanvasView
	mu       sync.Mutex // the pointer goroutine reads log while a drag loop holds the UI thread
	log      []string
}

// seen reports whether a line from index from on starts with prefix.
func (f *dndFixture) seen(from int, prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.ContainsFunc(f.log[min(from, len(f.log)):], func(l string) bool { return strings.HasPrefix(l, prefix) })
}

func (f *dndFixture) lines() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.log)
}

func newDndFixture(app *App) *dndFixture {
	f := &dndFixture{}
	f.w = app.Window("dnd",
		HBox(Gap(10),
			Tree(Ref(&f.src), Size(200, 120), Init(func(t *TreeView) { t.Node("apple"); t.Node("pear") })),
			Tree(Ref(&f.dst), Size(200, 120), Init(func(t *TreeView) { t.Node("basket"); t.Node("crate") })),
			Canvas(func(g *GC) {}, Ref(&f.cv), Size(200, 120))))
	note := func(format string, a ...any) {
		f.mu.Lock()
		f.log = append(f.log, fmt.Sprintf(format, a...))
		f.mu.Unlock()
	}
	DragFrom(f.src, Drag{
		Ops: OpCopy | OpMove,
		Text: func() string {
			if s := f.src.Selected(); len(s) > 0 {
				return s[0].Text()
			}
			return ""
		},
		OnStart: func() bool { note("start"); return true },
		OnDone:  func(op Op) { note("done %d", op) },
	})
	DropOn(f.dst, Drop{
		Ops: OpCopy | OpMove,
		Over: func(x, y int) bool {
			n, _ := f.dst.CellAt(x, y)
			if n != nil {
				note("over %s", n.Text())
			}
			return n != nil
		},
		Text: func(s string, op Op) { note("drop %q %d", s, op) },
	})
	DropOn(f.cv, Drop{
		Over: func(x, y int) bool { note("canvas over"); return true },
		Text: func(s string, op Op) { note("canvas drop %q %d", s, op) },
	})
	f.w.SetSize(700, 300)
	return f
}

// row is the middle of the i-th root row of tv in display points.
func row(tv *TreeView, i int32) swt.Point {
	b := tv.Unwrap().GetItem(i).GetBounds()
	p := tv.Unwrap().ToDisplay(b.X+b.Width/2, b.Y+b.Height/2)
	return swt.Point{X: p.X, Y: p.Y}
}

// dragPointer presses at from, moves in steps to to, hovers there and releases. It waits, 3 s at most
// each, for the drag to start before it leaves the source, for over at the target and for the end: on a
// slow machine the first drag session of a process started late, and the pointer had reached the target
// before it, so no DragOver ran.
func dragPointer(app *App, f *dndFixture, from, to swt.Point, over string) bool {
	n := f.lines()
	at := func(i int32) (int32, int32) { return from.X + (to.X-from.X)*i/12, from.Y + (to.Y-from.Y)*i/12 }
	ok := postMouse(app, swt.MouseMove, from.X, from.Y)
	time.Sleep(150 * time.Millisecond)
	ok = ok && postMouse(app, swt.MouseDown, from.X, from.Y)
	time.Sleep(150 * time.Millisecond)
	for i := int32(1); i <= 12; i++ {
		x, y := at(i)
		ok = ok && postMouse(app, swt.MouseMove, x, y)
		time.Sleep(30 * time.Millisecond)
		if i != 2 {
			continue
		}
		for j, wait := int32(0), time.Now().Add(3*time.Second); ok && !f.seen(n, "start") && time.Now().Before(wait); j++ {
			ok = postMouse(app, swt.MouseMove, x, y+j%2)
			time.Sleep(60 * time.Millisecond)
		}
	}
	wait := time.Now().Add(3 * time.Second)
	for i := int32(0); ok && (i < 4 || !f.seen(n, over) && time.Now().Before(wait)); i++ {
		ok = postMouse(app, swt.MouseMove, to.X+i%2, to.Y)
		time.Sleep(60 * time.Millisecond)
	}
	ok = ok && postMouse(app, swt.MouseUp, to.X, to.Y)
	for wait := time.Now().Add(3 * time.Second); ok && !f.seen(n, "done ") && time.Now().Before(wait); {
		time.Sleep(50 * time.Millisecond)
	}
	return ok
}

// TestDndRealMouse drags with the OS pointer (xdotool, SendInput, CGEventPost), so the native drag
// session runs: Tree to Tree and Tree to a canvas. The pointer is posted from a goroutine because the
// drag loop of Windows and macOS blocks the UI thread until the button is released.
func TestDndRealMouse(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var f *dndFixture
	var skip string
	var cut int
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			f = newDndFixture(app)
			f.w.Show()
			app.After(500*time.Millisecond, func() {
				if skip = postBlocked(app); skip != "" {
					f.w.Close()
					return
				}
				c := f.cv.Unwrap().ToDisplay(100, 60)
				a1, a2 := pointer(app, row(f.src, 0)), pointer(app, row(f.dst, 1))
				b1, b2 := pointer(app, row(f.src, 1)), pointer(app, swt.Point{X: c.X, Y: c.Y})
				go func() {
					defer app.Async(func() { f.w.Close() })
					ok := dragPointer(app, f, a1, a2, "over crate")
					app.Sync(func() { cut = len(f.log) })
					ok = ok && dragPointer(app, f, b1, b2, "canvas over")
					if !ok {
						app.Sync(func() { skip = "the mouse events could not be posted" })
					}
				}()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if skip != "" {
		t.Skip(skip)
	}
	t.Logf("log: %q", f.log)
	checkDrag(t, "tree to tree", f.log[:cut], "over crate", `drop "apple" `)
	checkDrag(t, "tree to canvas", f.log[cut:], "canvas over", `canvas drop "pear" `)
}

// checkDrag wants start first, the over line somewhere, then the drop with a real operation and
// done with the same one last.
func checkDrag(t *testing.T, name string, lines []string, over, drop string) {
	t.Helper()
	if len(lines) < 4 || lines[0] != "start" || !slices.Contains(lines, over) {
		t.Errorf("%s: log %q, want start first and %q", name, lines, over)
		return
	}
	d, done := lines[len(lines)-2], lines[len(lines)-1]
	op, ok := strings.CutPrefix(d, drop)
	if !ok || op == "0" || done != "done "+op {
		t.Errorf("%s: drop %q, done %q, want %q<op> and done <op>", name, d, done, drop)
	}
}
