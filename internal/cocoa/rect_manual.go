// os.c wraps NSIntersectionRect and CGDisplayBounds, which take/return their rect by value, as
// void functions writing through a pointer, and PtInRgn, which takes a Carbon Point by value, with a
// short[]. Binding the literal symbol with those pointer arguments (the generic native binding)
// never wrote the result / passed a wrong Point: a GC's clipping, Canvas/Table/Tree rects and
// Region.contains were wrong.
package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	nsIntersectionRect func(a, b NSRect) NSRect
	cgDisplayBounds    func(display uint32) CGRect
	ptInRgn            func(pt carbonPoint, rgn uintptr) bool
	rectOnce           sync.Once
)

// Carbon's Point: vertical first.
type carbonPoint struct{ V, H int16 }

func bindRectFuncs() {
	rectOnce.Do(func() {
		ensureFrameworks()
		purego.RegisterLibFunc(&nsIntersectionRect, purego.RTLD_DEFAULT, "NSIntersectionRect")
		purego.RegisterLibFunc(&cgDisplayBounds, purego.RTLD_DEFAULT, "CGDisplayBounds")
		purego.RegisterLibFunc(&ptInRgn, purego.RTLD_DEFAULT, "PtInRgn")
	})
}

func OSNSIntersectionRect(result *NSRect, aRect *NSRect, bRect *NSRect) {
	bindRectFuncs()
	*result = nsIntersectionRect(*aRect, *bRect)
}

func OSCGDisplayBounds(display int32, rect *CGRect) {
	bindRectFuncs()
	*rect = cgDisplayBounds(uint32(display))
}

func OSPtInRgn(pt []int16, rgnHandle int64) bool {
	bindRectFuncs()
	return ptInRgn(carbonPoint{pt[0], pt[1]}, uintptr(rgnHandle))
}
