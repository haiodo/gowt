package gowt

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/swt"
)

func TestIDKey(t *testing.T) {
	if qatree.IDKey != "gowt.id" {
		t.Errorf("qatree reads %q, ID writes gowt.id", qatree.IDKey)
	}
}

// qaFind returns the one node s picks in the dump.
func qaFind(t *testing.T, app *App, s qatree.Selector) qatree.Node {
	t.Helper()
	ns, err := qatree.Find(app.Unwrap(), s)
	if err != nil || len(ns) != 1 {
		t.Errorf("find %+v: %d nodes, err %v", s, len(ns), err)
		return qatree.Node{}
	}
	return ns[0]
}

func qaAct(t *testing.T, app *App, s qatree.Selector, action, args string) {
	t.Helper()
	if err := qatree.Act(app.Unwrap(), s, action, json.RawMessage(args)); err != nil {
		t.Errorf("%s %+v: %v", action, s, err)
	}
}

// qaRefuse expects Act to fail with a message holding want.
func qaRefuse(t *testing.T, app *App, s qatree.Selector, action, want string) {
	t.Helper()
	err := qatree.Act(app.Unwrap(), s, action, nil)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s %+v: err %v, want one with %q", action, s, err, want)
	}
}

func TestQATree(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var clicks, off, ghost, changes, agree, list, combo, row, node, tab, inTab, menu, dlg int
	var typed string
	var ghostBtn *ButtonView
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("qa")
			w.Add(VBox(
				Button("go", ID("go"), OnClick(func() { clicks++ })),
				Button("off", ID("off"), Disabled(), OnClick(func() { off++ })),
				Button("ghost", ID("ghost"), Ref(&ghostBtn), OnClick(func() { ghost++ })),
				Text(ID("name"), Hint("your name"), OnChange(func(s string) { changes++; typed = s })),
				Button("agree", Check(), ID("agree"), OnClick(func() { agree++ })),
				List([]string{"a", "b", "c"}, ID("list"), OnSelect(func(i int) { list = i })),
				Combo([]string{"x", "y"}, ReadOnly(), ID("combo"), OnSelect(func(i int) { combo = i })),
				Table(ID("tbl"), OnSelect(func(i int) { row = i }), Init(func(t *TableView) {
					t.Column("Name", 80)
					t.Column("CPU", 50)
					t.Row("alpha", "1")
					t.Row("beta", "2")
				})),
				Tree(ID("tree"), OnSelect(func(n *Node) { node++ }), Init(func(t *TreeView) {
					r := t.Node("root")
					r.Node("kid")
					r.SetExpanded(true)
				})),
				Tabs(ID("tabs"), OnSelect(func(i int) { tab = i }),
					Tab("one", Label("page one")),
					Tab("two", Button("in two", ID("in2"), OnClick(func() { inTab++ })))),
			))
			bar := w.MenuBar()
			file := bar.Submenu("File")
			file.Item("Open", func() { menu++ })
			w.Show()
			func() { // dump
				d := qatree.Dump(app.Unwrap())
				if len(d) != 1 || d[0].Class != "Shell" || d[0].Text != "qa" || d[0].Path != "0" || !d[0].Visible {
					t.Errorf("shells %+v", d)
					return
				}
				if g := qaFind(t, app, qatree.Selector{ID: "go"}); g.Class != "Button" || g.Text != "go" || !g.Enabled || !g.Visible || g.Bounds[2] == 0 {
					t.Errorf("go %+v", g)
				}
				if o := qaFind(t, app, qatree.Selector{ID: "off"}); o.Enabled {
					t.Error("off is enabled")
				}
				if n := qaFind(t, app, qatree.Selector{ID: "name"}); n.Class != "Text" || n.Text != "your name" || n.Value != "" {
					t.Errorf("name %+v", n)
				}
				if a := qaFind(t, app, qatree.Selector{ID: "agree"}); a.Checked == nil || *a.Checked {
					t.Errorf("agree %+v", a)
				}
				if l := qaFind(t, app, qatree.Selector{ID: "list"}); len(l.Children) != 3 || l.Children[1].Text != "b" {
					t.Errorf("list %+v", l)
				}
				if c := qaFind(t, app, qatree.Selector{ID: "combo"}); len(c.Children) != 2 {
					t.Errorf("combo %+v", c)
				}
				tb := qaFind(t, app, qatree.Selector{ID: "tbl"})
				if len(tb.Children) != 2 || tb.Children[1].Text != "beta" || tb.Children[1].Value != "2" || tb.Children[1].Class != "TableItem" {
					t.Errorf("tbl %+v", tb)
				}
				tr := qaFind(t, app, qatree.Selector{ID: "tree"})
				if len(tr.Children) != 1 || tr.Children[0].Text != "root" || len(tr.Children[0].Children) != 1 || tr.Children[0].Children[0].Text != "kid" {
					t.Errorf("tree %+v", tr)
				}
				tabs := qaFind(t, app, qatree.Selector{ID: "tabs"})
				if len(tabs.Selected) != 1 || tabs.Selected[0] != 0 || tabs.Children[0].Class != "TabItem" || tabs.Children[1].Text != "two" {
					t.Errorf("tabs %+v", tabs)
				}
				if p := qaFind(t, app, qatree.Selector{ID: "in2"}); p.Visible {
					t.Error("button on the unselected tab is visible")
				}
				if mi := qaFind(t, app, qatree.Selector{Class: "MenuItem", Text: "Open"}); !mi.Enabled || !strings.Contains(mi.Path, "MenuItem") {
					t.Errorf("menu item %+v", mi)
				}
				if f := qaFind(t, app, qatree.Selector{Text: "go", Class: "Button"}); f.ID != "go" {
					t.Errorf("by text %+v", f)
				}
				if f := qaFind(t, app, qatree.Selector{Text: "~ur na"}); f.ID != "name" {
					t.Errorf("by part of the hint %+v", f)
				}
				if f := qaFind(t, app, qatree.Selector{Class: "Button", Text: "agree", Within: &qatree.Selector{Class: "Shell"}}); f.ID != "agree" {
					t.Errorf("within %+v", f)
				}
			}()

			func() { // find
				if ns, err := qatree.Find(app.Unwrap(), qatree.Selector{ID: "nothing"}); err != nil || len(ns) != 0 {
					t.Errorf("no match: %v %v", ns, err)
				}
				_, err := qatree.Find(app.Unwrap(), qatree.Selector{Text: "go", Within: &qatree.Selector{Class: "Button"}})
				if err == nil || !strings.Contains(err.Error(), "Button.") {
					t.Errorf("within of several: %v", err)
				}
				if ns, _ := qatree.Find(app.Unwrap(), qatree.Selector{Class: "TableItem"}); len(ns) != 2 {
					t.Errorf("TableItem: %d", len(ns))
				}
			}()

			func() { // refused
				qaRefuse(t, app, qatree.Selector{Class: "Button"}, "click", "controls match")
				qaRefuse(t, app, qatree.Selector{ID: "nothing"}, "click", "0 controls match")
				qaRefuse(t, app, qatree.Selector{ID: "off"}, "click", "disabled")
			}()

			func() { // click
				qaAct(t, app, qatree.Selector{ID: "go"}, "click", "")
				qaAct(t, app, qatree.Selector{Class: "Button", Text: "go"}, "click", "")
				if clicks != 2 || off != 0 {
					t.Errorf("clicks=%d off=%d", clicks, off)
				}
				qaAct(t, app, qatree.Selector{Text: "Open", Class: "MenuItem"}, "click", "")
				if menu != 1 {
					t.Errorf("menu=%d", menu)
				}
			}()

			func() { // type
				qaAct(t, app, qatree.Selector{ID: "name"}, "type", `{"text":"hi"}`)
				if typed != "hi" || changes < 2 {
					t.Errorf("typed=%q changes=%d", typed, changes)
				}
				if n := qaFind(t, app, qatree.Selector{ID: "name"}); n.Value != "hi" {
					t.Errorf("name %+v", n)
				}
				qaAct(t, app, qatree.Selector{ID: "name"}, "type", `{"text":"yo"}`)
				if typed != "yo" {
					t.Errorf("retyped %q", typed)
				}
			}()

			func() { // check
				qaAct(t, app, qatree.Selector{ID: "agree"}, "check", `{"on":true}`)
				qaAct(t, app, qatree.Selector{ID: "agree"}, "check", `{"on":true}`)
				if a := qaFind(t, app, qatree.Selector{ID: "agree"}); agree != 1 || a.Checked == nil || !*a.Checked {
					t.Errorf("agree=%d %+v", agree, a)
				}
				qaAct(t, app, qatree.Selector{ID: "agree"}, "check", `{"on":false}`)
				if agree != 2 {
					t.Errorf("agree=%d", agree)
				}
			}()

			func() { // select
				qaAct(t, app, qatree.Selector{ID: "list"}, "select", `{"index":1}`)
				if l := qaFind(t, app, qatree.Selector{ID: "list"}); list != 1 || len(l.Selected) != 1 || l.Selected[0] != 1 {
					t.Errorf("list=%d %+v", list, l)
				}
				qaAct(t, app, qatree.Selector{ID: "list"}, "select", `{"text":"c"}`)
				qaAct(t, app, qatree.Selector{ID: "combo"}, "select", `{"text":"y"}`)
				qaAct(t, app, qatree.Selector{ID: "tbl"}, "select", `{"index":1}`)
				if list != 2 || combo != 1 || row != 1 {
					t.Errorf("list=%d combo=%d row=%d", list, combo, row)
				}
				qaAct(t, app, qatree.Selector{Class: "TableItem", Text: "alpha"}, "click", "")
				if row != 0 {
					t.Errorf("row=%d", row)
				}
				qaAct(t, app, qatree.Selector{Class: "TreeItem", Text: "kid"}, "click", "")
				if tr := qaFind(t, app, qatree.Selector{ID: "tree"}); node != 1 || len(tr.Selected) != 1 || tr.Selected[0] != 1 {
					t.Errorf("node=%d %+v", node, tr)
				}
				qaRefuse(t, app, qatree.Selector{ID: "in2"}, "click", "not visible")
				qaAct(t, app, qatree.Selector{ID: "tabs"}, "select", `{"text":"two"}`)
				if tab != 1 {
					t.Errorf("tab=%d", tab)
				}
				qaAct(t, app, qatree.Selector{ID: "in2"}, "click", "")
				if inTab != 1 {
					t.Errorf("inTab=%d", inTab)
				}
				if p := qaFind(t, app, qatree.Selector{Class: "Label", Text: "page one"}); p.Visible {
					t.Error("page one still visible")
				}
			}()

			func() { // modal
				s := swt.NewShellParentStyle(w.shell, swt.DIALOG_TRIM|swt.APPLICATION_MODAL)
				s.SetText("dialog")
				s.SetLayout(newBox(false, 0, 0))
				dw := &Window{panel: newPanel(&s.Composite), shell: s}
				dw.Add(Button("dlg ok", ID("dlgok"), OnClick(func() { dlg++ })))
				dw.Show()
				qaRefuse(t, app, qatree.Selector{ID: "go"}, "click", "modal")
				qaAct(t, app, qatree.Selector{ID: "dlgok"}, "click", "")
				if dlg != 1 || clicks != 2 {
					t.Errorf("dlg=%d clicks=%d", dlg, clicks)
				}
				if d := qatree.Dump(app.Unwrap()); len(d) != 2 {
					t.Errorf("shells %d", len(d))
				}
				dw.Close()
				qaAct(t, app, qatree.Selector{ID: "go"}, "click", "")
				if clicks != 3 {
					t.Errorf("clicks=%d", clicks)
				}
			}()

			func() { // hidden
				ghostBtn.SetVisible(false)
				qaRefuse(t, app, qatree.Selector{ID: "ghost"}, "click", "not visible")
				if ghost != 0 {
					t.Errorf("ghost=%d", ghost)
				}
			}()
			w.Close()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQATreeActions(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var activated, expanded, collapsed, keys, accel, closed int
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("qa actions")
			w.Add(VBox(
				Button("go", ID("go")),
				List([]string{"a", "b"}, ID("list"), OnActivate(func(i int) { activated += i + 1 })),
				Tree(ID("tree"), Init(func(t *TreeView) {
					t.Node("lazy").Node("loading")
					t.OnExpand(func(n *Node) { expanded++; n.Node("real") })
					t.OnCollapse(func(*Node) { collapsed++ })
				})),
			))
			w.OnKey("Cmd+K", func() { keys++ })
			w.OnClose(func() bool { closed++; return closed > 1 })
			w.MenuBar().Submenu("File").Item("Run", func() { accel++ }, Shortcut("Cmd+R"))
			w.Show()

			qaAct(t, app, qatree.Selector{Class: "ListItem", Text: "b"}, "activate", "")
			qaAct(t, app, qatree.Selector{Class: "TreeItem", Text: "lazy"}, "expand", "")
			qaFind(t, app, qatree.Selector{Class: "TreeItem", Text: "real"})
			qaAct(t, app, qatree.Selector{Class: "TreeItem", Text: "lazy"}, "collapse", "")
			qaRefuse(t, app, qatree.Selector{Class: "TreeItem", Text: "real"}, "activate", "not visible")
			qaAct(t, app, qatree.Selector{ID: "go"}, "key", `{"keys":"Mod+K"}`)
			qaAct(t, app, qatree.Selector{ID: "go"}, "key", `{"keys":"Mod+R"}`)
			qaAct(t, app, qatree.Selector{ID: "go"}, "focus", "")
			qaAct(t, app, qatree.Selector{}, "key", `{"keys":"Mod+K"}`) // to the focus control
			if activated != 2 || expanded != 1 || collapsed != 1 || keys != 2 || accel != 1 {
				t.Errorf("activated=%d expanded=%d collapsed=%d keys=%d accel=%d", activated, expanded, collapsed, keys, accel)
			}
			qaAct(t, app, qatree.Selector{Class: "Shell"}, "close", "")
			if closed != 1 || len(qatree.Dump(app.Unwrap())) != 1 {
				t.Errorf("the vetoed close: closed=%d", closed)
			}
			qaAct(t, app, qatree.Selector{Class: "Shell"}, "close", "")
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
