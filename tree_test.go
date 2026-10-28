package gowt

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// put creates the widget o describes in h and returns it, for tests that use it right away.
func put[T Widget](h host, o Option) T {
	w := o.(*node).build(h)
	h.added(w)
	return w.(T)
}

func TestStyleNames(t *testing.T) {
	for want, o := range map[string]Option{
		"Check": Check(), "Radio": Radio(), "Multiline": Multiline(), "Password": Password(),
		"ReadOnly": ReadOnly(), "Border": Border(), "CellSelect": CellSelect(), "NoBorder": NoBorder(),
		"Search": Search(), "MultiSelect": MultiSelect(), "Simple": Simple(), "Vertical": Vertical(),
		"Smooth": Smooth(), "Indeterminate": Indeterminate(), "Flat": Flat(), "Wrap": Wrap(),
		"Bottom": Bottom(), "Closable": Closable(), "DropDown": DropDown(), "Right": Right(),
		"Center": Center(), "AsTime": AsTime(), "AsCalendar": AsCalendar(), "Scrollbars": Scrollbars(),
		"Toggle": Toggle(),
	} {
		if got := o.(style).id.String(); got != want {
			t.Errorf("%s reports itself as %s", want, got)
		}
	}
}

func TestResolve(t *testing.T) {
	got := resolve(swt.BORDER|swt.FULL_SELECTION, []Option{Check(), GrowX(), NoBorder(), Label("x")})
	if got != swt.CHECK|swt.FULL_SELECTION {
		t.Errorf("resolve: %#x", got)
	}
}

func TestSpacingOf(t *testing.T) {
	if g, p := spacingOf([]Option{Label("x")}, 7); g != Spacing.SM || p != 7 {
		t.Errorf("defaults: gap %d padding %d", g, p)
	}
	if g, p := spacingOf([]Option{Gap(0), Padding(3), Gap(5)}, 7); g != 5 || p != 3 {
		t.Errorf("given: gap %d padding %d", g, p)
	}
}

func TestRefType(t *testing.T) {
	var btn *ButtonView
	var got *ButtonView
	b := &ButtonView{}
	if err := Ref(&btn).(hook)(b); err != nil || btn != b {
		t.Errorf("Ref: %v, btn %v", err, btn)
	}
	if err := Init(func(v *ButtonView) { got = v }).(hook)(b); err != nil || got != b {
		t.Errorf("Init: %v, got %v", err, got)
	}
}

func TestBesideBeforeSibling(t *testing.T) {
	var first *ButtonView
	_, err := FormCell{Left: Beside(&first, 4)}.data()
	if !errors.Is(err, errNoSibling) {
		t.Errorf("Beside an unset Ref: %v", err)
	}
}

