//go:build linux

package gtk

import "testing"

// BenchmarkNativeCallMatrixInitIdentity measures one bound native call through the
// active backend: purego by default, the typed C wrappers with -tags gowtcgo
// (CGO_ENABLED=1). Run both builds side by side to compare.
func BenchmarkNativeCallMatrixInitIdentity(b *testing.B) {
	if missing() {
		b.Skip("GTK is not loaded")
	}
	m := make([]float64, 6)
	CairoCairo_matrix_init_identity(m) // bind on first call, outside the timer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CairoCairo_matrix_init_identity(m)
	}
}
