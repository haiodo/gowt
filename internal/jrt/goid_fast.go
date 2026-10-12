//go:build amd64 || arm64

package jrt

import "sync"

func gword(off uintptr) uint64

// goidOffset is the byte offset of runtime.g.goid, found at first use by comparing the slow goid
// of several live goroutines with the words of their g; -1 when it is not unique (slow path).
var (
	goidOnce   sync.Once
	goidOffset = -1
)

func findGoidOffset() {
	const samples, words = 4, 64
	var ws [samples][words]uint64
	var ids [samples]uint64
	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i] = goidSlow()
			for w := range ws[i] {
				ws[i][w] = gword(uintptr(w) * 8)
			}
		}()
	}
	wg.Wait()
	cand := -1
	for w := 0; w < words; w++ {
		ok := true
		for i := range ids {
			if ws[i][w] != ids[i] {
				ok = false
				break
			}
		}
		if ok {
			if cand >= 0 {
				return
			}
			cand = w
		}
	}
	if cand >= 0 {
		goidOffset = cand * 8
	}
}

// goid reads the goroutine id straight from the runtime's g (no stack walk).
func goid() uint64 {
	goidOnce.Do(findGoidOffset)
	if goidOffset < 0 {
		return goidSlow()
	}
	return gword(uintptr(goidOffset))
}
