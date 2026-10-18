package gowt

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

var mainq = make(chan func())

// Tests run on their own goroutines; AppKit needs the main thread, which init pinned to this one.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func onMain(f func()) {
	ok := make(chan struct{})
	mainq <- func() { defer close(ok); f() }
	<-ok
}

// Opens real windows: run only where a display is available.
func TestRun(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var clicks int
	var changed string
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("t")
			var txt *TextView
			var btn *ButtonView
			w.Add(Grid(2, Padding(5),
				Text(Cell(GridCell{GrowX: true, Align: AlignFill}), Ref(&txt)),
				Button("go", OnClick(func() { clicks++ }), Ref(&btn)),
			))
			txt.OnChange(func(s string) { changed = s })
			w.Show()
			txt.SetText("abc")
			btn.Unwrap().NotifyListeners(swt.Selection, swt.NewEvent())
			app.After(10*time.Millisecond, func() { w.Close() })
		})
	})
	if err != nil || clicks != 1 || changed != "abc" {
		t.Fatalf("err=%v clicks=%d changed=%q", err, clicks, changed)
	}
}

func TestRunReturnsSWTError(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() { err = Run(func(app *App) { swt.Error(swt.ERROR_INVALID_ARGUMENT) }) })
	if err == nil {
		t.Fatal("want error")
	}
}

// Display.Close from Async disposes the display inside ReadAndDispatch, as the macOS Quit item does.
func TestRunDisplayDisposedInLoop(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.Window("t").Show()
			app.Clipboard()
			app.Async(func() { app.display.Close() })
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func guiRun(t *testing.T, setup func(app *App, w *Window)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("t")
			setup(app, w)
			w.Show()
			app.After(10*time.Millisecond, w.Close)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func notify(w interface{ NotifyListeners(int32, swt.EventLike) }, typ int32) {
	w.NotifyListeners(typ, swt.NewEvent())
}

func TestMultilineText(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		if put[*TextView](w, Text(Multiline())).Unwrap().GetStyle()&swt.MULTI == 0 {
			t.Error("Multiline text lost MULTI")
		}
	})
}

func TestListWidgets(t *testing.T) {
	var combo, list, row, node = -2, -2, -2, ""
	guiRun(t, func(app *App, w *Window) {
		c := put[*ComboView](w, Combo([]string{"a", "b", "c"}, ReadOnly()))
		c.OnSelect(func(i int) { combo = i })
		c.Select(1)
		notify(c.Unwrap(), swt.Selection)

		l := put[*ListView](w, List([]string{"x", "y", "z"}, MultiSelect()))
		l.OnSelect(func(i int) { list = i })
		l.Select(2, 0)
		notify(l.Unwrap(), swt.Selection)
		if got := l.Selected(); len(got) != 2 || l.Len() != 3 {
			t.Errorf("list selection %v len %d", got, l.Len())
		}
		l.Add("w")
		l.Remove(0)
		if items := l.Items(); len(items) != 3 || items[0] != "y" {
			t.Errorf("list items %v", items)
		}

		tb := put[*TableView](w, Table(Check()))
		tb.Column("n", 50)
		tb.Column("s", 50, Right())
		tb.Row("a", "1")
		r := tb.Row("b", "2")
		r.SetTextAt(1, "22")
		r.SetChecked(true)
		r.SetData("tag")
		tb.OnSelect(func(i int) { row = i })
		tb.Select(1)
		notify(tb.Unwrap(), swt.Selection)
		if tb.Len() != 2 || tb.RowAt(1).TextAt(1) != "22" || !tb.RowAt(1).Checked() || tb.RowAt(1).Data() != "tag" {
			t.Errorf("table state: len %d", tb.Len())
		}

		tr := put[*TreeView](w, Tree())
		root := tr.Node("root")
		kid := root.Node("kid")
		kid.Node("leaf")
		root.SetExpanded(true)
		tr.OnSelect(func(n *Node) {
			if n != nil {
				node = n.Text()
			}
		})
		tr.Select(kid)
		notify(tr.Unwrap(), swt.Selection)
		if len(tr.Roots()) != 1 || len(root.Children()) != 1 || !root.Expanded() {
			t.Error("tree shape")
		}
		var expanded string
		tr.OnExpand(func(n *Node) { expanded = n.Text() })
		kid.SetExpanded(true)
		ev := swt.NewEvent()
		ev.Item = &kid.Unwrap().Widget
		tr.Unwrap().NotifyListeners(swt.Expand, ev)
		if expanded != "kid" {
			t.Errorf("expand node %q", expanded)
		}
	})
	if combo != 1 || list != 0 && list != 2 || row != 1 || node != "kid" {
		t.Fatalf("combo=%d list=%d row=%d node=%q", combo, list, row, node)
	}
}

