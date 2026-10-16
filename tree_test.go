package gowt

import (
	"errors"
	"strings"
	"testing"

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
	var txt *TextView
	var btn *ButtonView
	b := &ButtonView{}
	if want := Ref(&txt).(ref)(b); want != "*gowt.TextView" || txt != nil {
		t.Errorf("Ref to the wrong type: want %q, txt %v", want, txt)
	}
	if want := Ref(&btn).(ref)(b); want != "" || btn != b {
		t.Errorf("Ref: want %q, btn %v", want, btn)
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
				Button("Add", nil),
			),
			Grid(2, GrowX(), Ref(&form),
				Label("Name"),
				Text(GrowX(), Ref(&name)),
			),
			Group("g", Ref(&grp), Label("in")),
			Tabs(Tab("t", Ref(&tab), Label("page"))),
			HBox(GrowX(), Spacer(), Button("Save", nil, Ref(&save))),
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
		bad := put[*LabelView](w, Label("x", Password(), Gap(2), Button("b", nil), Shortcut("Cmd+B")))
		var wrong *TextView
		put[*ButtonView](w, Button("b", nil, Ref(&wrong)))
		put[*Panel](w, Flow(Label("f", GrowX())))
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
