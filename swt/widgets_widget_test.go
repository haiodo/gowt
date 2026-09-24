// Hand-written: exercises Round 4's Widget.java translation - WidgetCheckBits (the mutually-
// exclusive style-bit resolver every checkOrientation/checkAlignment-style setter routes
// through) needs no Display, just the style ints.
package swt

import "testing"

func TestWidgetCheckBits(t *testing.T) {
	mask := LEFT_TO_RIGHT | RIGHT_TO_LEFT

	// Neither bit set: defaults to the first (int0) argument.
	got := WidgetCheckBits(0, LEFT_TO_RIGHT, RIGHT_TO_LEFT, 0, 0, 0, 0)
	if got&mask != LEFT_TO_RIGHT {
		t.Fatalf("no bits set: got %#x, want LEFT_TO_RIGHT set", got)
	}

	// Both bits set (conflicting style): resolves to the first one in argument order.
	got = WidgetCheckBits(LEFT_TO_RIGHT|RIGHT_TO_LEFT, LEFT_TO_RIGHT, RIGHT_TO_LEFT, 0, 0, 0, 0)
	if got&mask != LEFT_TO_RIGHT {
		t.Fatalf("both bits set: got %#x, want only LEFT_TO_RIGHT", got)
	}

	// Only the second bit set: kept as-is, first bit not forced on.
	got = WidgetCheckBits(RIGHT_TO_LEFT, LEFT_TO_RIGHT, RIGHT_TO_LEFT, 0, 0, 0, 0)
	if got&mask != RIGHT_TO_LEFT {
		t.Fatalf("only RIGHT_TO_LEFT set: got %#x, want RIGHT_TO_LEFT set", got)
	}

	// Bits outside the mask pass through untouched.
	extra := int32(1 << 20)
	got = WidgetCheckBits(extra, LEFT_TO_RIGHT, RIGHT_TO_LEFT, 0, 0, 0, 0)
	if got&extra == 0 {
		t.Fatalf("bit outside mask was dropped: got %#x", got)
	}
}
