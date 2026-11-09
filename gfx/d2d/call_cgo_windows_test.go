//go:build windows && cgo && gowtcgo

package d2d

import (
	"math"
	"testing"
)

// Runtime check of the cgo dispatcher's argument packing: the C stubs behind testAddr*
// (call_cgostub_windows.go) return the bit pattern of a sum computed from the float
// arguments, so a misaligned float register (f[1] instead of f[0] and so on) fails the
// exact comparison.

func TestCgoCallFloatArgs(t *testing.T) {
	got := math.Float64frombits(uint64(call(testAddrSumFF(), ui(0), fl(1.5), fl(2.25))))
	if got != 3.75 {
		t.Fatalf("I,F,F: got %v, want 3.75", got)
	}
}

func TestCgoCallMixedArgs(t *testing.T) {
	got := math.Float64frombits(uint64(call(testAddrMixed(), ui(0), fl(1.5), ui(100), fl(0.25), ui(7))))
	if got != 122.25 {
		t.Fatalf("I,F,I,F,I: got %v, want 122.25", got)
	}
}

func TestCgoCallPointArg(t *testing.T) {
	got := math.Float64frombits(uint64(call(testAddrPt(), ui(0), pt2(1.5, 2.5))))
	if got != 4 {
		t.Fatalf("I,P: got %v, want 4", got)
	}
}