// Opens real windows: the tree, Ref, Add on a live panel, default gaps and the error path.
func TestTree(t *testing.T) {
	var errs []string
	guiRun(t, func(app *App, w *Window) {
		app.OnError(func(err error) { errs = append(errs, err.Error()) })
		var name *TextView
		var head, form *Panel
		var save *ButtonView
		var tab *Panel
		var grp *GroupView
		w.Add(
			HBox(GrowX(), Ref(&head),
				Label("Contacts"),
				Spacer(),
				Button("Add"),
			),
			Grid(2, GrowX(), Ref(&form),
				Label("Name"),
				Text(GrowX(), Ref(&name)),
			),
			Group("g", Ref(&grp), Label("in")),
			Tabs(Tab("t", Ref(&tab), Label("page"))),
			HBox(GrowX(), Spacer(), Button("Save", Ref(&save))),
		)
		if name == nil || save == nil || head == nil || tab == nil {
			t.Fatalf("Ref: name %v save %v head %v tab %v", name, save, head, tab)
		}
		if name.t.GetParent() != form.c || form.c.GetParent() != w.c {
			t.Error("Text must be a child of the Grid, the Grid of the window")
		}
		for _, c := range []struct {
			p      *swt.Composite
			layout string
			gap    int32
			pad    int32
		}{
			{w.c, "VBox", int32(Spacing.SM), int32(Spacing.PageMargin)},
			{head.c, "HBox", int32(Spacing.SM), 0},
			{form.c, "Grid", int32(Spacing.SM), 0},
			{&grp.g.Composite, "VBox", int32(Spacing.SM), int32(Spacing.SM)},
			{tab.c, "VBox", int32(Spacing.SM), int32(Spacing.PageMargin)},
		} {
			g := c.p.GetLayout().Impl().(swt.GridLayoutLike).AsGridLayout()
			if layoutName(c.p) != c.layout || g.VerticalSpacing != c.gap || g.HorizontalSpacing != c.gap || g.MarginWidth != c.pad {
				t.Errorf("%s: layout %q gap %d/%d padding %d", c.layout, layoutName(c.p), g.HorizontalSpacing, g.VerticalSpacing, g.MarginWidth)
			}
		}
		spacer := head.c.GetChildren()[1].GetLayoutData().(*swt.GridData)
		if !spacer.GrabExcessHorizontalSpace || spacer.GrabExcessVerticalSpace {
			t.Errorf("Spacer in an HBox must take the width: %+v", spacer)
		}
		col := put[*Panel](w, VBox(Spacer()))
		if d := col.c.GetChildren()[0].GetLayoutData().(*swt.GridData); !d.GrabExcessVerticalSpace || d.GrabExcessHorizontalSpace {
			t.Errorf("Spacer in a VBox must take the height: %+v", d)
		}

		w.Show()
		hb := head.c.GetLayout().Impl().(*boxLayout)
		if hb.NumColumns != 3 {
			t.Errorf("HBox of 3 children: %d columns", hb.NumColumns)
		}
		var late *LabelView
		head.Add(Label("late", Ref(&late)))
		if late == nil || hb.NumColumns != 4 || late.l.GetBounds().Width == 0 {
			t.Errorf("Add on a shown HBox: columns %d, bounds %v", hb.NumColumns, late.l.GetBounds())
		}
		late.SetVisible(false)
		if hb.NumColumns != 3 {
			t.Errorf("hidden child keeps its column: %d", hb.NumColumns)
		}
		late.Dispose()
		head.Clear()
		head.Relayout()
		if hb.NumColumns != 1 {
			t.Errorf("empty HBox: %d columns", hb.NumColumns)
		}

		pages := put[*PagesView](w, Pages(Label("one"), VBox(Label("two"))))
		if pages.Current() != 0 || len(pages.c.GetChildren()) != 2 {
			t.Errorf("Pages: current %d of %d", pages.Current(), len(pages.c.GetChildren()))
		}

		errs = nil
		bad := put[*LabelView](w, Label("x", Password(), Gap(2), Button("b"), Shortcut("Cmd+B")))
		var wrong *TextView
		put[*ButtonView](w, Button("b", Ref(&wrong)))
		put[*Panel](w, Flow(Label("f", GrowX())))
		// Panes and pages fill their part: Grow there is accepted and changes nothing, Size is not.
		put[*SplitView](w, Split(Group("g", Grow()), Tabs(GrowY(), Tab("t", Grow())), CTabs(Tab("c", GrowX())), Label("s", Size(10, 10))))
		put[*PagesView](w, Pages(Label("p", Grow())))
		put[*Panel](w, Fill(Label("f", GrowX())))
		put[*Panel](w, VBox(Tab("lost")))
		put[*TabsView](w, Tabs(Label("stray")))
		w.Add(GrowX())
		put[*TableView](w, Table()).Column("c", 10, Tooltip("tip"))
		want := []string{
			"gowt: Password does not apply to Label",
			"gowt: Gap and Padding apply to containers, not to Label",
			"gowt: Label takes no children, Button is dropped",
			"gowt: Shortcut applies to Menu.Item, not to Label",
			"gowt: Ref to a *gowt.TextView given to Button",
			"gowt: GrowX on Label needs a VBox, HBox or Grid parent, not Flow",
			"gowt: Size on Label needs a VBox, HBox, Grid, Flow or Form parent, not Split",
			"gowt: Tab needs a Tabs or CTabs parent, not Panel",
			"gowt: Tabs takes Tab children, not Label",
			"gowt: Add on Window takes widgets; give options where the widget is created",
			"gowt: Table.Column takes only its style options",
		}
		if bad == nil || bad.l.IsDisposed() || strings.Join(errs, "\n") != strings.Join(want, "\n") {
			t.Errorf("error path:\n%s\nwant:\n%s", strings.Join(errs, "\n"), strings.Join(want, "\n"))
		}
	})
}

