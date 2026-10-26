//go:build darwin

// Hand-written: grapheme cluster boundaries for TextLayout cluster movement.
package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	graphemeOnce sync.Once
	graphemeFn   uintptr
)

const kCFStringGraphemeCluster = 1

// GraphemeRange returns the grapheme cluster of s that contains the UTF-16 index idx, ZWJ emoji
// sequences included. It needs an autorelease pool on the calling thread.
func GraphemeRange(s string, idx int) (loc, n int) {
	graphemeOnce.Do(func() {
		objcInit()
		graphemeFn, _ = purego.Dlsym(purego.RTLD_DEFAULT, "CFStringGetRangeOfCharacterClusterAtIndex")
	})
	// CFRange is two words: it comes back in the first two result registers.
	l, c, _ := purego.SyscallN(graphemeFn, nsString(s), uintptr(idx), kCFStringGraphemeCluster)
	return int(l), int(c)
}
