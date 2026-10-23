package qatree

import (
	"strings"
	"testing"
)

func leaf(class, id, text, value string, kids ...*entry) *entry {
	return &entry{n: Node{Class: class, ID: id, Text: text, Value: value, Path: id}, kids: kids}
}

func TestResolve(t *testing.T) {
	ok := leaf("Button", "ok", "OK", "")
	tree := []*entry{
		leaf("Shell", "", "main", "", ok, leaf("Text", "name", "hint", "bob"),
			leaf("Composite", "box", "", "", leaf("Button", "ok2", "OK", ""))),
	}
	for _, c := range []struct {
		s    Selector
		want int
	}{
		{Selector{Class: "Button"}, 2},
		{Selector{Text: "OK"}, 2},
		{Selector{ID: "ok"}, 1},
		{Selector{Text: "bob"}, 1},
		{Selector{Text: "~ob"}, 1},
		{Selector{Text: "~K", Within: &Selector{ID: "box"}}, 1},
		{Selector{Text: "ok"}, 0},
	} {
		if got, err := resolve(tree, c.s); err != nil || len(got) != c.want {
			t.Errorf("%v: %d entries, err %v, want %d", c.s, len(got), err, c.want)
		}
	}
	_, err := resolve(tree, Selector{Text: "OK", Within: &Selector{Class: "Button"}})
	if err == nil || !strings.Contains(err.Error(), "matches 2 controls") {
		t.Errorf("within of two: %v", err)
	}
	// Within excludes the control itself.
	if got, _ := resolve(tree, Selector{ID: "box", Within: &Selector{ID: "box"}}); len(got) != 0 {
		t.Errorf("within itself: %d", len(got))
	}
}

func TestRows(t *testing.T) {
	a, b := leaf("TreeItem", "a", "a", ""), leaf("TreeItem", "b", "b", "")
	a.kids = []*entry{leaf("TreeItem", "c", "c", "")}
	tree := leaf("Tree", "t", "", "", a, b, leaf("Button", "x", "x", ""))
	var ids []string
	for _, r := range rows(tree) {
		ids = append(ids, r.n.ID)
	}
	if strings.Join(ids, "") != "acb" {
		t.Errorf("pre-order %v", ids)
	}
}
