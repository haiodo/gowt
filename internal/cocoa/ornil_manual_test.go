//go:build darwin

package cocoa

import "testing"

func TestOrNil(t *testing.T) {
	calls := 0
	ctor := func(h int64) *id { calls++; return &id{Id: h} }
	if got := orNil(0, ctor); got != nil || calls != 0 {
		t.Fatalf("orNil(0) = %v with %d calls; want nil, 0 calls", got, calls)
	}
	if got := orNil(7, ctor); got == nil || got.Id != 7 || calls != 1 {
		t.Fatalf("orNil(7) = %v with %d calls; want &id{Id:7}, 1 call", got, calls)
	}
}
