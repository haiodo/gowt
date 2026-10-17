package gowt

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

type item struct {
	k string
	v int
}

func itemKey(it item) string { return it.k }

type fakeRow struct {
	id   int
	item item
	sel  bool
}

// fakeFlat is a flat widget: rows in order, each with the id of its native item.
type fakeFlat struct {
	rows    []fakeRow
	next    int
	fills   []string
	inserts int
	quiet   int
	failOn  string
}

func (f *fakeFlat) count() int { return len(f.rows) }

func (f *fakeFlat) insert(pos int, it item) int {
	f.next++
	f.rows = slices.Insert(f.rows, pos, fakeRow{id: f.next, item: it})
	f.inserts++
	f.fill(it)
	return f.next
}

func (f *fakeFlat) fill(it item) {
	f.fills = append(f.fills, it.k)
	if it.k == f.failOn {
		panic("fill failed")
	}
}

func (f *fakeFlat) update(pos int, h int, it item) {
	if f.rows[pos].id != h {
		panic(fmt.Sprintf("update of row %d at %d, which holds %d", h, pos, f.rows[pos].id))
	}
	f.rows[pos].item = it
	f.fill(it)
}

func (f *fakeFlat) remove(pos []int32) {
	if !slices.IsSorted(pos) {
		panic("remove positions not ascending")
	}
	for i := len(pos) - 1; i >= 0; i-- {
		at := int(pos[i])
		sel := f.rows[at].sel
		f.rows = slices.Delete(f.rows, at, at+1)
		if sel && len(f.rows) > 0 { // as GTK: a neighbour of a removed selected row gets selected
			f.rows[min(at, len(f.rows)-1)].sel = true
		}
	}
}

func (f *fakeFlat) clear() { f.rows = nil }

func (f *fakeFlat) selection() []int32 {
	var s []int32
	for i, r := range f.rows {
		if r.sel {
			s = append(s, int32(i))
		}
	}
	return s
}

func (f *fakeFlat) setSelection(pos []int32) {
	for i := range f.rows {
		f.rows[i].sel = slices.Contains(pos, int32(i))
	}
}

func (f *fakeFlat) redraw(on bool) {
	if on {
		f.quiet--
	} else {
		f.quiet++
	}
}

func (f *fakeFlat) ids() map[string]int {
	m := map[string]int{}
	for _, r := range f.rows {
		m[r.item.k] = r.id
	}
	return m
}

func (f *fakeFlat) selectedKeys() []string {
	var s []string
	for _, r := range f.rows {
		if r.sel {
			s = append(s, r.item.k)
		}
	}
	return s
}

func items(spec string) []item {
	var out []item
	for f := range strings.FieldsSeq(spec) {
		k, v, _ := strings.Cut(f, "=")
		n := 0
		fmt.Sscan(v, &n)
		out = append(out, item{k, n})
	}
	return out
}

func newFlat() (*flatList[item, string, int], *fakeFlat) {
	f := &fakeFlat{}
	return &flatList[item, string, int]{key: itemKey, view: f}, f
}

// checkFlat compares the widget with want and the records with the widget.
func checkFlat(t *testing.T, l *flatList[item, string, int], f *fakeFlat, want []item) {
	t.Helper()
	got := make([]item, len(f.rows))
	for i, r := range f.rows {
		got[i] = r.item
	}
	if !slices.Equal(got, want) {
		t.Fatalf("rows %v, want %v", got, want)
	}
	if len(l.rows) != len(f.rows) {
		t.Fatalf("%d records for %d rows", len(l.rows), len(f.rows))
	}
	for i, e := range l.rows {
		if e.h != f.rows[i].id || e.item != f.rows[i].item || e.key != f.rows[i].item.k {
			t.Fatalf("record %d %+v, row %+v", i, e, f.rows[i])
		}
	}
	if f.quiet != 0 {
		t.Fatalf("redraw left off %d times", f.quiet)
	}
}

