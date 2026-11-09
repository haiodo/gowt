//go:build linux

package cairo

import "testing"

// BenchmarkNativeCallMoveToLineTo measures the hot render path through the active
// backend: purego by default, the typed C wrappers with -tags gowtcgo
// (CGO_ENABLED=1). Run both builds side by side to compare.
func BenchmarkNativeCallMoveToLineTo(b *testing.B) {
	if err := bind(); err != nil {
		b.Skipf("cairo libraries are not loaded: %v", err)
	}
	surf := imageSurfaceCreate(formatARGB32, 64, 64)
	if surf == 0 {
		b.Fatal("cairo_image_surface_create returned nil")
	}
	defer surfaceDestroy(surf)
	cr := create(surf)
	if cr == 0 {
		b.Fatal("cairo_create returned nil")
	}
	defer destroy(cr)
	moveTo(cr, 1, 2) // bind everything, outside the timer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		moveTo(cr, 1, 2)
		lineTo(cr, 3, 4)
		newPath(cr)
	}
}
