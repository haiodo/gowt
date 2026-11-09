//go:build windows

package win32

import "testing"

// BenchmarkNativeCall measures the generated native call path: a cheap no-arg kernel32 export
// and a gdi32 export with two int arguments, plus a COM vtable call shape. Run the same
// benchmark with and without -tags gowtcgo (CGO_ENABLED=1) to compare the default
// syscall.SyscallN backend against the direct cgo backend:
//
//	go test ./internal/win32 -run=^$ -bench=NativeCall
//	go test ./internal/win32 -run=^$ -bench=NativeCall -tags gowtcgo
func BenchmarkNativeCall(b *testing.B) {
	b.ReportAllocs()
	b.Run("NoArgs", func(b *testing.B) {
		var sink int32
		for i := 0; i < b.N; i++ {
			sink += OSGetCurrentProcessId()
		}
		_ = sink
	})
	b.Run("TwoArgs", func(b *testing.B) {
		var sink int32
		for i := 0; i < b.N; i++ {
			sink += OSGetSystemMetrics(0)
		}
		_ = sink
	})
}
