package qatree

import (
	"runtime"
	"strings"
	"testing"

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/swt"
)

func TestUnmnemonic(t *testing.T) {
	for in, want := range map[string]string{"&File": "File", "Sa&ve": "Save", "Fish && Chips": "Fish & Chips", "Open...": "Open...", "": ""} {
		if got := unmnemonic(in); got != want {
			t.Errorf("unmnemonic(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseMods(t *testing.T) {
	m, err := parseMods("Shift+Alt")
	if err != nil || !m.Shift || !m.Alt || m.Ctrl || m.Cmd {
		t.Errorf("Shift+Alt: %+v, %v", m, err)
	}
	if m, err := parseMods(""); err != nil || m != (qaview.Mods{}) {
		t.Errorf("empty: %+v, %v", m, err)
	}
	if _, err := parseMods("Hyper"); err == nil {
		t.Error("Hyper gave no error")
	}
}

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

func TestKeyEvent(t *testing.T) {
	mod := int32(swt.CTRL)
	if runtime.GOOS == "darwin" {
		mod = swt.COMMAND
	}
	for _, c := range []struct {
		keys        string
		state, code int32
		char        uint16
	}{
		{"Mod+K", mod, 'k', 0},
		{"Enter", 0, int32(swt.CR), uint16(swt.CR)},
		{"a", 0, 'a', 'a'},
		{"Shift+A", swt.SHIFT, 'a', 'A'},
		{"Ctrl+Alt+Left", swt.CTRL | swt.ALT, swt.ARROW_LEFT, 0},
		{"Mod++", mod, '+', 0},
		{"f5", 0, swt.F5, 0},
	} {
		state, code, char, err := keyEvent(c.keys)
		if err != nil || state != c.state || code != c.code || char != c.char {
			t.Errorf("keyEvent(%q) = %d, %d, %d, %v; want %d, %d, %d", c.keys, state, code, char, err, c.state, c.code, c.char)
		}
	}
	for _, keys := range []string{"Hyper+K", "Mod+Nope", ""} {
		if _, _, _, err := keyEvent(keys); err == nil {
			t.Errorf("keyEvent(%q) gave no error", keys)
		}
	}
}
