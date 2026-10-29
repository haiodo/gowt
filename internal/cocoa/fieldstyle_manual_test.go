//go:build darwin

package cocoa

import (
	"math"
	"testing"

	"github.com/ebitengine/purego"
)

// SetFieldFill takes the bezel off, moves the background colour to a layer with the radius, and
// SetFieldInset moves the text in by the same amount on both sides.
func TestFieldFillAndInset(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		plain := affixField(12, "120", false)
		defer msg(plain, "release")
		f := affixField(12, "120", false)
		defer msg(f, "release")
		msg(f, "setBackgroundColor:", msg(class("NSColor"), "controlBackgroundColor"))
		view := NewNSViewOverload1(int64(f))
		SetFieldFill(view, 6)
		SetFieldInset(view, 8)
		defer ReleaseFieldAffix(view)

		if msg(f, "isBezeled")&0xff != 0 || msg(f, "drawsBackground")&0xff != 0 {
			t.Error("a filled field keeps its bezel or draws its own background")
		}
		if r := FieldFillRadius(view); r != 6 {
			t.Errorf("corner radius %v, want 6", r)
		}
		if c, ok := FieldFillRGBA(view); !ok || c[3] < 0.5 || c[0]+c[1]+c[2] == 0 {
			t.Errorf("layer fill %v (ok %v), want the field's background colour", c, ok)
		}
		if in := FieldInset(view); in != 8 {
			t.Errorf("inset %v, want 8", in)
		}

		var drawingRect func(self, sel uintptr, r NSRect) NSRect
		purego.RegisterFunc(&drawingRect, objcMsgSend)
		b := affixBounds(f, sel("bounds"))
		cell, plainCell := msg(f, "cell"), msg(plain, "cell")
		got := drawingRect(cell, sel("drawingRectForBounds:"), b)
		want := superDrawingRect(plainCell, sel("drawingRectForBounds:"), b)
		if math.Abs(got.X-(want.X+8)) > 0.5 || math.Abs(got.Width-(want.Width-16)) > 0.5 {
			t.Errorf("drawing rect %+v, want %+v moved by 8 and narrowed by 16", got, want)
		}
		titleRect := drawingRect(cell, sel("titleRectForBounds:"), b)
		if titleRect.Y <= 0 || titleRect.Height >= b.Height {
			t.Errorf("title rect %+v, want it centred on the line of the %v field", titleRect, b)
		}
		if math.Abs(titleRect.Y-(b.Height-titleRect.Height)/2) > 0.5 {
			t.Errorf("title rect y %.1f, want %.1f", titleRect.Y, (b.Height-titleRect.Height)/2)
		}
	})
}
