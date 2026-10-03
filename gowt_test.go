package gowt

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
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
			w.SetLayout(Grid{Columns: 2, Margin: 5})
			txt := w.Text(Cell(GridCell{GrowX: true, Align: AlignFill}))
			txt.OnChange(func(s string) { changed = s })
			btn := w.Button("go", func() { clicks++ })
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
		if w.Text(Multiline()).Unwrap().GetStyle()&swt.MULTI == 0 {
			t.Error("Multiline text lost MULTI")
		}
	})
}

func TestListWidgets(t *testing.T) {
	var combo, list, row, node = -2, -2, -2, ""
	guiRun(t, func(app *App, w *Window) {
		w.SetLayout(Grid{Columns: 1})
		c := w.Combo([]string{"a", "b", "c"}, ReadOnly())
		c.OnSelect(func(i int) { combo = i })
		c.Select(1)
		notify(c.Unwrap(), swt.Selection)

		l := w.List([]string{"x", "y", "z"}, MultiSelect())
		l.OnSelect(func(i int) { list = i })
		l.SetSelection(2, 0)
		notify(l.Unwrap(), swt.Selection)
		if got := l.Selected(); len(got) != 2 || l.Len() != 3 {
			t.Errorf("list selection %v len %d", got, l.Len())
		}
		l.Add("w")
		l.Remove(0)
		if items := l.Items(); len(items) != 3 || items[0] != "y" {
			t.Errorf("list items %v", items)
		}

		tb := w.Table(Check())
		tb.Column("n", 50)
		tb.Column("s", 50, Right())
		tb.Row("a", "1")
		r := tb.Row("b", "2")
		r.SetText(1, "22")
		r.SetChecked(true)
		r.SetData("tag")
		tb.OnSelect(func(i int) { row = i })
		tb.SetSelection(1)
		notify(tb.Unwrap(), swt.Selection)
		if tb.Len() != 2 || tb.RowAt(1).Text(1) != "22" || !tb.RowAt(1).Checked() || tb.RowAt(1).Data() != "tag" {
			t.Errorf("table state: len %d", tb.Len())
		}

		tr := w.Tree()
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
		sc := w.Scale(10, 50, 20)
		sc.OnChange(func(v int) { got = v })
		sc.SetValue(30)
		notify(sc.Unwrap(), swt.Selection)
		sl := w.Slider(0, 200, 50)
		sp := w.Spinner(-300, -150, -200)
		pb := w.Progress(10)
		pb.SetValue(4)
		if got != 30 || sl.Value() != 50 || sp.Value() != -200 || pb.Value() != 4 || sc.Unwrap().GetMinimum() != 10 {
			t.Errorf("scale=%d slider=%d spinner=%d progress=%d min=%d", got, sl.Value(), sp.Value(), pb.Value(), sc.Unwrap().GetMinimum())
		}
		dt := w.DateTime()
		day := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.Local)
		dt.SetValue(day)
		if !dt.Value().Equal(day) {
			t.Errorf("date %v", dt.Value())
		}
		tm := w.DateTime(AsTime())
		tm.SetValue(time.Date(2000, 1, 1, 13, 45, 7, 0, time.Local))
		if v := tm.Value(); v.Hour() != 13 || v.Minute() != 45 || v.Second() != 7 {
			t.Errorf("time %v", v)
		}
		var href string
		lk := w.Link(`<a href="http://x">x</a>`, func(h string) { href = h })
		ev := swt.NewEvent()
		ev.Text = "http://x"
		lk.Unwrap().NotifyListeners(swt.Selection, ev)
		if href != "http://x" {
			t.Errorf("link href %q", href)
		}
	})
}

func TestContainers(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		w.SetLayout(Fill{})
		tabs := w.Tabs()
		a, b := tabs.Tab("a"), tabs.Tab("b")
		var sel int
		tabs.OnSelect(func(i int) { sel = i })
		tabs.Select(1)
		notify(tabs.Unwrap(), swt.Selection)
		if sel != 1 || tabs.Unwrap().GetItemCount() != 2 {
			t.Errorf("tabs sel=%d", sel)
		}
		a.SetLayout(Stack{})
		l1, l2 := a.Label("1"), a.Label("2")
		a.ShowTop(l2)
		if a.stack.TopControl != l2.control() || a.stack.TopControl == l1.control() {
			t.Error("stack top")
		}
		b.SetLayout(Form{Margin: 2})
		first := b.Button("x", nil, Anchor(FormCell{Left: Percent(0, 3), Top: Percent(0, 3)}))
		b.Button("y", nil, Anchor(FormCell{Left: Beside(first, 4), Top: Same(first, 0)}))
		b.Panel().SetLayout(Row{Wrap: true, Spacing: 2}).Label("r", InRow(RowCell{Width: 20}))

		ct := w.CTabs()
		ct.Tab("c", Closable())
		ct.Tab("d", Closable())
		closed := -1
		ct.OnClose(func(i int) bool { closed = i; return false })
		ct.Unwrap().GetItem(1).Dispose() // programmatic dispose does not ask; the veto path needs the UI
		_ = closed

		sp := w.Split(Vertical())
		sp.Label("p")
		sp.Label("q")
		sp.SetWeights(1, 2)
		if ws := sp.Unwrap().GetWeights(); len(ws) != 2 || ws[1] < ws[0]*19/10 { // SWT rescales the weights
			t.Errorf("weights %v", ws)
		}
		sc := w.Scroll()
		sc.Content().SetLayout(Row{Vertical: true})
		sc.Content().Label("line")
		sc.Fit()
		if sc.Unwrap().GetMinHeight() <= 0 {
			t.Error("scroll Fit")
		}
		g := w.Group("grp")
		g.Label("in group")
		tb := w.ToolBar()
		clicks := 0
		it := tb.Item("t", func() { clicks++ }, Check())
		tb.Separator()
		it.SetChecked(true)
		notify(it.Unwrap(), swt.Selection)
		cb := w.CoolBar()
		cb.Add(cb.ToolBar())
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
		pop := PopupMenu(w.Text())
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
		w.Label("l").SetImage(fromBytes)
		fromFile.Dispose()
		fromFile.Dispose() // idempotent

		w.SetLayout(Fill{})
		cv := w.Canvas(func(g *GC) {
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
			g.Image(fromBytes, 0, 0)
			drawn++
		})
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
		tb := w.Table()
		for name, c := range map[string]struct{ got, want int32 }{
			"check":    {w.Button("b", nil, Check()).Unwrap().GetStyle(), swt.CHECK},
			"radio":    {w.Button("b", nil, Radio()).Unwrap().GetStyle(), swt.RADIO},
			"simple":   {w.Combo(nil, Simple()).Unwrap().GetStyle(), swt.SIMPLE},
			"vscale":   {w.Scale(0, 10, 0, Vertical()).Unwrap().GetStyle(), swt.VERTICAL},
			"right":    {tb.Column("c", 10, Right()).Unwrap().GetStyle(), swt.RIGHT},
			"calendr":  {w.DateTime(AsCalendar()).Unwrap().GetStyle(), swt.CALENDAR},
			"password": {w.Text(Password()).Unwrap().GetStyle(), swt.PASSWORD | swt.SINGLE},
		} {
			if c.got&c.want != c.want {
				t.Errorf("%s: style %#x lacks %#x", name, c.got, c.want)
			}
		}
	})
}
