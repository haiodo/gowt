// Natives whose C function takes or returns a struct by value (os_custom.c's out-parameter
// wrappers, PtInRgn): dlsym'ing the JNI name and passing pointers, as a plain native does, calls them wrongly.
package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

// OSNSIntersectionRect is *result = NSIntersectionRect(*a, *b); result may alias a or b.
func OSNSIntersectionRect(result *NSRect, aRect *NSRect, bRect *NSRect) {
	a, b := *aRect, *bRect
	x1, y1 := max(a.X, b.X), max(a.Y, b.Y)
	x2, y2 := min(a.X+a.Width, b.X+b.Width), min(a.Y+a.Height, b.Y+b.Height)
	if x2 <= x1 || y2 <= y1 {
		*result = NSRect{}
		return
	}
	*result = NSRect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

var (
	cgDisplayBounds     func(display uint32) CGRect
	cgDisplayBoundsOnce sync.Once
)

// OSCGDisplayBounds is *rect = CGDisplayBounds(display).
func OSCGDisplayBounds(display int32, rect *CGRect) {
	cgDisplayBoundsOnce.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&cgDisplayBounds, purego.RTLD_DEFAULT, "CGDisplayBounds")
	})
	*rect = cgDisplayBounds(uint32(display))
}

var (
	ptInRgn     func(pt uint32, rgn int64) bool
	ptInRgnOnce sync.Once
)

// OSPtInRgn passes pt {v, h} as the two packed shorts of Carbon's Point, which PtInRgn takes by value.
func OSPtInRgn(pt []int16, rgnHandle int64) bool {
	ptInRgnOnce.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&ptInRgn, purego.RTLD_DEFAULT, "PtInRgn")
	})
	return ptInRgn(uint32(uint16(pt[0]))|uint32(uint16(pt[1]))<<16, rgnHandle)
}
