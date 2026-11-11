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

// Without a tint the backdrop is exactly as before: no stored tint, the previous material.
func TestBackdropTintDefaultUnchanged(t *testing.T) {
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		InstallBackdrop(view, BackdropGlass, false)
		if _, ok := BackdropTint(view); ok {
			t.Error("a backdrop without SetGlassTint must not report a tint")
		}
		if name, material, _ := BackdropOf(view); name == "NSVisualEffectView" && material != materialUnderWindowBG {
			t.Errorf("default material %d, want %d", material, materialUnderWindowBG)
		}
	})
}

// A tint reaches NSGlassEffectView tintColor as the exact sRGB colour and survives a reinstall;
// on a system without glass the fallback material is the denser one.
func TestBackdropTintApplied(t *testing.T) {
	const (
		r, g, b, a = 200.0 / 255, 100.0 / 255, 50.0 / 255, 178.0 / 255
	)
	if ReduceTransparency() {
		t.Skip("Reduce transparency: covered by TestBackdropTintReduceTransparency")
	}
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		SetBackdropTint(view, true, r, g, b, a)
		for i := 0; i < 2; i++ {
			InstallBackdrop(view, BackdropGlass, false)
			rgba, ok := BackdropTint(view)
			if !ok {
				t.Fatalf("install %d: tint lost", i)
			}
			name, material, _ := BackdropOf(view)
			switch {
			case GlassAvailable():
				if name != "NSGlassEffectView" {
					t.Fatalf("install %d: backdrop %q, want NSGlassEffectView", i, name)
				}
				for j, want := range []float64{r, g, b, a} {
					if d := rgba[j] - want; d > 1e-4 || d < -1e-4 {
						t.Errorf("install %d: tintColor component %d = %v, want %v", i, j, rgba[j], want)
					}
				}
			default:
				if name != "NSVisualEffectView" || material != materialUnderPageBackground {
					t.Errorf("install %d: fallback %q material %d, want NSVisualEffectView %d", i, name, material, materialUnderPageBackground)
				}
			}
		}
	})
}

// off removes the tint from the view itself, not only from the storage: tintColor goes back to
// nil on the glass, and setting the tint again restores it.
func TestBackdropTintOffRemovesTintColor(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("needs NSGlassEffectView")
	}
	if ReduceTransparency() {
		t.Skip("Reduce transparency: no glass view")
	}
	const (
		r, g, b, a = 90.0 / 255, 130.0 / 255, 220.0 / 255, 255.0 / 255
	)
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		InstallBackdrop(view, BackdropGlass, false)
		SetBackdropTint(view, true, r, g, b, a)
		SetBackdropTint(view, false, 0, 0, 0, 0)
		if _, ok := BackdropTint(view); ok {
			t.Error("off must clear the stored tint")
		}
		bd := backdropOf(v)
		if c := msg(bd, "tintColor"); c != 0 {
			t.Errorf("off: tintColor still set on the glass view")
		}
		SetBackdropTint(view, true, r, g, b, a)
		rgba, ok := BackdropTint(view)
		if !ok {
			t.Fatal("re-on: stored tint lost")
		}
		for j, want := range []float64{r, g, b, a} {
			if d := rgba[j] - want; d > 1e-4 || d < -1e-4 {
				t.Errorf("re-on: tintColor component %d = %v, want %v", j, rgba[j], want)
			}
		}
	})
}

// Under Reduce transparency a tinted glass backdrop is the NSVisualEffectView with the previous
// material: the tint must not break the fallback.
func TestBackdropTintReduceTransparency(t *testing.T) {
	old := reduceTransparency
	reduceTransparency = func() bool { return true }
	defer func() { reduceTransparency = old }()
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		SetBackdropTint(view, true, 0.5, 0.5, 0.5, 0.7)
		InstallBackdrop(view, BackdropGlass, false)
		if _, ok := BackdropTint(view); !ok {
			t.Error("stored tint lost")
		}
		if name, material, _ := BackdropOf(view); name != "NSVisualEffectView" || material != materialUnderWindowBG {
			t.Errorf("reduce transparency: %q material %d, want NSVisualEffectView %d", name, material, materialUnderWindowBG)
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
