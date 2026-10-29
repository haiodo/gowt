//go:build darwin

package cocoa

import "testing"

func TestRequireCompatibleLook(t *testing.T) {
	t.Logf("glass available: %v", GlassAvailable())
	onMain(func() {
		RequireCompatibleLook()
		if !CompatibleLookKey() {
			t.Error("UIDesignRequiresCompatibility not readable from the main bundle after RequireCompatibleLook")
		}
	})
}

func TestInstallBackdropReplaces(t *testing.T) {
	onMain(func() {
		for _, cls := range []string{"NSView", "NSBox"} {
			v := msg(msg(class(cls), "alloc"), "init")
			view := NewNSViewOverload1(int64(v))
			host := msg(msg(class("NSView"), "alloc"), "init")
			msg(host, "addSubview:", v)
			count := func() uintptr { return msg(msg(host, "subviews"), "count") }
			base := count()
			if InstallBackdrop(view, BackdropGlass, false) {
				t.Errorf("%s: first install reported a replaced backdrop", cls)
			}
			if !InstallBackdrop(view, BackdropGlass, false) || count() != base+1 {
				t.Errorf("%s: second install must replace, got %d subviews over %d", cls, count(), base)
			}
			if !InstallBackdrop(view, BackdropNone, false) || count() != base {
				t.Errorf("%s: None must remove the backdrop", cls)
			}
		}
	})
}

// Under Reduce transparency a Glass backdrop is the NSVisualEffectView material, also for a panel.
func TestGlassBackdropReduceTransparency(t *testing.T) {
	old := reduceTransparency
	reduceTransparency = func() bool { return true }
	defer func() { reduceTransparency = old }()
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		for _, panel := range []bool{false, true} {
			installBackdrop(view, BackdropGlass, false, panel)
			if name, _, _ := BackdropOf(view); name != "NSVisualEffectView" {
				t.Errorf("panel %v: backdrop %q, want NSVisualEffectView", panel, name)
			}
		}
	})
}

func TestPanelBackdropIsLowestChild(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("no NSGlassEffectView")
	}
	var reduce bool
	onMain(func() { reduce = ReduceTransparency() })
	if reduce {
		// Under the setting the panel backdrop is the NSVisualEffectView sibling of
		// TestGlassBackdropReduceTransparency, not a child of the panel.
		t.Skip("Reduce transparency: no panel glass")
	}
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		InstallPanelBackdrop(view, BackdropGlass)
		if n := msg(msg(host, "subviews"), "count"); n != 1 {
			t.Errorf("the glass must not be a sibling, host has %d subviews", n)
		}
		subs := msg(v, "subviews")
		if n := msg(subs, "count"); n == 0 {
			t.Fatal("the panel has no glass child")
		}
		bd := msg(subs, "objectAtIndex:", 0)
		child := msg(msg(class("NSView"), "alloc"), "init")
		msg(v, "addSubview:positioned:relativeTo:", child, windowBelowArg, 0)
		KeepPanelBackdropBelow(view)
		if msg(msg(v, "subviews"), "objectAtIndex:", 0) != bd {
			t.Error("the glass must stay below a child added later")
		}
		if !InstallPanelBackdrop(view, BackdropNone) || msg(msg(v, "subviews"), "count") != 1 {
			t.Error("None must remove the glass")
		}
	})
}
