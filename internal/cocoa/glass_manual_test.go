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
			host := backdropHost(view)
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