func TestRangeWidgets(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var got int
		sc := put[*ScaleView](w, Scale(10, 50, Value(20)))
		sc.OnChange(func(v int) { got = v })
		sc.SetValue(30)
		notify(sc.Unwrap(), swt.Selection)
		sl := put[*SliderView](w, Slider(0, 200, Value(50)))
		sp := put[*SpinnerView](w, Spinner(-300, -150, Value(-200)))
		pb := put[*ProgressView](w, Progress(10))
		pb.SetValue(4)
		if got != 30 || sl.Value() != 50 || sp.Value() != -200 || pb.Value() != 4 || sc.Unwrap().GetMinimum() != 10 {
			t.Errorf("scale=%d slider=%d spinner=%d progress=%d min=%d", got, sl.Value(), sp.Value(), pb.Value(), sc.Unwrap().GetMinimum())
		}
		dt := put[*DateTimeView](w, DateTime())
		day := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.Local)
		dt.SetValue(day)
		if !dt.Value().Equal(day) {
			t.Errorf("date %v", dt.Value())
		}
		tm := put[*DateTimeView](w, DateTime(AsTime()))
		tm.SetValue(time.Date(2000, 1, 1, 13, 45, 7, 0, time.Local))
		if v := tm.Value(); v.Hour() != 13 || v.Minute() != 45 || v.Second() != 7 {
			t.Errorf("time %v", v)
		}
		var href string
		lk := put[*LinkView](w, Link(`<a href="http://x">x</a>`, OnClick(func(h string) { href = h })))
		ev := swt.NewEvent()
		ev.Text = "http://x"
		lk.Unwrap().NotifyListeners(swt.Selection, ev)
		if href != "http://x" {
			t.Errorf("link href %q", href)
		}
	})
}