func TestRowsDiff(t *testing.T) {
	steps := []struct {
		name, list string
		fills      string // keys fill ran for, in order
		inserts    int
	}{
		{"first fill", "a=1 b=1 c=1 d=1 e=1", "a b c d e", 5},
		{"update two", "a=1 b=2 c=1 d=2 e=1", "b d", 0},
		{"equal list", "a=1 b=2 c=1 d=2 e=1", "", 0},
		{"insert middle", "a=1 b=2 x=1 c=1 d=2 e=1", "x", 1},
		{"insert front and end", "y=1 a=1 b=2 x=1 c=1 d=2 e=1 z=1", "y z", 2},
		{"delete", "a=1 b=2 c=1 d=2 e=1", "", 0},
		{"move one", "a=1 c=1 d=2 b=2 e=1", "b", 1},
		{"swap ends", "e=1 c=1 d=2 b=2 a=1", "e a", 2},
		{"move and update", "c=5 e=1 d=2 b=2 a=1", "c", 1},
		{"full replace", "p=1 q=1 r=1", "p q r", 3},
		{"empty", "", "", 0},
		{"refill", "a=1 b=1", "a b", 2},
	}
	l, f := newFlat()
	before := map[string]int{}
	for _, s := range steps {
		f.fills, f.inserts = nil, 0
		want := items(s.list)
		l.sync(want)
		checkFlat(t, l, f, want)
		if got := strings.Join(f.fills, " "); got != s.fills || f.inserts != s.inserts {
			t.Errorf("%s: fills %q inserts %d, want %q %d", s.name, got, f.inserts, s.fills, s.inserts)
		}
		// A key that was there and got no new native item keeps its id.
		ids := f.ids()
		for k, id := range ids {
			if old, ok := before[k]; ok && old != id && !slices.Contains(f.fills, k) {
				t.Errorf("%s: %s got a new row without a fill", s.name, k)
			}
		}
		before = ids
	}
}

func TestRowsKeepSelection(t *testing.T) {
	l, f := newFlat()
	l.sync(items("a b c d e"))
	for i := range f.rows {
		f.rows[i].sel = f.rows[i].item.k == "b" || f.rows[i].item.k == "d"
	}
	l.sync(items("e b x d=1 a")) // c gone, x new, d changed, e and a moved
	checkFlat(t, l, f, items("e b x d=1 a"))
	if got := f.selectedKeys(); !slices.Equal(got, []string{"b", "d"}) {
		t.Errorf("selection %v after an update", got)
	}
	l.sync(items("d=1 a b e")) // d moves to the front
	if got := f.selectedKeys(); !slices.Equal(got, []string{"d", "b"}) {
		t.Errorf("selection %v after moving d", got)
	}
	l.sync(items("a e"))
	if got := f.selectedKeys(); len(got) != 0 {
		t.Errorf("selection %v after its rows went", got)
	}
}

func TestRowsDuplicateKeys(t *testing.T) {
	l, f := newFlat()
	for _, s := range []string{"a a b", "a b a", "b a a a", "a"} {
		l.sync(items(s))
		checkFlat(t, l, f, items(s))
	}
}

func TestRowsRecoverFromFill(t *testing.T) {
	l, f := newFlat()
	l.sync(items("a b c"))
	f.failOn = "x"
	func() {
		defer func() { recover() }()
		l.sync(items("a x b c"))
	}()
	f.failOn = ""
	l.sync(items("a x b c"))
	checkFlat(t, l, f, items("a x b c"))
	f.rows = f.rows[:1] // rows removed by hand
	l.sync(items("a b"))
	checkFlat(t, l, f, items("a b"))
}

func TestRowsRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	l, f := newFlat()
	for range 2000 {
		n := r.IntN(30)
		list := make([]item, 0, n)
		for _, k := range r.Perm(40)[:n] {
			list = append(list, item{fmt.Sprint(k), r.IntN(3)})
		}
		l.sync(list)
		checkFlat(t, l, f, list)
	}
}

// fnode is a node of fakeTree.
type fnode struct {
	id     int
	item   item
	kids   []*fnode
	parent *fnode
	open   bool
	sel    bool
	gone   bool
}

