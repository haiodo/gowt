package shot

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// TestNoFocusRings checks that the ring type is none on a nested button and on its cell.
func TestNoFocusRings(t *testing.T) {
	sel := objc.RegisterName
	cls := func(n string) objc.ID { return objc.ID(objc.GetClass(n)) }
	root := cls("NSView").Send(sel("alloc")).Send(sel("initWithFrame:"), nsRect{W: 100, H: 100})
	btn := cls("NSButton").Send(sel("alloc")).Send(sel("initWithFrame:"), nsRect{W: 50, H: 20})
	box := cls("NSView").Send(sel("alloc")).Send(sel("initWithFrame:"), nsRect{W: 100, H: 100})
	root.Send(sel("addSubview:"), box)
	box.Send(sel("addSubview:"), btn)
	noFocusRings(root)
	if got := objc.Send[uint64](btn, sel("focusRingType")); got != 1 {
		t.Errorf("button focusRingType = %d, want 1", got)
	}
	if got := objc.Send[uint64](btn.Send(sel("cell")), sel("focusRingType")); got != 1 {
		t.Errorf("cell focusRingType = %d, want 1", got)
	}
}