// A tab added to a shown Tabs is as wide as its title; cocoa kept the width of an empty title.
func TestTabsAddAfterShow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var first, second, count, sel int32
	var client, page *swt.Rectangle
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("t")
			var tabs *TabsView
			var p *Panel
			w.Add(Tabs(Grow(), Ref(&tabs), Tab("Welcome", Label("hello"))))
			w.SetSize(600, 400)
			w.Show()
			app.After(100*time.Millisecond, func() {
				tabs.Add(Tab("Second session", Ref(&p), Label("page two")))
				tabs.Select(1)
			})
			app.After(400*time.Millisecond, func() {
				f := tabs.Unwrap()
				count, sel = f.GetItemCount(), f.GetSelectionIndex()
				first, second = f.GetItem(0).GetBounds().Width, f.GetItem(1).GetBounds().Width
				client, page = f.GetClientArea(), p.Unwrap().GetBounds()
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || sel != 1 {
		t.Errorf("items %d, selected %d; want 2, 1", count, sel)
	}
	if first <= 0 || second <= first {
		t.Errorf("tab widths %d, %d: the added tab is not wider than \"Welcome\"", first, second)
	}
	// GTK reported a 1x1 client area for the page it had not allocated yet, and the page kept that size.
	if client == nil || client.Width < 100 || page.Width != client.Width || page.Height != client.Height {
		t.Errorf("added page %v, client area %v: the page does not fill the client area", page, client)
	}
}

func TestContainers(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var a, b *Panel
		tabs := put[*TabsView](w, Tabs(Tab("a", Ref(&a)), Tab("b", Ref(&b))))
		var sel int
		tabs.OnSelect(func(i int) { sel = i })
		tabs.Select(1)
		notify(tabs.Unwrap(), swt.Selection)
		if sel != 1 || tabs.Unwrap().GetItemCount() != 2 {
			t.Errorf("tabs sel=%d", sel)
		}
		if a.Unwrap().GetParent() != &tabs.f.Composite || layoutName(a.c) != "VBox" {
			t.Errorf("Tab page: parent %v layout %q", a.Unwrap().GetParent(), layoutName(a.c))
		}
		var first, second *ButtonView
		b.Add(Form(Padding(2),
			Button("x", Anchor(FormCell{Left: Percent(0, 3), Top: Percent(0, 3)}), Ref(&first)),
			Button("y", Anchor(FormCell{Left: Beside(&first, 4), Top: Same(&first, 0)}), Ref(&second)),
		), Flow(Gap(2), Label("r", Size(20, 0))))
		if fd, ok := second.b.GetLayoutData().(*swt.FormData); !ok || fd.Left == nil || fd.Left.Control != &first.b.Control {
			t.Errorf("Beside(&first) did not attach to first: %v", second.b.GetLayoutData())
		}

		b.Dispose()
		if n := tabs.Unwrap().GetItemCount(); n != 1 || a.c.IsDisposed() {
			t.Errorf("a disposed Tab page left %d tabs, want 1", n)
		}

		var c, d *Panel
		ct := put[*CTabsView](w, CTabs(Tab("c", Closable(), Ref(&c))))
		ct.Add(Tab("d", Closable(), Ref(&d)))
		closed := -1
		ct.OnClose(func(i int) bool { closed = i; return false })
		ct.Unwrap().GetItem(1).Dispose() // programmatic dispose does not ask; the veto path needs the UI
		_ = closed
		if !d.c.IsDisposed() || c.c.IsDisposed() {
			t.Error("a closed CTabs tab must dispose its page only")
		}
		c.Dispose()
		if n := ct.Unwrap().GetItemCount(); n != 0 {
			t.Errorf("a disposed CTabs page left %d tabs", n)
		}

		sp := put[*SplitView](w, Split(Vertical(), Label("p"), Label("q")))
		sp.SetWeights(1, 2)
		if ws := sp.Unwrap().GetWeights(); len(ws) != 2 || ws[1] < ws[0]*19/10 { // SWT rescales the weights
			t.Errorf("weights %v", ws)
		}
		sc := put[*ScrollView](w, Scroll(Label("line")))
		if sc.Unwrap().GetMinHeight() <= 0 {
			t.Error("scroll Fit")
		}
		g := put[*GroupView](w, Group("grp", Label("in group")))
		tb := put[*ToolBarView](w, ToolBar())
		clicks := 0
		it := tb.Item("t", func() { clicks++ }, Check())
		tb.Separator()
		it.SetChecked(true)
		notify(it.Unwrap(), swt.Selection)
		cb := put[*CoolBarView](w, CoolBar(ToolBar()))
		if clicks != 1 || !it.Checked() || cb.Unwrap().GetItemCount() != 1 || g.Unwrap().GetText() != "grp" {
			t.Errorf("toolbar clicks=%d", clicks)
		}
	})
}

func TestMenus(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		bar := w.MenuBar()
		var hit []string
		file := bar.Submenu("File")
		file.Item("Open", func() { hit = append(hit, "open") })
		file.Separator()
		chk := file.Item("Check", func() { hit = append(hit, "check") }, Check())
		chk.SetChecked(true)
		notify(chk.Unwrap(), swt.Selection)
		notify(file.Unwrap().GetItem(0), swt.Selection)
		if len(hit) != 2 || !chk.Checked() || bar.Unwrap().GetItemCount() != 1 || file.Unwrap().GetItemCount() != 3 {
			t.Errorf("menu hits %v", hit)
		}
		pop := PopupMenu(put[*TextView](w, Text()))
		pop.Item("x", nil)
		if pop.Unwrap().GetItemCount() != 1 {
			t.Error("popup")
		}
	})
}

