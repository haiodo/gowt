package synth

import (
	"runtime"
	"testing"
)

func TestFootprint(t *testing.T) {
	for _, n := range []int{10000, 50000} {
		var a, b runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&a)
		d := New(n, 1)
		runtime.GC()
		runtime.ReadMemStats(&b)
		live := float64(b.HeapAlloc - a.HeapAlloc)
		t.Logf("%d boxes, %d connectors: %.0f MB live (%.0f B per node)", n, len(d.Conns), live/1e6, live/float64(n+len(d.Conns)))
		runtime.KeepAlive(d)
	}
}
