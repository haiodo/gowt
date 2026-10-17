package gowt

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/haiodo/gowt/rx"
)

type proc struct {
	pid  int
	name string
	cpu  string
}

func procKey(p proc) int { return p.pid }

func fillProc(r *TableRow, p proc) {
	r.SetText(0, p.name)
	r.SetText(1, p.cpu)
}

func tableTexts(t *TableView) string {
	var s []string
	for i := range t.Len() {
		r := t.RowAt(i)
		s = append(s, r.Text(0)+":"+r.Text(1))
	}
	return strings.Join(s, " ")
}

func tableSelection(t *TableView) string {
	var s []string
	for _, i := range t.Selected() {
		s = append(s, t.RowAt(i).Text(0))
	}
	return strings.Join(s, " ")
}

func TestRowsTable(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		list := rx.NewFunc([]proc{{1, "a", "0"}, {2, "b", "0"}, {3, "c", "0"}}, nil)
		// Rows before Init: the rows are still filled after the columns exist.
		tb := put[*TableView](w, Table(MultiSelect(), Rows(list, procKey, fillProc),
			Init(func(t *TableView) { t.Column("Name", 80); t.Column("CPU", 50) })))
		if got := tableTexts(tb); got != "a:0 b:0 c:0" {
			t.Fatalf("first fill %q", got)
		}
		first := tb.RowAt(0).Unwrap()
		tb.SetSelection(1, 2)
		list.Set([]proc{{1, "a", "5"}, {4, "d", "1"}, {3, "c", "0"}, {2, "b", "0"}})
		rx.Flush()
		if got := tableTexts(tb); got != "a:5 d:1 c:0 b:0" {
			t.Errorf("after update %q", got)
		}
		if got := tableSelection(tb); got != "c b" {
			t.Errorf("selection %q, want c b", got)
		}
		if tb.RowAt(0).Unwrap() != first {
			t.Error("the row of a changed element was made again")
		}
		list.Set([]proc{{1, "a", "5"}, {3, "c", "0"}, {2, "b", "0"}}) // d goes; GTK would select a neighbour
		tb.SetSelection(1)
		list.Set([]proc{{1, "a", "5"}, {2, "b", "0"}})
		rx.Flush()
		if got := tableSelection(tb); got != "" {
			t.Errorf("selection %q after its row went", got)
		}
		list.Set(nil)
		rx.Flush()
		if tb.Len() != 0 {
			t.Errorf("%d rows after an empty list", tb.Len())
		}
	})
}

type tnodeItem struct {
	key, text string
}

func TestNodesTree(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		kids := map[string][]tnodeItem{
			"g": {{"g/a", "a"}, {"g/b", "b"}},
			"h": {{"h/c", "c"}},
		}
		roots := rx.NewFunc([]tnodeItem{{"g", "g"}, {"h", "h"}, {"d", "d"}}, nil)
		created := 0
		tr := put[*TreeView](w, Tree(MultiSelect(), Nodes(roots, func(n tnodeItem) string { return n.key },
			func(n tnodeItem) []tnodeItem { return kids[n.key] },
			func(n *Node, it tnodeItem, isNew bool) {
				n.SetText(it.text)
				if isNew {
					created++
					n.SetExpanded(true)
				}
			})))
		w.Show()
		if created != 6 || len(tr.Roots()) != 3 || !tr.Roots()[0].Expanded() {
			t.Fatalf("first fill: %d created, %d roots", created, len(tr.Roots()))
		}
		tr.Roots()[1].SetExpanded(false)
		tr.Select(tr.Roots()[0].Children()[1])
		kids["g"] = []tnodeItem{{"g/b", "b2"}, {"g/a", "a"}, {"g/x", "x"}}
		roots.Set([]tnodeItem{{"d", "d"}, {"h", "h"}, {"g", "g"}})
		rx.Flush()
		var got []string
		for _, r := range tr.Roots() {
			s := r.Text()
			if r.Expanded() {
				s += "!"
			}
			for _, c := range r.Children() {
				s += " " + c.Text()
			}
			got = append(got, s)
		}
		if want := "d|h c|g! b2 a x"; strings.Join(got, "|") != want {
			t.Errorf("tree %q, want %q", strings.Join(got, "|"), want)
		}
		if sel := tr.Selected(); len(sel) != 1 || sel[0].Text() != "b2" {
			t.Errorf("selection %v", nodeTexts(sel))
		}
	})
}

