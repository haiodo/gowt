package swt

import "testing"

func TestVisibleRgnHolders(t *testing.T) {
	saved := visibleRgnHolders
	t.Cleanup(func() { visibleRgnHolders = saved })
	visibleRgnHolders = map[*Control]struct{}{}
	d := &Display{}
	root := &Composite{}
	mid := &Composite{}
	leaf := &Control{}
	other := &Composite{}
	for _, c := range []*Control{&root.Control, &mid.Control, leaf, &other.Control} {
		c.display = d
	}
	mid.parent, leaf.parent, other.parent = root, mid, root

	if !visibleRegionsIdle(&root.Control) {
		t.Fatal("empty set must be idle")
	}
	markVisibleRgn(leaf)
	for _, c := range []*Control{leaf, &mid.Control, &root.Control} {
		if visibleRegionsIdle(c) {
			t.Fatal("chain to the root must be marked")
		}
	}
	if !visibleRegionsIdle(&other.Control) {
		t.Fatal("a sibling subtree must stay idle")
	}
	// reparent: marks the new chain
	leaf.parent = other
	markVisibleRgnReparent(leaf)
	if visibleRegionsIdle(&other.Control) {
		t.Fatal("new parent must be marked after reparent")
	}
	clearVisibleRgn(leaf)
	forgetVisibleRgn(&mid.Control)
	if !visibleRegionsIdle(leaf) || !visibleRegionsIdle(&mid.Control) {
		t.Fatal("cleared nodes must be idle")
	}
	d.contexts = make([]*GCData, 1)
	markVisibleRgn(leaf)
	clearVisibleRgn(leaf)
	if visibleRegionsIdle(&root.Control) || visibleRegionsIdle(&other.Control) {
		t.Fatal("open GC context must keep the flags and disable idle")
	}
}