func TestImageAndCanvas(t *testing.T) {
	var drawn, imgW int
	var loadErr, badErr error
	guiRun(t, func(app *App, w *Window) {
		im := image.NewRGBA(image.Rect(0, 0, 7, 5))
		var buf bytes.Buffer
		if err := png.Encode(&buf, im); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "i.png")
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		fromFile, err := app.LoadImage(path)
		loadErr = err
		fromBytes, err := app.ImageFrom(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		imgW, _ = fromBytes.Size()
		if _, badErr = app.LoadImage(filepath.Join(t.TempDir(), "nope.png")); badErr == nil {
			t.Error("missing file: want error")
		}
		put[*LabelView](w, Label("l")).SetImage(fromBytes)
		fromFile.Dispose()
		fromFile.Dispose() // idempotent

		cv := put[*CanvasView](w, Canvas(func(g *GC) {
			g.SetColor(RGB{255, 0, 0})
			g.SetFill(RGB{0, 255, 0})
			g.SetLineWidth(2)
			g.Line(0, 0, 5, 5)
			g.Rect(1, 1, 5, 5)
			g.FillRect(1, 1, 5, 5)
			g.Oval(1, 1, 5, 5)
			g.FillOval(1, 1, 5, 5)
			g.Text("hi", 0, 0)
			if tw, th := g.TextSize("hi"); tw <= 0 || th <= 0 {
				t.Errorf("text size %dx%d", tw, th)
			}
			_, plain := g.TextSize("hi")
			g.SetFont(Font{Size: 40, Weight: Semibold})
			g.SetFont(Font{Size: 40, Weight: Semibold})
			if _, big := g.TextSize("hi"); big <= plain || len(g.c.fonts.fonts) != 1 {
				t.Errorf("SetFont: height %d, canvas font %d; %d fonts made", big, plain, len(g.c.fonts.fonts))
			}
			g.SetFont(Font{})
			if _, h := g.TextSize("hi"); h != plain {
				t.Errorf("SetFont(Font{}): height %d, want the canvas's %d", h, plain)
			}
			g.Image(fromBytes, 0, 0)
			drawn++
		}, Grow()))
		w.SetSize(200, 100)
		w.Show()
		cv.Redraw()
		cv.Unwrap().Update()
		for i := 0; i < 20 && drawn == 0; i++ {
			for app.Unwrap().ReadAndDispatch() {
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
	if loadErr != nil || imgW != 7 || drawn == 0 {
		t.Fatalf("load=%v width=%d painted=%d", loadErr, imgW, drawn)
	}
}

// SWT keeps the first of PUSH/CHECK/... (HORIZONTAL/VERTICAL, DROP_DOWN/SIMPLE, LEFT/RIGHT) in its own
// order, so a default OR-ed in by the facade would override the option.
func TestStyleOptionsWin(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		tb := put[*TableView](w, Table())
		for name, c := range map[string]struct{ got, want int32 }{
			"check":    {put[*ButtonView](w, Button("b", Check())).Unwrap().GetStyle(), swt.CHECK},
			"radio":    {put[*ButtonView](w, Button("b", Radio())).Unwrap().GetStyle(), swt.RADIO},
			"simple":   {put[*ComboView](w, Combo(nil, Simple())).Unwrap().GetStyle(), swt.SIMPLE},
			"vscale":   {put[*ScaleView](w, Scale(0, 10, Vertical())).Unwrap().GetStyle(), swt.VERTICAL},
			"right":    {tb.Column("c", 10, Right()).Unwrap().GetStyle(), swt.RIGHT},
			"calendr":  {put[*DateTimeView](w, DateTime(AsCalendar())).Unwrap().GetStyle(), swt.CALENDAR},
			"password": {put[*TextView](w, Text(Password())).Unwrap().GetStyle(), swt.PASSWORD | swt.SINGLE},
		} {
			if c.got&c.want != c.want {
				t.Errorf("%s: style %#x lacks %#x", name, c.got, c.want)
			}
		}
	})
}

func TestTreeColumnsAndCells(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		tr := put[*TreeView](w, Tree(Cell(GridCell{Align: AlignFill, VAlign: AlignFill, GrowX: true, GrowY: true, Width: 300, Height: 200})))
		a := tr.Column("Name", 150)
		b := tr.Column("Size", 100, Right())
		b.SetResizable(false)
		if !tr.Unwrap().GetHeaderVisible() {
			t.Error("first Column must show the header")
		}
		tr.SetSort(a, SortAscending)
		n := tr.Node("root")
		n.SetTextAt(1, "42")
		if n.TextAt(1) != "42" || n.Text() != "root" {
			t.Errorf("cells %q %q", n.Text(), n.TextAt(1))
		}
		// The table goes in before Show, which lays it out; put alone leaves it 0x0, where GTK finds no row.
		tb := put[*TableView](w, Table())
		tb.Column("a", 80)
		tb.Column("b", 80)
		r := tb.Row("x", "y")
		tb.SetSort(nil, SortNone)
		w.Show()
		// GTK allocates the table on an event-loop turn; until then a Table column's GetWidth is 0.
		for i := 0; i < 50 && tb.Unwrap().GetColumnIndex(0).GetWidth() == 0; i++ {
			for app.display.ReadAndDispatch() {
			}
			time.Sleep(time.Millisecond)
		}
		bounds := n.Unwrap().GetBoundsIndex(1)
		got, col := tr.CellAt(int(bounds.X+bounds.Width/2), int(bounds.Y+bounds.Height/2))
		if got == nil || got.Text() != "root" || col != 1 {
			t.Errorf("tree CellAt: %v col %d", got, col)
		}
		if got, col := tr.CellAt(5, 5000); got != nil || col != -1 {
			t.Errorf("tree CellAt outside: %v col %d", got, col)
		}

		rb := r.Unwrap().GetBoundsIndex(1)
		if row, col := tb.CellAt(int(rb.X+rb.Width/2), int(rb.Y+rb.Height/2)); row != 0 || col != 1 {
			t.Errorf("table CellAt: row %d col %d", row, col)
		}
	})
}

