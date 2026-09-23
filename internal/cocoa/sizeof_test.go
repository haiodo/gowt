package cocoa

import "testing"

// Guards the deferred-static-init wiring (func init() calling the sizeof_manual.go shims):
// if a name or ordering regresses, these come back 0 instead of panicking on import.
func TestSizeofValues(t *testing.T) {
	cases := []struct {
		name string
		got  int32
		want int32
	}{
		{"NSPoint", NSPointSizeof, 16},
		{"NSSize", NSSizeSizeof, 16},
		{"NSRect", NSRectSizeof, 32},
		{"NSRange", NSRangeSizeof, 16},
		{"CGPoint", CGPointSizeof, 16},
		{"CGSize", CGSizeSizeof, 16},
		{"CGRect", CGRectSizeof, 32},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%sSizeof = %d, want %d", c.name, c.got, c.want)
		}
	}
}
