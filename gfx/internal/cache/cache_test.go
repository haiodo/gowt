package cache

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestGenBounded(t *testing.T) {
	c := Gen[int, int]{Max: 100}
	for i := range 1000 {
		c.LoadOrStore(i, i)
		if _, ok := c.Load(0); !ok { // used every round: never dropped
			t.Fatalf("key 0 dropped after %d inserts", i)
		}
	}
	if n := c.Len(); n > 2*c.Max {
		t.Fatalf("Len %d, want at most %d", n, 2*c.Max)
	}
	if _, ok := c.Load(500); ok {
		t.Fatal("key 500 survived 499 newer inserts")
	}
	if v, ok := c.Load(999); !ok || v != 999 {
		t.Fatalf("Load(999) = %d, %v", v, ok)
	}
}

func TestWeakReleases(t *testing.T) {
	type raster struct{ pix []byte }
	var c Weak[raster, int]
	var released atomic.Int32
	release := func(int) { released.Add(1) }
	keep := &raster{make([]byte, 64)}
	c.LoadOrStore(keep, -1, release)
	for i := range 100 {
		r := &raster{make([]byte, 64)}
		c.LoadOrStore(r, i, release)
		if v, ok := c.Load(r); !ok || v != i {
			t.Fatalf("Load = %d, %v", v, ok)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for released.Load() < 100 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if n := released.Load(); n != 100 {
		t.Fatalf("released %d of 100 dropped keys", n)
	}
	if v, ok := c.Load(keep); !ok || v != -1 || c.Len() != 1 {
		t.Fatalf("live key: Load = %d, %v, Len %d", v, ok, c.Len())
	}
	c.Drain(release)
	runtime.KeepAlive(keep)
	keep = nil
	for range 5 {
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if n := released.Load(); n != 101 {
		t.Fatalf("released %d after Drain, want 101: the cleanup must not release twice", n)
	}
}