// Rows added at creation, before the first layout, must not leave the first row under the header (macOS), and a
// selection made inside Batch may name rows the native view has not been told about yet.
func TestTableTreeFilledBeforeLayout(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var tb *TableView
		var tr *TreeView
		w.Add(HBox(Grow(),
			Table(Grow(), Ref(&tb), Init(func(t *TableView) {
				t.Column("a", 80)
				for i := range 200 {
					t.Row(strconv.Itoa(i))
				}
			})),
			Tree(Grow(), Ref(&tr), Init(func(t *TreeView) {
				t.Column("a", 80)
				for i := range 200 {
					t.Node(strconv.Itoa(i))
				}
			}))))
		w.SetSize(400, 300)
		w.Show()
		for range 20 {
			for app.display.ReadAndDispatch() {
			}
			time.Sleep(time.Millisecond)
		}
		if top := tb.Unwrap().GetTopIndex(); top != 0 {
			t.Errorf("table top index %d", top)
		}
		if top := tr.Unwrap().GetTopItem(); top == nil || top.GetTextIndex(0) != "0" {
			t.Errorf("tree top item %v", top)
		}
		tb.Batch(func() {
			for i := range 100 {
				tb.Row("b" + strconv.Itoa(i))
			}
			tb.Select(299)
			tb.Unwrap().ShowSelection()
		})
		if tb.Len() != 300 || tb.Index() != 299 {
			t.Errorf("after batch: len %d index %d", tb.Len(), tb.Index())
		}
	})
}

func TestEnabledSwitchAndTextLog(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var got []bool
		sw := put[*SwitchView](w, Switch("Wi-Fi", OnChange(func(on bool) { got = append(got, on) })))
		sw.SetChecked(true)
		if !sw.Checked() || len(got) != 0 {
			t.Errorf("SetChecked: checked %v callbacks %v", sw.Checked(), got)
		}
		sw.Unwrap().SetSelection(false)
		notify(sw.Unwrap(), swt.Selection)
		if len(got) != 1 || got[0] {
			t.Errorf("OnChange got %v", got)
		}
		if !sw.Enabled() {
			t.Error("new control must be enabled")
		}
		sw.SetEnabled(false)
		if sw.Enabled() {
			t.Error("SetEnabled(false) not applied")
		}
		lbl := put[*LabelView](w, Label("x", Disabled()))
		if lbl.Enabled() {
			t.Error("Disabled() option not reflected by Enabled")
		}

		tx := put[*TextView](w, Text(Multiline()))
		for i := 0; i < 10; i++ {
			tx.Append("line\n")
		}
		if tx.Lines() != 11 {
			t.Errorf("Text.Lines = %d", tx.Lines())
		}
		tx.TrimHead(3)
		if tx.Lines() != 3 || tx.Text() != "line\nline\n" {
			t.Errorf("Text.TrimHead: %d lines %q", tx.Lines(), tx.Text())
		}
		tx.ScrollToEnd()

		st := put[*StyledTextView](w, StyledText(Scrollbars()))
		for i := 0; i < 10; i++ {
			st.Append("l\n")
		}
		st.TrimHead(3)
		if st.Lines() != 3 || st.Text() != "l\nl\n" {
			t.Errorf("StyledText.TrimHead: %d lines %q", st.Lines(), st.Text())
		}
		st.TrimHead(0)
		if st.Text() != "" {
			t.Errorf("StyledText.TrimHead(0) left %q", st.Text())
		}

		ro := put[*TextView](w, Text(Multiline(), ReadOnly()))
		ro.Append("a\nb\nc\nd")
		ro.TrimHead(2)
		if ro.Text() != "c\nd" || ro.Unwrap().GetEditable() {
			t.Errorf("read-only Text.TrimHead: %q editable %v", ro.Text(), ro.Unwrap().GetEditable())
		}
		ro.TrimHead(0)
		if ro.Text() != "" {
			t.Errorf("Text.TrimHead(0) left %q", ro.Text())
		}
		st.ScrollToEnd()
	})
}

