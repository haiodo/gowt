package jrt

import (
	"sync"
	"testing"
)

func TestGoidMatchesSlow(t *testing.T) {
	if goid() != goidSlow() {
		t.Fatal("main goroutine")
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if goid() != goidSlow() {
				t.Error("goroutine mismatch")
			}
		}()
	}
	wg.Wait()
}

func BenchmarkCurrentThread(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CurrentThread()
	}
}

func BenchmarkGoidSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		goidSlow()
	}
}
