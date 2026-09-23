package cocoa

import "testing"

// Guards the constant-global accessor codegen (Emitter.emitConstantAccessor): a Dlsym'd C
// global's address, not a callable function - checks one resolves to a real NSString.
func TestConstantAccessorNSDefaultRunLoopMode(t *testing.T) {
	if OSNSDefaultRunLoopMode_ == nil || OSNSDefaultRunLoopMode_.Id == 0 {
		t.Fatal("NSDefaultRunLoopMode did not resolve to a non-zero NSString")
	}
	addr := OSNSDefaultRunLoopMode_.UTF8String()
	if addr == 0 {
		t.Fatal("UTF8String() returned nil")
	}
	n := CStrlen(addr)
	buf := make([]int8, n)
	CMemmoveOverload8(buf, addr, int64(n))
	got := make([]byte, n)
	for i, b := range buf {
		got[i] = byte(b)
	}
	if want := "kCFRunLoopDefaultMode"; string(got) != want {
		t.Fatalf("NSDefaultRunLoopMode.UTF8String() = %q, want %q", got, want)
	}
}