func TestVisibilityFontsAndCanvasMouse(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		a := put[*LabelView](w, Label("a"))
		b := put[*LabelView](w, Label("b"))
		w.Show()
		before := b.l.GetBounds().Y
		a.SetVisible(false)
		if a.Visible() || b.l.GetBounds().Y >= before {
			t.Errorf("hidden label must free its cell: visible %v y %d -> %d", a.Visible(), before, b.l.GetBounds().Y)
		}
		a.SetVisible(true)

		row := put[*Panel](w, Flow())
		r1 := put[*LabelView](row, Label("r1"))
		r2 := put[*LabelView](row, Label("r2"))
		w.Unwrap().LayoutOverload2(true, true)
		x2 := r2.l.GetBounds().X
		r1.SetVisible(false)
		if r2.l.GetBounds().X >= x2 {
			t.Errorf("Flow layout kept the gap of a hidden label: x %d -> %d", x2, r2.l.GetBounds().X)
		}
		b.SetFont(Font{Size: 14})
		b.SetFont(Font{Size: 20, Bold: true})
		if len(ownedFonts) != 1 {
			t.Errorf("ownedFonts %d", len(ownedFonts))
		}
		b.SetFont(Font{Size: 15, Weight: Semibold})
		if len(ownedFonts) != 1 {
			t.Errorf("ownedFonts %d after Weight", len(ownedFonts))
		}
		if fd := b.l.GetFont().GetFontData()[0]; fd.GetHeight() != 15 {
			t.Errorf("Weight font height %d", fd.GetHeight())
		}
		b.SetFont(Font{Size: 15, Weight: Bold})
		if b.l.GetFont().GetFontData()[0].GetStyle()&swt.BOLD == 0 {
			t.Error("Weight Bold is not BOLD in GetFontData")
		}
		b.SetFont(Font{Size: 20, Bold: true})
		b.SetForeground(RGB{200, 0, 0})
		if b.l.GetFont().GetFontData()[0].GetHeight() != 20 {
			t.Error("SetFont size not applied")
		}

		tr := put[*TreeView](w, Tree(NoBorder()))
		tr.ShowRowLines(true)
		if tr.Unwrap().GetStyle()&swt.BORDER != 0 {
			t.Error("NoBorder tree has BORDER")
		}
		put[*TextView](w, Text(Search()))

		cv := put[*CanvasView](w, Canvas(nil))
		var got Mouse
		cv.OnMouseDown(func(m Mouse) { got = m })
		ev := &swt.Event{}
		ev.Button, ev.X, ev.Y, ev.Count = 3, 4, 5, 2
		cv.Unwrap().NotifyListeners(swt.MouseDown, ev)
		if got != (Mouse{X: 4, Y: 5, Button: 3, Count: 2}) {
			t.Errorf("canvas mouse %+v", got)
		}
	})
}