type fakeTree struct {
	root  fnode
	next  int
	fills []string // key, "+" when created
	quiet int
}

func (f *fakeTree) at(parent *fnode) *fnode {
	if parent == nil {
		return &f.root
	}
	return parent
}

func (f *fakeTree) count(parent *fnode) int { return len(f.at(parent).kids) }

func (f *fakeTree) insert(parent *fnode, pos int) *fnode {
	f.next++
	p := f.at(parent)
	n := &fnode{id: f.next, parent: p}
	p.kids = slices.Insert(p.kids, pos, n)
	return n
}

func (f *fakeTree) fill(n *fnode, it item, created bool) {
	n.item = it
	s := it.k
	if created {
		s += "+"
	}
	f.fills = append(f.fills, s)
}

func (f *fakeTree) dispose(n *fnode) {
	ks := n.parent.kids
	i := slices.Index(ks, n)
	n.parent.kids = slices.Delete(ks, i, i+1)
	if f.drop(n) && len(n.parent.kids) > 0 { // as GTK: a neighbour of a removed selected node gets selected
		n.parent.kids[min(i, len(n.parent.kids)-1)].sel = true
	}
}

// drop marks n and its subtree disposed and reports whether one of them was selected.
func (f *fakeTree) drop(n *fnode) bool {
	n.gone = true
	sel := n.sel
	for _, k := range n.kids {
		sel = f.drop(k) || sel
	}
	return sel
}

func (f *fakeTree) clear(parent *fnode) {
	p := f.at(parent)
	for _, k := range p.kids {
		f.drop(k)
	}
	p.kids = nil
}

func (f *fakeTree) disposed(n *fnode) bool { return n.gone }
func (f *fakeTree) expanded(n *fnode) bool { return n.open }

// setExpanded does nothing on a leaf, as SWT's TreeItem.setExpanded.
func (f *fakeTree) setExpanded(n *fnode, on bool) {
	if len(n.kids) > 0 {
		n.open = on
	}
}

