package gowt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// dndFixture is a window with two trees and a canvas; log collects what the handlers saw.
type dndFixture struct {
	w              *Window
	src, dst       *TreeView
	src2, dst2     *ListView
	urlSrc, urlDst *CanvasView
	cv             *CanvasView
	files          []string   // dragged by src2
	cancelNext     bool       // the next OnStart cancels its drag
	mu             sync.Mutex // the pointer goroutine reads log while a drag loop holds the UI thread
	log            []string
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

func newDndFixture(app *App, files []string) *dndFixture {
	f := &dndFixture{files: files}
	f.w = app.Window("dnd",
		HBox(Gap(10),
			Tree(Ref(&f.src), Size(110, 120), Init(func(t *TreeView) { t.Node("apple"); t.Node("pear") })),
			Tree(Ref(&f.dst), Size(110, 120), Init(func(t *TreeView) { t.Node("basket"); t.Node("crate") })),
			Canvas(func(g *GC) {}, Ref(&f.cv), Size(110, 120)),
			List([]string{"one.txt", "two.txt"}, Ref(&f.src2), Size(110, 120)),
			List(nil, Ref(&f.dst2), Size(110, 120)),
			Canvas(func(g *GC) {}, Ref(&f.urlSrc), Size(160, 120)),
			Canvas(func(g *GC) {}, Ref(&f.urlDst), Size(160, 120))))
	note := func(format string, a ...any) {
		f.mu.Lock()
		f.log = append(f.log, fmt.Sprintf(format, a...))
		f.mu.Unlock()
	}
	start := func() bool {
		f.mu.Lock()
		cancel := f.cancelNext
		f.cancelNext = false
		f.mu.Unlock()
		note("start")
		return !cancel
	}
	DragFrom(f.src, Drag{
		Ops: OpCopy | OpMove,
		Text: func() string {
			if s := f.src.Selected(); len(s) > 0 {
				return s[0].Text()
			}
			return ""
		},
		OnStart: start,
		OnDone:  func(op Op) { note("done %d", op) },
	})
	DragFrom(f.src2, Drag{
		Ops:     OpCopy | OpMove,
		Files:   func() []string { return f.files },
		OnStart: start,
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
	DropOn(f.dst2, Drop{
		Ops:   OpCopy | OpMove,
		Over:  func(x, y int) bool { note("files over"); return true },
		Files: func(paths []string, op Op) { note("files %d %d", len(paths), op) },
	})
	urlTransfer := swt.URLTransferGetInstance()
	urlDrag := swt.NewDragSource(f.urlSrc.Unwrap(), swt.DNDDROP_COPY)
	urlDrag.SetTransfer([]*swt.Transfer{urlTransfer.AsTransfer()})
	urlDrag.AddDragListener(urlSourceListener{url: "https://example.com/report", note: note})
	urlDrop := swt.NewDropTarget(f.urlDst.Unwrap(), swt.DNDDROP_COPY|swt.DNDDROP_DEFAULT)
	urlDrop.SetTransfer([]*swt.Transfer{urlTransfer.AsTransfer()})
	urlDrop.AddDropListener(&urlTargetListener{DropTargetAdapter: swt.NewDropTargetAdapter(), note: note})
	f.w.SetSize(980, 300)
	f.w.Unwrap().SetLocation(0, 0)
	return f
}

// urlSourceListener drags one URL through the plain swt URLTransfer; the facade
// does not wrap URL drags.
type urlSourceListener struct {
	url  string
	note func(string, ...any)
}

func (l urlSourceListener) DragStart(e *swt.DragSourceEvent)    { l.note("url start") }
func (l urlSourceListener) DragSetData(e *swt.DragSourceEvent)  { e.Data = l.url }
func (l urlSourceListener) DragFinished(e *swt.DragSourceEvent) { l.note("url done %d", e.Detail) }

// urlTargetListener accepts URL drops; chooseOp maps the plain-drag default to
// a real operation, as the facade listener does.
type urlTargetListener struct {
	*swt.DropTargetAdapter
	note func(string, ...any)
}

func (l *urlTargetListener) DragEnter(e *swt.DropTargetEvent) {
	l.note("url enter %d %d", e.Detail, e.Operations)
	urlChooseOp(e)
}
func (l *urlTargetListener) DragOperationChanged(e *swt.DropTargetEvent) { urlChooseOp(e) }

func urlChooseOp(e *swt.DropTargetEvent) {
	if e.Detail != swt.DNDDROP_DEFAULT {
		return
	}
	e.Detail = swt.DNDDROP_NONE
	for _, op := range []int32{swt.DNDDROP_MOVE, swt.DNDDROP_COPY, swt.DNDDROP_LINK} {
		if e.Operations&op != 0 {
			e.Detail = op
			return
		}
	}
}

func (l *urlTargetListener) DragOver(e *swt.DropTargetEvent) { l.note("url over") }
func (l *urlTargetListener) DropAccept(e *swt.DropTargetEvent) {
	l.note("url accept %d", e.Detail)
}

func (l *urlTargetListener) Drop(e *swt.DropTargetEvent) {
	if s, ok := e.Data.(string); ok {
		l.note("url %q %d", s, Op(e.Detail))
	}
}

// row is the middle of the i-th root row of tv in display points.
func row(tv *TreeView, i int32) swt.Point {
	b := tv.Unwrap().GetItem(i).GetBounds()
	p := tv.Unwrap().ToDisplay(b.X+b.Width/2, b.Y+b.Height/2)
	return swt.Point{X: p.X, Y: p.Y}
}

// dragPointer presses at from, moves in steps to to, hovers there and releases. It waits, 3 s at most
// each, for the drag to start before it leaves the source and, when over is not empty, for that log
// line at the target and always hovers ten moves before mouse-up (rejected drops and cancelled
// drags log no over line). It waits, again 3 s at most, for the end: on a slow machine the first drag
// session of a process started late, and the pointer had reached the target before it, so no DragOver ran.
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
	min := int32(10) // let the native target finish DragEnter before mouse-up
	wait := time.Now().Add(3 * time.Second)
	for i := int32(0); ok && (i < min || over != "" && !f.seen(n, over) && time.Now().Before(wait)); i++ {
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
			f = newDndFixture(app, nil)
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

// center is the middle of w in display points; call it on the UI thread.
func center(w Widget) swt.Point {
	b := w.control().GetBounds()
	p := w.control().ToDisplay(b.Width/2, b.Height/2)
	return swt.Point{X: p.X, Y: p.Y}
}

// TestDndRealMouseScenarios drives the OS pointer through the native drag session for the cases
// TestDndRealMouse leaves out: a files drop, a drop the target refuses (the empty area of the tree),
// a drag the source cancels in OnStart, and a URL drop through the plain swt URLTransfer (the facade
// does not wrap URL drags; URLTransfer is disabled upstream on macOS, so that scenario runs on
// Linux and Windows only).
func TestDndRealMouseScenarios(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "one.txt"), filepath.Join(dir, "two.txt")}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("dnd"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var f *dndFixture
	var skip string
	var cuts = []int{0}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			f = newDndFixture(app, files)
			f.w.Show()
			app.After(500*time.Millisecond, func() {
				if skip = postBlocked(app); skip != "" {
					f.w.Close()
					return
				}
				fileStart := f.src2.Unwrap().ToDisplay(20, 10)
				filesFrom := pointer(app, swt.Point{X: fileStart.X, Y: fileStart.Y})
				filesTo := pointer(app, center(f.dst2))
				rejectedFrom := pointer(app, row(f.src, 0))
				emptyP := f.dst.Unwrap().ToDisplay(80, 112)
				rejectedTo := pointer(app, swt.Point{X: emptyP.X, Y: emptyP.Y})
				cancelFrom, cancelTo := pointer(app, row(f.src, 1)), pointer(app, row(f.dst, 0))
				urlFrom, urlTo := pointer(app, center(f.urlSrc)), pointer(app, center(f.urlDst))
				screenW, screenH := app.ScreenSize()
				t.Logf("dnd window=%v screen=%dx%d url=%v->%v", f.w.Unwrap().GetBounds(), screenW, screenH, urlFrom, urlTo)
				go func() {
					defer app.Async(func() { f.w.Close() })
					mark := func() {
						app.Sync(func() { cuts = append(cuts, len(f.log)) })
					}
					ok := dragPointer(app, f, filesFrom, filesTo, "files over")
					mark()
					if runtime.GOOS != "darwin" {
						ok = ok && dragPointer(app, f, urlFrom, urlTo, "url over")
					}
					mark()
					ok = ok && dragPointer(app, f, rejectedFrom, rejectedTo, "")
					mark()
					app.Sync(func() { f.cancelNext = true })
					ok = ok && dragPointer(app, f, cancelFrom, cancelTo, "")
					mark()
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
	slice := func(i int) []string {
		from, to := cuts[i], len(f.log)
		if i+1 < len(cuts) {
			to = cuts[i+1]
		}
		return f.log[from:to]
	}
	t.Logf("log: %q", f.log)
	checkDrag(t, "files drop", slice(0), "files over", "files 2 ")
	checkRejected(t, slice(2))
	checkCancelled(t, slice(3))
	if runtime.GOOS != "darwin" {
		checkURLDrop(t, slice(1))
	} else {
		t.Log("url drop: not run, URLTransfer is disabled upstream on macOS (deprecated pasteboard type, eclipse-platform/eclipse.platform.swt#2669)")
	}
}

// checkRejected wants no drop line and a none operation: the target refused the whole drag.
func checkRejected(t *testing.T, lines []string) {
	t.Helper()
	for _, l := range lines {
		if strings.HasPrefix(l, "over ") || strings.HasPrefix(l, "drop ") {
			t.Errorf("rejected drop: log %q, want no over/drop line", lines)
			return
		}
	}
	if len(lines) == 0 || lines[len(lines)-1] != "done 0" {
		t.Errorf("rejected drop: log %q, want done 0 last", lines)
	}
}

// checkCancelled wants the start vetoed by the source: no over or drop line, and only the start
// and, when the platform sends DragFinished for a vetoed drag, a none operation.
func checkCancelled(t *testing.T, lines []string) {
	t.Helper()
	for _, l := range lines {
		if strings.HasPrefix(l, "over ") || strings.HasPrefix(l, "drop ") || l != "start" && l != "done 0" {
			t.Errorf("cancelled drag: log %q, want only start and maybe done 0", lines)
			return
		}
	}
	if len(lines) == 0 || lines[0] != "start" {
		t.Errorf("cancelled drag: log %q, want start first", lines)
	}
}

// checkURLDrop wants the over line, then the URL with a real operation and url done with the same
// operation last.
func checkURLDrop(t *testing.T, lines []string) {
	t.Helper()
	if !slices.Contains(lines, "url over") {
		t.Errorf("url drop: log %q, want url over", lines)
		return
	}
	d, done := lines[len(lines)-2], lines[len(lines)-1]
	rest, ok := strings.CutPrefix(d, "url ")
	if !ok {
		t.Errorf("url drop: drop %q, want it to start with url", d)
		return
	}
	url, op, _ := strings.Cut(rest, " ")
	if url != `"https://example.com/report"` || op == "0" || done != "url done "+op {
		t.Errorf("url drop: drop %q, done %q, want the url with a real operation and url done <op>", d, done)
	}
}
