package gowt

import (
	"fmt"
	"slices"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// Rows shows one row of a Table per element of list, in the order of list: the {#each list as item (key)}
// of Svelte. fill sets up the row of a new element and runs again for a row whose element is no longer ==
// to the one it showed. When list changes, rows are matched by key: a row whose key is gone is removed,
// a new key gets a new row, and the other rows stay as they are, selection included. A row whose key moved
// is made again at its new place (fill runs) and keeps its selection. Keys should be unique; a repeated
// key gets a row made again on every change.
//
// The binding owns the rows: add the columns in Init and no rows by hand. fill runs untracked, after
// the Init options, whatever the order of the options.
func Rows[T, K comparable](list rx.Reader[[]T], key func(T) K, fill func(r *TableRow, item T)) Option {
	return lateHook(func(w Widget) error {
		t, ok := w.(*TableView)
		if !ok {
			return fmt.Errorf("gowt: Rows applies to Table, not to %s; a Tree takes Nodes", nameOf(w.control()))
		}
		l := &flatList[T, K, *swt.TableItem]{key: key, view: &tableRows[T]{t.t, fill}}
		Watch(w, list, func(v []T) { l.sync(v); refitColumns(w.control()) })
		return nil
	})
}

// Nodes shows one root node of a Tree per element of list and below each node one child per element of
// children(item), nil for a flat tree. It is Rows for a tree, level by level: siblings are matched by
// key, a surviving node keeps its expansion and selection, and a node whose key moved among its siblings
// is made again with its subtree and keeps the expansion and selection of every node in it. fill runs
// after the children of the node exist; created is true when the node is made, also again after a move,
// and false when its element changed: set the initial expansion when created.
func Nodes[T, K comparable](list rx.Reader[[]T], key func(T) K, children func(T) []T, fill func(n *Node, item T, created bool)) Option {
	return lateHook(func(w Widget) error {
		t, ok := w.(*TreeView)
		if !ok {
			return fmt.Errorf("gowt: Nodes applies to Tree, not to %s; a Table takes Rows", nameOf(w.control()))
		}
		l := &treeList[T, K, *swt.TreeItem]{key: key, children: children, view: &treeNodes[T]{t.t, fill}}
		Watch(w, list, func(v []T) { l.sync(v); refitColumns(w.control()) })
		return nil
	})
}

// Items shows the strings of list as the entries of a List or a Combo. Entries are matched by text, as
// Rows matches rows by key, so the selection stays on an entry that is still there. The binding owns
// the entries: give nil items to the constructor.
func Items(list rx.Reader[[]string]) Option {
	return lateHook(func(w Widget) error {
		var v flatView[string, struct{}]
		switch w := w.(type) {
		case *ListView:
			v = listItems{w.l}
		case *ComboView:
			v = comboItems{w.c}
		default:
			return fmt.Errorf("gowt: Items applies to List and Combo, not to %s", nameOf(w.control()))
		}
		l := &flatList[string, string, struct{}]{key: func(s string) string { return s }, view: v}
		Watch(w, list, l.sync)
		return nil
	})
}

// planner matches the old and new sibling lists between their common prefix and suffix. Its buffers
// live as long as the binding and grow to the largest middle seen; the map is emptied after each match.
type planner[K comparable] struct {
	at    map[K]int32
	src   []int32 // per new element: its index in the old middle, -1 for a new key
	fate  []uint8 // per old element
	tails []int32
	prev  []int32
}

const (
	gone  uint8 = iota // its key is not in the new list
	moved              // made again at its new place
	stays              // in the longest run of matched elements already in order

	picked uint8 = 0x80 // added to a fate: the row was selected
)

func (p *planner[K]) match(oldN int, oldKey func(int) K, newN int, newKey func(int) K) {
	if p.at == nil {
		p.at = make(map[K]int32, oldN)
	}
	p.fate = sized(p.fate, oldN)
	clear(p.fate)
	for i := range oldN {
		p.at[oldKey(i)] = int32(i)
	}
	p.src = sized(p.src, newN)
	for j := range newN {
		k := newKey(j)
		i, ok := p.at[k]
		if !ok {
			p.src[j] = -1
			continue
		}
		delete(p.at, k) // a repeated key in the new list is a new element
		p.src[j] = i
		p.fate[i] = moved
	}
	for i := range oldN {
		delete(p.at, oldKey(i))
	}
	p.longestRun()
}

// longestRun marks stays the longest increasing subsequence of src, O(n log n): the most old elements
// that can keep their native items while the others move around them.
func (p *planner[K]) longestRun() {
	src := p.src
	p.prev = sized(p.prev, len(src))
	tails := p.tails[:0]
	for j, s := range src {
		if s < 0 {
			continue
		}
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := int(uint(lo+hi) >> 1)
			if src[tails[mid]] < s {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		p.prev[j] = -1
		if lo > 0 {
			p.prev[j] = tails[lo-1]
		}
		if lo == len(tails) {
			tails = append(tails, int32(j))
		} else {
			tails[lo] = int32(j)
		}
	}
	p.tails = tails
	if len(tails) == 0 {
		return
	}
	for j := tails[len(tails)-1]; j >= 0; j = p.prev[j] {
		p.fate[src[j]] = stays
	}
}

func sized[E any](s []E, n int) []E {
	if cap(s) < n {
		return make([]E, n)
	}
	return s[:n]
}

// commonEnds returns the lengths of the common prefix and suffix of the old keys and the new list.
func commonEnds[T any, K comparable](m int, oldKey func(int) K, items []T, key func(T) K) (a, b int) {
	n := len(items)
	for a < m && a < n && oldKey(a) == key(items[a]) {
		a++
	}
	for b < m-a && b < n-a && oldKey(m-1-b) == key(items[n-1-b]) {
		b++
	}
	return a, b
}

// flatView is the widget side of a flat list of rows; H is what a row keeps of its native item.
type flatView[T, H any] interface {
	count() int
	insert(pos int, item T) H // makes the row at pos and fills it
	update(pos int, h H, item T)
	remove(pos []int32) // ascending positions in the current rows
	clear()
	selection() []int32
	setSelection(pos []int32) // without scrolling
	redraw(on bool)
}

type entry[T, K, H any] struct {
	key  K
	item T
	h    H
}

// flatList keeps the rows of a widget in step with a slice; rows holds the key and the element each
// row shows, which is all the diff needs.
type flatList[T, K comparable, H any] struct {
	key  func(T) K
	view flatView[T, H]
	rows []entry[T, K, H]
	p    planner[K]
	rm   []int32
	busy bool // still set when a fill panicked: the rows no longer match the records
}

func (l *flatList[T, K, H]) sync(items []T) {
	if l.busy || l.view.count() != len(l.rows) {
		l.view.clear()
		l.rows = nil
	}
	l.busy = true
	old := l.rows
	a, b := commonEnds(len(old), func(i int) K { return old[i].key }, items, l.key)
	if a+b == len(old) && a+b == len(items) {
		for j := range old {
			l.update(j, &old[j], items[j])
		}
	} else {
		l.restructure(items, a, b)
	}
	l.busy = false
}

func (l *flatList[T, K, H]) update(pos int, e *entry[T, K, H], item T) {
	if e.item != item {
		e.item = item
		l.view.update(pos, e.h, item)
	}
}

func (l *flatList[T, K, H]) restructure(items []T, a, b int) {
	l.view.redraw(false)
	defer l.view.redraw(true)
	old := l.rows
	m, n := len(old), len(items)
	oldMid, newMid := old[a:m-b], items[a:n-b]
	p := &l.p
	p.match(len(oldMid), func(i int) K { return oldMid[i].key }, len(newMid), func(j int) K { return l.key(newMid[j]) })
	// GTK selects a neighbour of a removed selected row: then the selection is set again at the end.
	sel := l.view.selection()
	lost := false
	for _, s := range sel {
		if i := int(s) - a; i >= 0 && i < len(oldMid) {
			p.fate[i] |= picked
			lost = lost || p.fate[i] != stays|picked
		}
	}
	rm := l.rm[:0]
	for i := range oldMid {
		if p.fate[i]&^picked != stays {
			rm = append(rm, int32(a+i))
		}
	}
	// Removing rows one by one costs more than making them all again once most of them go.
	full := 2*len(rm) > m
	if full {
		l.view.clear()
		lost = lost || len(sel) > 0
	} else if len(rm) > 0 {
		l.view.remove(rm)
	}
	want := rm[:0]
	for _, s := range sel {
		if int(s) < a {
			want = append(want, s)
		}
	}
	rows := make([]entry[T, K, H], n)
	for j := range a {
		rows[j] = l.keep(full, j, old[j], items[j])
	}
	for j, it := range newMid {
		pos, s := a+j, p.src[j]
		if s >= 0 && p.fate[s]&picked != 0 {
			want = append(want, int32(pos))
		}
		if s >= 0 && p.fate[s]&^picked == stays {
			rows[pos] = l.keep(full, pos, oldMid[s], it)
		} else {
			rows[pos] = entry[T, K, H]{key: l.key(it), item: it, h: l.view.insert(pos, it)}
		}
	}
	for k := range b {
		pos := n - b + k
		rows[pos] = l.keep(full, pos, old[m-b+k], items[pos])
	}
	for _, s := range sel {
		if int(s) >= m-b {
			want = append(want, s-int32(m-n))
		}
	}
	if lost {
		l.view.setSelection(want)
	}
	l.rm = want[:0]
	l.rows = rows
}

// keep returns the record of a row whose key stays at pos: the same row updated, or a new one after a clear.
func (l *flatList[T, K, H]) keep(full bool, pos int, e entry[T, K, H], item T) entry[T, K, H] {
	if full {
		return entry[T, K, H]{key: e.key, item: item, h: l.view.insert(pos, item)}
	}
	l.update(pos, &e, item)
	return e
}

// treeView is the widget side of a tree; the zero H is the invisible root.
type treeView[T any, H comparable] interface {
	count(parent H) int
	insert(parent H, pos int) H
	fill(h H, item T, created bool)
	dispose(h H)
	clear(parent H)
	expanded(h H) bool
	setExpanded(h H, v bool)
	selection() []H
	setSelection(hs []H) // without scrolling
	disposed(h H) bool
	redraw(on bool)
}

type tentry[T, K, H any] struct {
	key  K
	item T
	h    H
	kids []tentry[T, K, H]
	open bool // read from the widget just before a moved node is made again
	sel  bool
}

type treeList[T, K comparable, H comparable] struct {
	key      func(T) K
	children func(T) []T
	view     treeView[T, H]
	roots    []tentry[T, K, H]
	ps       []*planner[K] // one per depth: a level's plan is read while its children are synced
	sel      []H           // the selection, read once per update before the first node is disposed
	selRead  bool
	restored []H  // nodes made again after a move that were selected before
	cut      bool // a node was disposed while some were selected
	quiet    bool // drawing is off until the update ends
	busy     bool
}

func (t *treeList[T, K, H]) sync(items []T) {
	var root H
	if t.busy || t.view.count(root) != len(t.roots) {
		t.view.clear(root)
		t.roots = nil
	}
	t.busy = true
	defer func() {
		if t.quiet {
			t.view.redraw(true)
			t.quiet = false
		}
		t.sel, t.selRead, t.restored, t.cut = nil, false, nil, false
	}()
	t.roots = t.level(root, t.roots, items, 0)
	if t.cut {
		t.reselect()
	}
	t.busy = false
}

// reselect puts back the selection a disposal changed: GTK selects a neighbour of a removed selected node.
func (t *treeList[T, K, H]) reselect() {
	want := t.restored
	for _, h := range t.sel {
		if !t.view.disposed(h) {
			want = append(want, h)
		}
	}
	got := t.view.selection()
	if len(got) != len(want) || slices.ContainsFunc(want, func(h H) bool { return !slices.Contains(got, h) }) {
		t.view.setSelection(want)
	}
}

func (t *treeList[T, K, H]) level(parent H, old []tentry[T, K, H], items []T, depth int) []tentry[T, K, H] {
	a, b := commonEnds(len(old), func(i int) K { return old[i].key }, items, t.key)
	if a+b < len(old) || a+b < len(items) {
		return t.restructure(parent, old, items, a, b, depth)
	}
	for j := range old {
		t.update(&old[j], items[j], depth)
	}
	return old
}

func (t *treeList[T, K, H]) update(e *tentry[T, K, H], item T, depth int) {
	if t.children != nil {
		e.kids = t.level(e.h, e.kids, t.children(item), depth+1)
	}
	if e.item != item {
		e.item = item
		t.view.fill(e.h, item, false)
	}
}

func (t *treeList[T, K, H]) restructure(parent H, old []tentry[T, K, H], items []T, a, b, depth int) []tentry[T, K, H] {
	if !t.quiet {
		t.view.redraw(false)
		t.quiet = true
	}
	m, n := len(old), len(items)
	oldMid, newMid := old[a:m-b], items[a:n-b]
	p := t.planner(depth)
	p.match(len(oldMid), func(i int) K { return oldMid[i].key }, len(newMid), func(j int) K { return t.key(newMid[j]) })
	t.readSelection()
	removed := 0
	for i := range oldMid {
		if p.fate[i] != stays {
			removed++
		}
	}
	// As in flatList: once most siblings go, clearing them all is cheaper; the rest come back as ghosts.
	full := 2*removed > m
	for i := range old {
		if i < a || i >= m-b {
			if full {
				t.remember(&old[i])
			}
		} else if f := p.fate[i-a]; f == moved || (full && f == stays) {
			t.remember(&old[i])
		}
	}
	t.cut = t.cut || ((removed > 0 || full) && len(t.sel) > 0)
	switch {
	case full:
		t.view.clear(parent)
	case removed > 0:
		for i := range oldMid {
			if p.fate[i] != stays {
				t.view.dispose(oldMid[i].h)
			}
		}
	}
	rows := make([]tentry[T, K, H], n)
	for j := range a {
		rows[j] = t.keep(full, parent, j, &old[j], items[j], depth)
	}
	for j, it := range newMid {
		pos, s := a+j, p.src[j]
		switch {
		case s < 0:
			rows[pos] = t.create(parent, pos, it, nil, depth)
		case p.fate[s] == stays:
			rows[pos] = t.keep(full, parent, pos, &oldMid[s], it, depth)
		default:
			rows[pos] = t.create(parent, pos, it, &oldMid[s], depth)
		}
	}
	for k := range b {
		pos := n - b + k
		rows[pos] = t.keep(full, parent, pos, &old[m-b+k], items[pos], depth)
	}
	return rows
}

// keep returns the record of a node whose key stays at pos: the same node updated, or one made
// again from its ghost after a clear.
func (t *treeList[T, K, H]) keep(full bool, parent H, pos int, e *tentry[T, K, H], item T, depth int) tentry[T, K, H] {
	if full {
		return t.create(parent, pos, item, e, depth)
	}
	t.update(e, item, depth)
	return *e
}

// create makes the node of item at pos, its subtree, then fills it. ghost is the record of the node
// with the same key before it moved: the new nodes take its expansion and selection, level by level.
func (t *treeList[T, K, H]) create(parent H, pos int, item T, ghost *tentry[T, K, H], depth int) tentry[T, K, H] {
	h := t.view.insert(parent, pos)
	e := tentry[T, K, H]{key: t.key(item), item: item, h: h}
	var kids []T
	if t.children != nil {
		kids = t.children(item)
	}
	if len(kids) > 0 {
		e.kids = make([]tentry[T, K, H], len(kids))
		var ghosts []tentry[T, K, H]
		if ghost != nil {
			ghosts = ghost.kids
		}
		var p *planner[K]
		if len(ghosts) > 0 {
			// depth+1 is free here: the recursion below plans at depth+2 and deeper.
			p = t.planner(depth + 1)
			p.match(len(ghosts), func(i int) K { return ghosts[i].key }, len(kids), func(j int) K { return t.key(kids[j]) })
		}
		for j, kid := range kids {
			var g *tentry[T, K, H]
			if p != nil && p.src[j] >= 0 {
				g = &ghosts[p.src[j]]
			}
			e.kids[j] = t.create(h, j, kid, g, depth+1)
		}
	}
	t.view.fill(h, item, true)
	if ghost != nil {
		t.view.setExpanded(h, ghost.open)
		if ghost.sel {
			t.restored = append(t.restored, h)
		}
	}
	return e
}

// remember reads the expansion and selection of a moved subtree before it is disposed. The selection
// test is linear in the selected nodes: fine for a hand-made selection, slow after SelectAll on a big tree.
func (t *treeList[T, K, H]) remember(e *tentry[T, K, H]) {
	e.open = t.view.expanded(e.h)
	e.sel = slices.Contains(t.sel, e.h)
	for i := range e.kids {
		t.remember(&e.kids[i])
	}
}

func (t *treeList[T, K, H]) readSelection() {
	if !t.selRead {
		t.sel, t.selRead = t.view.selection(), true
	}
}

func (t *treeList[T, K, H]) planner(depth int) *planner[K] {
	for len(t.ps) <= depth {
		t.ps = append(t.ps, &planner[K]{})
	}
	return t.ps[depth]
}

type tableRows[T any] struct {
	t    *swt.Table
	fill func(*TableRow, T)
}

func (v *tableRows[T]) count() int { return int(v.t.GetItemCount()) }

func (v *tableRows[T]) insert(pos int, item T) *swt.TableItem {
	it := swt.NewTableItemParentStyleIndex(v.t, swt.NONE, int32(pos))
	v.fill(&TableRow{it}, item)
	return it
}

func (v *tableRows[T]) update(_ int, it *swt.TableItem, item T) { v.fill(&TableRow{it}, item) }
func (v *tableRows[T]) remove(pos []int32)                      { v.t.RemoveIndices(pos) }
func (v *tableRows[T]) clear()                                  { v.t.RemoveAll() }
func (v *tableRows[T]) selection() []int32                      { return v.t.GetSelectionIndices() }

func (v *tableRows[T]) setSelection(pos []int32) {
	v.t.DeselectAll0()
	v.t.SelectIndices(pos)
}

func (v *tableRows[T]) redraw(on bool) { v.t.SetRedraw(on) }

type treeNodes[T any] struct {
	t *swt.Tree
	f func(*Node, T, bool)
}

func (v *treeNodes[T]) count(parent *swt.TreeItem) int {
	if parent == nil {
		return int(v.t.GetItemCount())
	}
	return int(parent.GetItemCount())
}

func (v *treeNodes[T]) insert(parent *swt.TreeItem, pos int) *swt.TreeItem {
	if parent == nil {
		return swt.NewTreeItemParentStyleIndex(v.t, swt.NONE, int32(pos))
	}
	return swt.NewTreeItemParentItemStyleIndex(parent, swt.NONE, int32(pos))
}

func (v *treeNodes[T]) fill(it *swt.TreeItem, item T, created bool) { v.f(&Node{it}, item, created) }
func (v *treeNodes[T]) dispose(it *swt.TreeItem)                    { it.Dispose() }

func (v *treeNodes[T]) clear(parent *swt.TreeItem) {
	if parent == nil {
		v.t.RemoveAll()
		return
	}
	parent.RemoveAll()
}

func (v *treeNodes[T]) expanded(it *swt.TreeItem) bool        { return it.GetExpanded() }
func (v *treeNodes[T]) setExpanded(it *swt.TreeItem, on bool) { it.SetExpanded(on) }
func (v *treeNodes[T]) selection() []*swt.TreeItem            { return v.t.GetSelection() }
func (v *treeNodes[T]) disposed(it *swt.TreeItem) bool        { return it.IsDisposed() }

func (v *treeNodes[T]) setSelection(its []*swt.TreeItem) {
	v.t.DeselectAll0()
	for _, it := range its {
		v.t.Select(it)
	}
}

func (v *treeNodes[T]) redraw(on bool) { v.t.SetRedraw(on) }

type listItems struct{ l *swt.List }

func (v listItems) count() int { return int(v.l.GetItemCount()) }
func (v listItems) insert(pos int, s string) struct{} {
	v.l.AddStringIndex(s, int32(pos))
	return struct{}{}
}
func (v listItems) update(pos int, _ struct{}, s string) { v.l.SetItem(int32(pos), s) }
func (v listItems) remove(pos []int32)                   { v.l.RemoveIndices(pos) }
func (v listItems) clear()                               { v.l.RemoveAll() }
func (v listItems) selection() []int32                   { return v.l.GetSelectionIndices() }

func (v listItems) setSelection(pos []int32) {
	v.l.DeselectAll()
	v.l.SelectIndices(pos)
}

func (v listItems) redraw(on bool) { v.l.SetRedraw(on) }

type comboItems struct{ c *swt.Combo }

func (v comboItems) count() int { return int(v.c.GetItemCount()) }
func (v comboItems) insert(pos int, s string) struct{} {
	v.c.AddStringIndex(s, int32(pos))
	return struct{}{}
}
func (v comboItems) update(pos int, _ struct{}, s string) { v.c.SetItem(int32(pos), s) }

func (v comboItems) remove(pos []int32) {
	for i := len(pos) - 1; i >= 0; i-- {
		v.c.Remove(pos[i])
	}
}

func (v comboItems) clear() { v.c.RemoveAll() }
func (v comboItems) selection() []int32 {
	if i := v.c.GetSelectionIndex(); i >= 0 {
		return []int32{i}
	}
	return nil
}

func (v comboItems) setSelection(pos []int32) {
	if len(pos) == 0 {
		v.c.DeselectAll()
		return
	}
	v.c.Select(pos[0])
}

func (v comboItems) redraw(on bool) { v.c.SetRedraw(on) }