func nodeTexts(ns []*Node) []string {
	var s []string
	for _, n := range ns {
		s = append(s, n.Text())
	}
	return s
}

func TestItemsListCombo(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		names := rx.NewFunc([]string{"a", "b", "c"}, nil)
		l := put[*ListView](w, List(nil, Items(names)))
		c := put[*ComboView](w, Combo(nil, ReadOnly(), Items(names)))
		l.SetSelection(2)
		c.Select(2)
		names.Set([]string{"x", "c", "a"})
		rx.Flush()
		if !slices.Equal(l.Items(), []string{"x", "c", "a"}) || !slices.Equal(c.Items(), []string{"x", "c", "a"}) {
			t.Errorf("items %v %v", l.Items(), c.Items())
		}
		if l.Index() != 1 || c.Index() != 1 {
			t.Errorf("selection list %d combo %d, want 1", l.Index(), c.Index())
		}
	})
}

func TestRowsMisfit(t *testing.T) {
	var errs []string
	guiRun(t, func(app *App, w *Window) {
		app.OnError(func(err error) { errs = append(errs, err.Error()) })
		list := rx.NewFunc([]proc(nil), nil)
		put[*TreeView](w, Tree(Rows(list, procKey, fillProc)))
		put[*LabelView](w, Label("", Items(rx.NewFunc([]string(nil), nil))))
	})
	want := "gowt: Rows applies to Table, not to Tree; a Tree takes Nodes\ngowt: Items applies to List and Combo, not to Label"
	if strings.Join(errs, "\n") != want {
		t.Errorf("errors:\n%s\nwant:\n%s", strings.Join(errs, "\n"), want)
	}
}

func benchProcs(n, gen int, changed int) []proc {
	s := make([]proc, n)
	for i := range s {
		s[i] = proc{i, "process " + strconv.Itoa(i), "0.0"}
	}
	for i := range changed {
		s[i*n/changed].cpu = strconv.Itoa(gen)
	}
	return s
}

// benchTable runs one table of 1000 rows; step(i) applies update i.
func benchTable(b *testing.B, build func(w *Window) func(i int)) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		w.SetSize(400, 300)
		step := build(w)
		w.Show()
		app.Flush()
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			step(i)
			app.Flush()
		}
		b.StopTimer()
		w.Close()
	})
}

func columns(t *TableView) {
	t.Column("Name", 200)
	t.Column("CPU", 60)
}

// lists alternates between the two lists per step, built before the timer runs.
func rowsBench(b *testing.B, a, c []proc) {
	benchTable(b, func(w *Window) func(int) {
		list := rx.NewFunc(a, nil)
		put[*TableView](w, Table(Init(columns), Rows(list, procKey, fillProc)))
		return func(i int) {
			if i%2 == 0 {
				list.Set(c)
			} else {
				list.Set(a)
			}
		}
	})
}

// refillBench is the code Rows replaces: clear the table and add every row again.
func refillBench(b *testing.B, a, c []proc) {
	benchTable(b, func(w *Window) func(int) {
		t := put[*TableView](w, Table(Init(columns)))
		refill := func(ps []proc) {
			t.Batch(func() {
				t.Clear()
				for _, p := range ps {
					t.Row(p.name, p.cpu)
				}
			})
		}
		refill(a)
		return func(i int) {
			if i%2 == 0 {
				refill(c)
			} else {
				refill(a)
			}
		}
	})
}

func reversed(p []proc) []proc {
	r := slices.Clone(p)
	slices.Reverse(r)
	return r
}

func renamed(p []proc) []proc {
	r := slices.Clone(p)
	for i := range r {
		r[i].pid += len(p)
	}
	return r
}

func BenchmarkTableRowsUpdate10of1000(b *testing.B) {
	rowsBench(b, benchProcs(1000, 1, 10), benchProcs(1000, 2, 10))
}
func BenchmarkTableRefillUpdate10of1000(b *testing.B) {
	refillBench(b, benchProcs(1000, 1, 10), benchProcs(1000, 2, 10))
}