func TestCommonMethodsAndPages(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		btn := put[*ButtonView](w, Button("b"))
		pages := put[*PagesView](w, Pages(GrowX()))
		for _, x := range []Widget{btn, put[*LabelView](w, Label("l")), put[*Panel](w, VBox()), put[*TableView](w, Table()), pages, w} {
			x.SetEnabled(false)
			if x.Enabled() || x.control().GetEnabled() {
				t.Errorf("%T: SetEnabled(false) did not reach the control", x)
			}
			x.SetEnabled(true)
		}
		btn.SetTooltip("tip")
		btn.SetBackground(RGB{10, 20, 30})
		btn.SetForeground(RGB{40, 50, 60})
		if btn.b.GetToolTipText() != "tip" || rgbOf(btn.b.GetBackground().GetRGB()) != (RGB{10, 20, 30}) || rgbOf(btn.b.GetForeground().GetRGB()) != (RGB{40, 50, 60}) {
			t.Error("SetTooltip, SetBackground or SetForeground did not reach the control")
		}
		btn.Focus()

		gone := 0
		l := put[*LabelView](w, Label("x"))
		l.OnDispose(func() { gone++ })
		l.Dispose()
		l.Dispose()
		if gone != 1 || !l.l.IsDisposed() {
			t.Errorf("Dispose: OnDispose ran %d times, disposed %v", gone, l.l.IsDisposed())
		}
		p := put[*Panel](w, VBox())
		p.Add(Label("a"), Label("b"))
		p.Relayout()
		p.Clear()
		if n := len(p.c.GetChildren()); n != 0 {
			t.Errorf("Clear left %d children", n)
		}

		if pages.Current() != -1 {
			t.Errorf("empty Pages: Current %d", pages.Current())
		}
		first, second := put[*Panel](pages, VBox()), put[*Panel](pages, VBox())
		if pages.Current() != 0 || pages.stack.TopControl != first.control() {
			t.Errorf("first page must be shown: Current %d", pages.Current())
		}
		pages.Show(1)
		if pages.Current() != 1 || pages.stack.TopControl != second.control() {
			t.Errorf("Show(1): Current %d", pages.Current())
		}
		// Disposing pages: the shown one hands over to the page before it, or after it when it was first.
		second.Dispose()
		if pages.Current() != 0 || pages.stack.TopControl != first.control() || !first.ctl.GetVisible() {
			t.Errorf("shown last page disposed: Current %d, want the one before it", pages.Current())
		}
		third, fourth := put[*Panel](pages, VBox()), put[*Panel](pages, VBox())
		pages.Show(0)
		first.Dispose()
		if pages.Current() != 0 || pages.stack.TopControl != third.control() || !third.ctl.GetVisible() {
			t.Errorf("shown first page disposed: Current %d, want the next one renumbered to 0", pages.Current())
		}
		fourth.Dispose()
		if pages.Current() != 0 || pages.stack.TopControl != third.control() {
			t.Errorf("hidden page disposed: Current %d, want 0 kept", pages.Current())
		}
		third.Dispose()
		if pages.Current() != -1 || pages.stack.TopControl != nil {
			t.Errorf("last page disposed: Current %d", pages.Current())
		}
		fifth := put[*Panel](pages, VBox())
		if pages.Current() != 0 || pages.stack.TopControl != fifth.control() {
			t.Errorf("a page added while none is shown must be shown: Current %d", pages.Current())
		}
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "Pages.Show(1) with 1 pages") {
					t.Errorf("Show out of range: %v", r)
				}
			}()
			pages.Show(1)
		}()

		tb := put[*TableView](w, Table())
		tb.Batch(func() {
			for i := 0; i < 3; i++ {
				tb.Row("r")
			}
		})
		if tb.Len() != 3 {
			t.Errorf("Batch rows %d", tb.Len())
		}

		file := w.MenuBar().Submenu("&File")
		open := file.Item("&Open", nil, Shortcut("Cmd+O"))
		want := swt.CTRL | 'O'
		if runtime.GOOS == "darwin" {
			want = swt.COMMAND | 'O'
		}
		if open.i.GetAccelerator() != want || !strings.Contains(open.i.GetText(), "\t") {
			t.Errorf("Shortcut: accelerator %#x text %q", open.i.GetAccelerator(), open.i.GetText())
		}
		open.SetText("&Open...")
		if !strings.HasPrefix(open.i.GetText(), "&Open...\t") {
			t.Errorf("SetText dropped the shortcut: %q", open.i.GetText())
		}

		w.SetMinSize(200, 100)
		if m := w.shell.GetMinimumSize(); m.X < 200 || m.Y < 100 {
			t.Errorf("SetMinSize: %v", m)
		}
		if sw, sh := app.ScreenSize(); sw <= 0 || sh <= 0 {
			t.Errorf("ScreenSize %dx%d", sw, sh)
		}
	})
}