// Event, initial-value and binding options on real widgets, and the errors of the misfits.
func TestEventOptions(t *testing.T) {
	var errs []string
	guiRun(t, func(app *App, w *Window) {
		app.OnError(func(err error) { errs = append(errs, err.Error()) })
		var clicks, picked, activated int
		var on bool
		var text string
		var href string
		var node string
		name := rx.New("ann")
		count := rx.New(3)
		agree := rx.New(true)
		title := rx.Compute(func() string { return "Hello, " + name.Get() })
		btn := put[*ButtonView](w, Button("b", OnClick(func() { clicks++ }), BindEnabled(agree)))
		chk := put[*ButtonView](w, Button("c", Check(), Checked(true), OnChange(func(v bool) { on = v })))
		sw := put[*SwitchView](w, Switch("s", Bind(agree)))
		tx := put[*TextView](w, Text(Hint("Name"), Bind(name), OnChange(func(s string) { text = s }), OnActivate(func() { activated++ })))
		lb := put[*LabelView](w, Label("", BindText(title), BindVisible(agree)))
		lk := put[*LinkView](w, Link(`<a href="x">x</a>`, OnClick(func(h string) { href = h })))
		sp := put[*SpinnerView](w, Spinner(0, 10, Bind(count)))
		sc := put[*ScaleView](w, Scale(0, 100, Value(40)))
		cb := put[*ComboView](w, Combo([]string{"a", "b"}, Selected(1), OnSelect(func(i int) { picked = i })))
		dt := put[*DateTimeView](w, DateTime(Value(time.Date(2024, 2, 29, 0, 0, 0, 0, time.Local))))
		st := put[*StyledTextView](w, StyledText(Value("styled")))
		ls := put[*ListView](w, List([]string{"x", "y"}, Bind(count)))
		tr := put[*TreeView](w, Tree(OnSelect(func(n *Node) { node = n.Text() })))
		if !chk.Checked() || tx.t.GetMessage() != "Name" || tx.Text() != "ann" || lb.l.GetText() != "Hello, ann" ||
			sp.Value() != 3 || sc.Value() != 40 || cb.Index() != 1 || !sw.Checked() || dt.Value().Day() != 29 || st.Text() != "styled" {
			t.Errorf("initial values: checked %v hint %q text %q label %q spinner %d scale %d combo %d switch %v",
				chk.Checked(), tx.t.GetMessage(), tx.Text(), lb.l.GetText(), sp.Value(), sc.Value(), cb.Index(), sw.Checked())
		}
		notify(btn.b, swt.Selection)
		chk.b.SetSelection(false)
		notify(chk.b, swt.Selection)
		tx.SetText("bob")
		notify(tx.t, swt.DefaultSelection)
		ev := swt.NewEvent()
		ev.Text = "x"
		lk.l.NotifyListeners(swt.Selection, ev)
		notify(cb.c, swt.Selection)
		n := tr.Node("n")
		tr.Select(n)
		notify(tr.t, swt.Selection)
		rx.Flush()
		if clicks != 1 || on || text != "bob" || activated != 1 || href != "x" || picked != 1 || node != "n" ||
			name.Peek() != "bob" || lb.l.GetText() != "Hello, bob" {
			t.Errorf("events: clicks %d on %v text %q activated %d href %q picked %d node %q state %q label %q",
				clicks, on, text, activated, href, picked, node, name.Peek(), lb.l.GetText())
		}
		sw.b.SetSelection(false)
		notify(sw.b, swt.Selection)
		rx.Flush()
		if agree.Peek() || btn.Enabled() || lb.Visible() {
			t.Errorf("Switch Bind: state %v, enabled %v, visible %v", agree.Peek(), btn.Enabled(), lb.Visible())
		}
		count.Set(1)
		rx.Flush()
		if sp.Value() != 1 || ls.Index() != 1 {
			t.Errorf("Set did not reach Spinner %d and List %d", sp.Value(), ls.Index())
		}

		// Hooks run after the children: Selected finds the tabs, Init the panes.
		tabs := put[*TabsView](w, Tabs(Selected(1), Tab("a"), Tab("b")))
		var weights []int32
		put[*SplitView](w, Split(Init(func(s *SplitView) { s.SetWeights(1, 3); weights = s.s.GetWeights() }), Label("l"), Label("r")))
		if tabs.Index() != 1 || len(weights) != 2 {
			t.Errorf("hooks before children: tab %d, weights %v", tabs.Index(), weights)
		}

		errs = nil
		put[*LabelView](w, Label("x", OnClick(func() {}), Hint("h"), Bind(name), Value(1)))
		put[*TextView](w, Text(OnChange(func(int) {}), BindText(title), Value(2)))
		want := []string{
			"gowt: OnClick(func()) does not apply to Label",
			"gowt: Hint applies to Text, not to Label",
			"gowt: Bind(*rx.State[string]) applies to Text and Combo, not to Label",
			"gowt: Value(int) does not apply to Label",
			"gowt: OnChange(func(int)) does not apply to Text",
			"gowt: BindText applies to Label, Button and Link, not to Text; a Text takes Bind",
			"gowt: Value(int) does not apply to Text",
		}
		if strings.Join(errs, "\n") != strings.Join(want, "\n") {
			t.Errorf("misfits:\n%s\nwant:\n%s", strings.Join(errs, "\n"), strings.Join(want, "\n"))
		}
	})
}

// BenchmarkTree200 creates and disposes 200 widgets: 50 HBox rows of a Label, a Text and a Button.
func BenchmarkTree200(b *testing.B) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		b.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		_ = Run(func(app *App) {
			w := app.Window("bench")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var p *Panel
				rows := make([]Option, 0, 51)
				rows = append(rows, Ref(&p))
				for r := 0; r < 50; r++ {
					rows = append(rows, HBox(Label("Name"), Text(GrowX()), Button("OK")))
				}
				w.Add(VBox(rows...))
				p.Dispose()
			}
			b.StopTimer()
			w.Close()
		})
	})
}

// with applies a hook option to an existing widget, as finish would at creation.
func with(w Widget, o Option) {
	if err := o.(hook)(w); err != nil {
		panic(err)
	}
}