func (f *fakeTree) selection() []*fnode {
	var s []*fnode
	var walk func(*fnode)
	walk = func(n *fnode) {
		if n.sel {
			s = append(s, n)
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(&f.root)
	return s
}

func (f *fakeTree) setSelection(ns []*fnode) {
	for _, n := range f.selection() {
		n.sel = false
	}
	for _, n := range ns {
		n.sel = true
	}
}

func (f *fakeTree) redraw(on bool) {
	if on {
		f.quiet--
	} else {
		f.quiet++
	}
}

// dump writes the tree as "a(b c*) d", * for selected, ! for expanded.
func (f *fakeTree) dump() string {
	var b strings.Builder
	var walk func(ns []*fnode)
	walk = func(ns []*fnode) {
		for i, n := range ns {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(n.item.k)
			if n.item.v != 0 {
				fmt.Fprintf(&b, "=%d", n.item.v)
			}
			if n.sel {
				b.WriteByte('*')
			}
			if n.open {
				b.WriteByte('!')
			}
			if len(n.kids) > 0 {
				b.WriteByte('(')
				walk(n.kids)
				b.WriteByte(')')
			}
		}
	}
	walk(f.root.kids)
	return b.String()
}

func (f *fakeTree) find(k string) *fnode {
	var hit *fnode
	var walk func(ns []*fnode)
	walk = func(ns []*fnode) {
		for _, n := range ns {
			if n.item.k == k {
				hit = n
			}
			walk(n.kids)
		}
	}
	walk(f.root.kids)
	return hit
}

// forest parses "a(b c(d)) e" into roots and a children map.
func forest(spec string) ([]item, map[string][]item) {
	kids := map[string][]item{}
	var stack []string
	var roots []item
	tok := ""
	flush := func() {
		if tok == "" {
			return
		}
		it := items(tok)[0]
		if len(stack) == 0 {
			roots = append(roots, it)
		} else {
			p := stack[len(stack)-1]
			kids[p] = append(kids[p], it)
		}
		tok = ""
	}
	last := ""
	for _, c := range spec {
		switch c {
		case ' ':
			if tok != "" {
				last = items(tok)[0].k
			}
			flush()
		case '(':
			if tok != "" {
				last = items(tok)[0].k
			}
			flush()
			stack = append(stack, last)
		case ')':
			flush()
			stack = stack[:len(stack)-1]
		default:
			tok += string(c)
		}
	}
	flush()
	return roots, kids
}

func TestNodesDiff(t *testing.T) {
	var kids map[string][]item
	f := &fakeTree{}
	l := &treeList[item, string, *fnode]{key: itemKey, children: func(it item) []item { return kids[it.k] }, view: f}
	step := func(spec, wantFills string) {
		t.Helper()
		var roots []item
		roots, kids = forest(spec)
		f.fills = nil
		l.sync(roots)
		if got := f.dump(); strings.ReplaceAll(strings.ReplaceAll(got, "*", ""), "!", "") != spec {
			t.Fatalf("tree %q, want %q", got, spec)
		}
		if got := strings.Join(f.fills, " "); got != wantFills {
			t.Errorf("%s: fills %q, want %q", spec, got, wantFills)
		}
		if f.quiet != 0 {
			t.Fatalf("redraw left off")
		}
	}
	step("g(a b) h(c) d", "a+ b+ g+ c+ h+ d+")
	f.find("g").open, f.find("h").open, f.find("b").sel = true, true, true
	step("g(a b=1) h(c) d", "b") // a changed leaf: only it is filled
	if got := f.dump(); got != "g!(a b=1*) h!(c) d" {
		t.Errorf("after update %q", got)
	}
	step("g(a b=1 x) h(c) d", "x+")
	step("g(b=1 a x) h(c) d", "b+") // the longest run in order is a x: b is made again and stays selected
	if got := f.dump(); got != "g!(b=1* a x) h!(c) d" {
		t.Errorf("after moving b %q", got)
	}
	f.find("h").open = false
	step("h(c) d g(b=1 a x)", "b+ a+ x+ g+") // g moved with its subtree, which keeps expansion and selection
	if got := f.dump(); got != "h(c) d g!(b=1* a x)" {
		t.Errorf("after moving g %q", got)
	}
	step("h(c g(b=1 a x)) d", "b+ a+ x+ g+") // a new parent: g is new there; h is not filled, its element is equal
	step("d=1", "d")
	step("", "")
}

// TestNodesGhostDepth moves a node whose subtree is two levels deep: every level keeps its state.
func TestNodesGhostDepth(t *testing.T) {
	var kids map[string][]item
	f := &fakeTree{}
	l := &treeList[item, string, *fnode]{key: itemKey, children: func(it item) []item { return kids[it.k] }, view: f}
	set := func(spec string) {
		var roots []item
		roots, kids = forest(spec)
		l.sync(roots)
	}
	set("r(a(b(c) d)) y z")
	f.find("a").open, f.find("b").open, f.find("c").sel, f.find("r").open = true, true, true, true
	set("y z r(a(d b(c)))")
	if got := f.dump(); got != "y z r!(a!(d b!(c*)))" {
		t.Errorf("after moves %q", got)
	}
}

// Steady state: the same keys with a few changed elements allocate nothing.
func TestRowsUpdateAllocs(t *testing.T) {
	l := &flatList[item, string, struct{}]{key: itemKey, view: &nullFlat{}}
	a, b := benchItems(1000, 0), benchItems(1000, 10)
	l.sync(a)
	n := 0
	allocs := testing.AllocsPerRun(100, func() {
		if n++; n%2 == 0 {
			l.sync(a)
		} else {
			l.sync(b)
		}
	})
	if allocs != 0 {
		t.Errorf("%v allocs per update", allocs)
	}
}

func benchItems(n, changed int) []item {
	s := make([]item, n)
	for i := range s {
		s[i] = item{fmt.Sprintf("k%04d", i), 0}
	}
	for i := 0; i < changed; i++ {
		s[i*n/max(changed, 1)].v = 1
	}
	return s
}

// nullFlat is a widget that does nothing, for the cost of the diff alone.
type nullFlat struct{ n int }

func (v *nullFlat) count() int                 { return v.n }
func (v *nullFlat) insert(int, item) struct{}  { v.n++; return struct{}{} }
func (v *nullFlat) update(int, struct{}, item) {}
func (v *nullFlat) remove(p []int32)           { v.n -= len(p) }
func (v *nullFlat) clear()                     { v.n = 0 }
func (v *nullFlat) selection() []int32         { return nil }
func (v *nullFlat) setSelection([]int32)       {}
func (v *nullFlat) redraw(bool)                {}

func benchDiff(b *testing.B, a, c []item) {
	l := &flatList[item, string, struct{}]{key: itemKey, view: &nullFlat{}}
	l.sync(a)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if i%2 == 0 {
			l.sync(c)
		} else {
			l.sync(a)
		}
	}
}

func BenchmarkDiffUpdate10of1000(b *testing.B) {
	benchDiff(b, benchItems(1000, 0), benchItems(1000, 10))
}

func BenchmarkDiffReverse1000(b *testing.B) {
	a := benchItems(1000, 0)
	c := slices.Clone(a)
	slices.Reverse(c)
	benchDiff(b, a, c)
}

func BenchmarkDiffReplace1000(b *testing.B) {
	a := benchItems(1000, 0)
	c := make([]item, len(a))
	for i := range c {
		c[i] = item{fmt.Sprintf("n%04d", i), 0}
	}
	benchDiff(b, a, c)
}

func TestNodesRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	kids := map[string][]item{}
	f := &fakeTree{}
	l := &treeList[item, string, *fnode]{key: itemKey, children: func(it item) []item { return kids[it.k] }, view: f}
	pick := func(prefix string) []item {
		var out []item
		for _, k := range r.Perm(8)[:r.IntN(6)] {
			out = append(out, item{fmt.Sprint(prefix, k), r.IntN(2)})
		}
		return out
	}
	var same func(ns []*fnode, want []item) bool
	same = func(ns []*fnode, want []item) bool {
		if len(ns) != len(want) {
			return false
		}
		for i, n := range ns {
			if n.item != want[i] || !same(n.kids, kids[want[i].k]) {
				return false
			}
		}
		return true
	}
	for range 2000 {
		clear(kids)
		roots := pick("r")
		for _, it := range roots {
			kids[it.k] = pick(it.k + ".")
			for _, k := range kids[it.k] {
				kids[k.k] = pick(k.k + ".")
			}
		}
		l.sync(roots)
		if !same(f.root.kids, roots) || f.quiet != 0 {
			t.Fatalf("tree %s for roots %v", f.dump(), roots)
		}
	}
}

func TestNodesUpdateAllocs(t *testing.T) {
	kids := map[string][]item{}
	a, b := benchItems(100, 0), benchItems(100, 10)
	for _, it := range a {
		for i := range 10 {
			kids[it.k] = append(kids[it.k], item{fmt.Sprint(it.k, ".", i), 0})
		}
	}
	l := &treeList[item, string, *fnode]{key: itemKey, children: func(it item) []item { return kids[it.k] }, view: &fakeTree{}}
	l.sync(a)
	n := 0
	allocs := testing.AllocsPerRun(100, func() {
		if n++; n%2 == 0 {
			l.sync(a)
		} else {
			l.sync(b)
		}
	})
	if allocs != 0 {
		t.Errorf("%v allocs per update", allocs)
	}
}

// A removed selected node takes its selection with it; the fake, as GTK, passes it to a neighbour.
func TestNodesRemoveSelected(t *testing.T) {
	f := &fakeTree{}
	l := &treeList[item, string, *fnode]{key: itemKey, view: f}
	l.sync(items("a b c"))
	f.find("b").sel = true
	l.sync(items("a c"))
	if got := f.dump(); got != "a c" {
		t.Errorf("tree %q after removing the selected node", got)
	}
}
