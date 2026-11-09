//go:build windows

package d2d

import (
	"syscall"
	"testing"
)

// BenchmarkCallSysMetrics measures the COM-call path (call) against a real, safe export:
// user32 GetSystemMetrics(SM_CXSCREEN) needs no window. Run with and without -tags gowtcgo
// (CGO_ENABLED=1) to compare the default syscall.SyscallN packing with the direct cgo
// dispatcher:
//
//	go test ./gfx/d2d -run=^$ -bench=CallSysMetrics
//	go test ./gfx/d2d -run=^$ -bench=CallSysMetrics -tags gowtcgo
func BenchmarkCallSysMetrics(b *testing.B) {
	p := syscall.NewLazyDLL("user32.dll").NewProc("GetSystemMetrics")
	if err := p.Find(); err != nil {
		b.Skip(err)
	}
	fn := p.Addr()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(fn, ui(0))
	}
}
