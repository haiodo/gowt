package gowt

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// dndFixture is a window with two trees and a canvas; log collects what the handlers saw.
type dndFixture struct {
	w        *Window
	src, dst *TreeView
	cv       *CanvasView
	log      []string
}

func newDndFixture(app *App) *dndFixture {
	f := &dndFixture{}
	f.w = app.Window("dnd",
		HBox(Gap(10),
			Tree(Ref(&f.src), Size(200, 120), Init(func(t *TreeView) { t.Node("apple"); t.Node("pear") })),
			Tree(Ref(&f.dst), Size(200, 120), Init(func(t *TreeView) { t.Node("basket"); t.Node("crate") })),
			Canvas(func(g *GC) {}, Ref(&f.cv), Size(200, 120))))
	note := func(format string, a ...any) { f.log = append(f.log, fmt.Sprintf(format, a...)) }
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

// dragPointer presses at from, moves in steps to to, hovers there and releases.
func dragPointer(app *App, from, to swt.Point) bool {
	ok := postMouse(app, swt.MouseMove, from.X, from.Y)
	time.Sleep(150 * time.Millisecond)
	ok = ok && postMouse(app, swt.MouseDown, from.X, from.Y)
	time.Sleep(150 * time.Millisecond)
	for i := int32(1); i <= 12; i++ {
		ok = ok && postMouse(app, swt.MouseMove, from.X+(to.X-from.X)*i/12, from.Y+(to.Y-from.Y)*i/12)
		time.Sleep(30 * time.Millisecond)
	}
	for i := int32(0); i < 4; i++ {
		ok = ok && postMouse(app, swt.MouseMove, to.X+i%2, to.Y)
		time.Sleep(60 * time.Millisecond)
	}
	ok = ok && postMouse(app, swt.MouseUp, to.X, to.Y)
	time.Sleep(600 * time.Millisecond)
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
					ok := dragPointer(app, a1, a2)
					app.Sync(func() { cut = len(f.log) })
					ok = ok && dragPointer(app, b1, b2)
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
