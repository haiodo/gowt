//go:build darwin

package cocoa

import (
	"os"
	"sync"
	"testing"
)

// Guards the helper every generated native's lazy binding lowers to (NativeEmitter):
// the first call must resolve a real libSystem symbol through RTLD_DEFAULT and be callable.
func TestLazyFuncRealNative(t *testing.T) {
	var getpid lazyFunc[func() int32]
	if got, want := getpid.get("getpid")(), int32(os.Getpid()); got != want {
		t.Fatalf("getpid() = %d, want %d", got, want)
	}
}

// Concurrent first calls must all observe the one registration (sync.Once inside get),
// not race or re-bind.
func TestLazyFuncConcurrentFirstCall(t *testing.T) {
	var getppid lazyFunc[func() int32]
	const n = 32
	pids := make([]int32, n)
	var wg sync.WaitGroup
	for i := range pids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pids[i] = getppid.get("getppid")()
		}()
	}
	wg.Wait()
	want := int32(os.Getppid())
	for i, got := range pids {
		if got != want {
			t.Fatalf("goroutine %d: getppid() = %d, want %d", i, got, want)
		}
	}
}
