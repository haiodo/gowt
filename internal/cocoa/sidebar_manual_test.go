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
	s.clicked(sidebarSeparatorID, -1)
	if len(got) != 2 || got[0] != "plain" || got[1] != "forward" {
		t.Errorf("clicks %v", got)
	}
}

func TestToolbarSidebarSeparator(t *testing.T) {
	onMain(func() {
		s := &toolbarState{items: []ToolbarItem{{Label: "a"}, {Space: true}}}
		if n := msg(s.ids(), "count"); n != 2 {
			t.Errorf("without a sidebar: %d ids", n)
		}
		s.split = 1 // any non-zero split view marks a sidebar; ids never sends to it
		ids := s.ids()
		if n := msg(ids, "count"); n != 3 || goString(msg(ids, "objectAtIndex:", 0)) != sidebarSeparatorID {
			t.Errorf("with a sidebar: %d ids, first %q", n, goString(msg(ids, "objectAtIndex:", 0)))
		}
	})
}
