package main

import "testing"

func TestChannelsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range channels {
		if c.id == "" || c.name == "" {
			t.Errorf("channel with empty id or name: %+v", c)
		}
		if seen[c.id] {
			t.Errorf("duplicate channel id %q", c.id)
		}
		seen[c.id] = true
		if c.dm && c.who == nil {
			t.Errorf("direct message %q has no person", c.id)
		}
	}
	for _, g := range groups[1:] {
		found := false
		for _, c := range channels {
			if c.group == g {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("group %q has no channels", g)
		}
	}
}

func TestChannelTitle(t *testing.T) {
	if got := channelTitle(channels[2]); got != "# announcements" {
		t.Errorf("channel title = %q, want # announcements", got)
	}
	dm := channel{id: "x", name: "Maya Chen", dm: true}
	if got := channelTitle(dm); got != "Maya Chen" {
		t.Errorf("dm title = %q, want Maya Chen", got)
	}
}

func TestInitialFeed(t *testing.T) {
	f := initialFeed()
	if len(f) < 2 {
		t.Fatalf("feed has %d messages, want at least 2", len(f))
	}
	if split < 0 || split >= len(f) {
		t.Fatalf("split = %d outside feed of %d", split, len(f))
	}
	for i, m := range f {
		if m.who.name == "" || len(m.paras) == 0 {
			t.Errorf("message %d has no author or no text: %+v", i, m)
		}
	}
}