// The best hand-written update: the rows have not changed, set the cells of the 10 that did.
func BenchmarkTableHandUpdate10of1000(b *testing.B) {
	a, c := benchProcs(1000, 1, 10), benchProcs(1000, 2, 10)
	benchTable(b, func(w *Window) func(int) {
		t := put[*TableView](w, Table(Init(columns)))
		for _, p := range a {
			t.Row(p.name, p.cpu)
		}
		shown := a
		return func(i int) {
			next := c
			if i%2 == 1 {
				next = a
			}
			for j := range next {
				if next[j] != shown[j] {
					fillProc(t.RowAt(j), next[j])
				}
			}
			shown = next
		}
	})
}

func BenchmarkTableRowsReplace1000(b *testing.B) {
	a := benchProcs(1000, 1, 10)
	rowsBench(b, a, renamed(a))
}
func BenchmarkTableRefillReplace1000(b *testing.B) {
	a := benchProcs(1000, 1, 10)
	refillBench(b, a, renamed(a))
}
func BenchmarkTableRowsReverse1000(b *testing.B) {
	a := benchProcs(1000, 1, 10)
	rowsBench(b, a, reversed(a))
}
func BenchmarkTableRefillReverse1000(b *testing.B) {
	a := benchProcs(1000, 1, 10)
	refillBench(b, a, reversed(a))
}

// Insert 10 rows spread over 1000 and remove them again.
func BenchmarkTableRowsInsert10of1000(b *testing.B) {
	a := benchProcs(1000, 1, 10)
	c := slices.Clone(a)
	for i := 9; i >= 0; i-- {
		c = slices.Insert(c, i*100, proc{5000 + i, "new", "1"})
	}
	rowsBench(b, a, c)
}

// A tree of 100 groups with 10 nodes each, 10 leaves changed per update.
type benchNode struct {
	key  string
	text string
}

func benchForest(gen int) ([]benchNode, map[string][]benchNode) {
	roots := make([]benchNode, 100)
	kids := map[string][]benchNode{}
	for g := range roots {
		roots[g] = benchNode{fmt.Sprint("g", g), fmt.Sprint("group ", g)}
		ks := make([]benchNode, 10)
		for k := range ks {
			ks[k] = benchNode{fmt.Sprint(g, "/", k), "0"}
		}
		if g%10 == 0 {
			ks[0].text = strconv.Itoa(gen)
		}
		kids[roots[g].key] = ks
	}
	return roots, kids
}

func BenchmarkTreeNodesUpdate10of1100(b *testing.B) {
	ra, ka := benchForest(1)
	rc, kc := benchForest(2)
	benchTable(b, func(w *Window) func(int) {
		roots := rx.NewFunc(ra, nil)
		kids := ka
		put[*TreeView](w, Tree(Nodes(roots, func(n benchNode) string { return n.key },
			func(n benchNode) []benchNode { return kids[n.key] },
			func(n *Node, it benchNode, isNew bool) {
				n.SetText(it.text)
				if isNew {
					n.SetExpanded(true)
				}
			})))
		return func(i int) {
			if i%2 == 0 {
				kids = kc
				roots.Set(rc)
			} else {
				kids = ka
				roots.Set(ra)
			}
		}
	})
}

// dockerdesk's rebuild: clear the tree and make every node again, expanded.
func BenchmarkTreeRefillUpdate10of1100(b *testing.B) {
	ra, ka := benchForest(1)
	rc, kc := benchForest(2)
	benchTable(b, func(w *Window) func(int) {
		tr := put[*TreeView](w, Tree())
		refill := func(roots []benchNode, kids map[string][]benchNode) {
			tr.Batch(func() {
				tr.Clear()
				for _, r := range roots {
					n := tr.Node(r.text)
					for _, k := range kids[r.key] {
						n.Node(k.text)
					}
					n.SetExpanded(true)
				}
			})
		}
		refill(ra, ka)
		return func(i int) {
			if i%2 == 0 {
				refill(rc, kc)
			} else {
				refill(ra, ka)
			}
		}
	})
}
