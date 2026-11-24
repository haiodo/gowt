package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func TestWidgetFontLoadedOnDemand(t *testing.T) {
	onUI(t, func(t *uiT) {
		display := NewDisplay()
		defer display.Dispose()
		if display.boxFont != nil {
			t.Fatalf("Display initialized the group font before creating a group")
		}
		shell := NewShellDisplay(display)
		group := NewGroup(shell, NONE)
		got := group.defaultNSFont_()
		want := display.GetFont(cocoa.OSClass_NSBox, cocoa.OSSel_titleFont)
		defer want.Release()
		if got.FontName().GetString() != want.FontName().GetString() || got.PointSize() != want.PointSize() {
			t.Errorf("group default font = %s %g, want %s %g", got.FontName().GetString(), got.PointSize(), want.FontName().GetString(), want.PointSize())
		}
		if got != display.boxFont || got != group.defaultNSFont_() {
			t.Errorf("group did not reuse its cached native font")
		}
	})
}
