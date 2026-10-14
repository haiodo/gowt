//go:build darwin

package cocoa

import "testing"

func TestInstallCard(t *testing.T) {
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		view := NewNSViewOverload1(int64(v))
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		count := func() uintptr { return msg(msg(host, "subviews"), "count") }
		InstallCard(view, 12, true)
		InstallCard(view, 12, true)
		if !HasCard(view) || count() != 2 {
			t.Fatalf("second install must replace the card: %d subviews", count())
		}
		if msg(msg(host, "subviews"), "objectAtIndex:", 0) == v {
			t.Error("the card must lie below the view")
		}
		SetCardHidden(view, true)
		if c := msg(msg(host, "subviews"), "objectAtIndex:", 0); msg(c, "isHidden")&0xff == 0 {
			t.Error("SetCardHidden did not hide the card")
		}
		InstallCard(view, 0, false)
		if HasCard(view) || count() != 1 {
			t.Errorf("remove left %d subviews", count())
		}
	})
}

func TestToolbarClicked(t *testing.T) {
	var got []string
	s := &toolbarState{items: []ToolbarItem{
		{Label: "plain", Click: func() { got = append(got, "plain") }},
		{Label: "nav", Group: []ToolbarItem{
			{Label: "back", Click: func() { got = append(got, "back") }},
			{Label: "forward", Click: func() { got = append(got, "forward") }},
		}},
		{Label: "find", Search: func(string) {}},
	}}
	s.clicked(itemIDPrefix+"0", -1)
	s.clicked(itemIDPrefix+"1", 1)
	s.clicked(itemIDPrefix+"1", -1)
	s.clicked(itemIDPrefix+"1", 2)
	s.clicked(itemIDPrefix+"2", -1)
	s.clicked(sidebarSpacerID, -1)
	if len(got) != 2 || got[0] != "plain" || got[1] != "forward" {
		t.Errorf("clicks %v", got)
	}
}

func TestToolbarSidebarSpacer(t *testing.T) {
	onMain(func() {
		s := &toolbarState{items: []ToolbarItem{{Label: "a"}, {Space: true}}}
		if n := msg(s.ids(), "count"); n != 2 {
			t.Errorf("without a sidebar: %d ids", n)
		}
		s.sidebar = true
		ids := s.ids()
		if n := msg(ids, "count"); n != 3 || goString(msg(ids, "objectAtIndex:", 0)) != sidebarSpacerID {
			t.Errorf("with a sidebar: %d ids, first %q", n, goString(msg(ids, "objectAtIndex:", 0)))
		}
	})
}

func TestToolbarSpacerItem(t *testing.T) {
	onMain(func() {
		s := &toolbarState{sidebar: true}
		ti := s.item(0, sidebarSpacerID)
		if ti == 0 || s.width == 0 || msg(ti, "view") != s.spacer || msg(ti, "isNavigational")&0xff == 0 {
			t.Fatalf("spacer item %x view %x width %x", ti, s.spacer, s.width)
		}
	})
}

func TestSourceListHooks(t *testing.T) {
	onMain(func() {
		tv := msg(msg(class("NSTableView"), "alloc"), "init")
		plain := msg(msg(class("NSTableView"), "alloc"), "init")
		SetSourceList(NewNSViewOverload1(int64(tv)))
		red := msg(class("NSColor"), "redColor")
		msg(tv, "setBackgroundColor:", red)
		msg(plain, "setBackgroundColor:", red)
		if msg(tv, "backgroundColor") == red {
			t.Error("a source list took an opaque background")
		}
		if msg(plain, "backgroundColor") != red {
			t.Error("the hook changed a table that is no source list")
		}
		if rv := msg(tv, "tableView:rowViewForRow:", tv, 0); rv == 0 || msg(rv, "isEmphasized")&0xff != 0 {
			t.Errorf("source-list row view %x", rv)
		}
		if rv := msg(plain, "tableView:rowViewForRow:", plain, 0); rv != 0 {
			t.Errorf("plain table got row view %x", rv)
		}
	})
}