func TestEvery(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var n int
	var errs []error
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("t")
			w.Show()
			var stop func()
			stop = app.Every(5*time.Millisecond, func() {
				n++
				if n == 1 {
					panic("tick")
				}
				if n == 3 {
					stop()
				}
			})
			app.After(150*time.Millisecond, w.Close)
		})
	})
	if err != nil || n != 3 || len(errs) != 1 {
		t.Fatalf("err %v, ticks %d, errors %v", err, n, errs)
	}
}

func TestWindowOnKey(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var hits []string
		w.OnKey("Cmd+K", func() { hits = append(hits, "Cmd+K") })
		w.OnKey("Delete", func() { hits = append(hits, "Delete") })
		txt := put[*TextView](w, Text())
		btn := put[*ButtonView](w, Button("b"))
		other := app.Window("other")
		defer other.Unwrap().Dispose()
		elsewhere := put[*ButtonView](other, Button("o"))
		cmd := int32(swt.CTRL)
		if runtime.GOOS == "darwin" {
			cmd = swt.COMMAND
		}
		key := func(c interface{ NotifyListeners(int32, swt.EventLike) }, state, code int32) bool {
			e := swt.NewEvent()
			e.StateMask, e.KeyCode = state, code
			c.NotifyListeners(swt.KeyDown, e)
			return e.Doit
		}
		if key(btn.Unwrap(), cmd, 'k') {
			t.Error("the bound key went on to the button")
		}
		key(txt.Unwrap(), cmd, 'k')
		key(btn.Unwrap(), 0, int32(swt.DEL))
		if !key(txt.Unwrap(), 0, int32(swt.DEL)) {
			t.Error("Delete did not reach the text field")
		}
		key(elsewhere.Unwrap(), cmd, 'k')
		key(btn.Unwrap(), cmd, 'j')
		if want := []string{"Cmd+K", "Cmd+K", "Delete"}; !slices.Equal(hits, want) {
			t.Errorf("hits %v, want %v", hits, want)
		}
	})
}

// A setter that changes a widget's size lays the parents out on the next turn, without Relayout;
// a Wrap label in a column that fills the window takes the lines its text needs.
func TestSetTextLaysOut(t *testing.T) {
	const long = "a long text that needs several lines in a window this narrow, set after the window was shown"
	ran := false
	guiRun(t, func(app *App, w *Window) {
		var l, wrap, below *LabelView
		var b *ButtonView
		w.Add(HBox(Label("short", Ref(&l)), Button("ok", Ref(&b))),
			VBox(GrowX(), Label("", Wrap(), GrowX(), Ref(&wrap)), Label("neighbour", Ref(&below))))
		w.SetSize(400, 300)
		w.Show()
		app.Flush()
		lineH := below.l.GetSize().Y
		l.SetText("a longer text, set after Show")
		b.SetText("a wider button")
		wrap.SetText(long)
		app.Async(func() {
			ran = true
			for _, c := range []*swt.Control{&l.l.Control, &b.b.Control} {
				if got, want := c.GetSize().X, c.ComputeSize(swt.DEFAULT, swt.DEFAULT).X; got != want {
					t.Errorf("%v: width %d after SetText, want %d", c, got, want)
				}
			}
			wb := wrap.l.GetBounds()
			if wb.Height < 2*lineH || below.l.GetLocation().Y < wb.Y+wb.Height {
				t.Errorf("wrap label %v, line %d, neighbour at %d", wb, lineH, below.l.GetLocation().Y)
			}
			l.SetText("a longer text, set after Show")
			if len(app.relayouts) != 0 {
				t.Errorf("the same text queued a relayout: %d", len(app.relayouts))
			}
		})
	})
	if !ran {
		t.Error("the check did not run")
	}
}
