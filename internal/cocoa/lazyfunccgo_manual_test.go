//go:build darwin && gowtcgo && cgo

package cocoa

import "testing"

// The generated registry must resolve real natives end to end: bind through get() by
// binding name and call through the typed C wrapper (malloc/free round trip).
func TestCgoRegistryRealNative(t *testing.T) {
	var malloc lazyFunc[func(int64) int64]
	var free lazyFunc[func(int64)]
	p := malloc.get("CMalloc", "ignored")(16)
	if p == 0 {
		t.Fatal("CMalloc(16) returned nil")
	}
	free.get("CFree", "ignored")(p)
}
