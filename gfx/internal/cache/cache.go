// Package cache holds the caches the gfx backends keep per Renderer, bounded so that a scene which
// keeps making new strings, shadows or Rasters does not grow them without end. All are safe for
// concurrent use; a value is never released while a draw on another goroutine may still use it.
package cache

import (
	"runtime"
	"sync"
	"sync/atomic"
	"weak"
)

// Gen keeps two generations of at most Max entries; a hit in the old one moves the entry to the
// current one, so what every frame draws survives while a frame adds fewer than Max new keys.
// A dropped value is only unreferenced: a native one is released by its runtime.AddCleanup.
type Gen[K comparable, V any] struct {
	Max    int
	g      atomic.Pointer[gens]
	misses atomic.Int64
}

type gens struct {
	cur, old *sync.Map
	n        atomic.Int64
}

func (c *Gen[K, V]) gens() *gens {
	if g := c.g.Load(); g != nil {
		return g
	}
	c.g.CompareAndSwap(nil, &gens{cur: new(sync.Map)})
	return c.g.Load()
}

// Load returns the value cached for k.
func (c *Gen[K, V]) Load(k K) (V, bool) {
	g := c.gens()
	if v, ok := g.cur.Load(k); ok {
		return v.(V), true
	}
	if g.old != nil {
		if v, ok := g.old.Load(k); ok {
			v, _ := c.store(g, k, v.(V))
			return v, true
		}
	}
	var zero V
	return zero, false
}

// LoadOrStore returns the value cached for k if there is one, else caches v; loaded is true when
// v was not stored and the caller still owns it.
func (c *Gen[K, V]) LoadOrStore(k K, v V) (actual V, loaded bool) {
	c.misses.Add(1)
	return c.store(c.gens(), k, v)
}

func (c *Gen[K, V]) store(g *gens, k K, v V) (V, bool) {
	a, loaded := g.cur.LoadOrStore(k, v)
	if !loaded && g.n.Add(1) == int64(c.Max) {
		c.g.CompareAndSwap(g, &gens{cur: new(sync.Map), old: g.cur})
	}
	return a.(V), loaded
}

// Clear drops every entry.
func (c *Gen[K, V]) Clear() { c.g.Store(nil) }

// Len counts the entries of both generations; one in both counts twice.
func (c *Gen[K, V]) Len() int {
	g := c.g.Load()
	if g == nil {
		return 0
	}
	n := 0
	for _, m := range []*sync.Map{g.cur, g.old} {
		if m != nil {
			m.Range(func(_, _ any) bool { n++; return true })
		}
	}
	return n
}

// Misses is the number of LoadOrStore calls: the values that had to be made.
func (c *Gen[K, V]) Misses() int64 { return c.misses.Load() }

// Weak maps a *K to a value without keeping the K alive; once the K is collected, release runs on
// the value on the cleanup goroutine, so a draw that uses the value keeps the K alive until done.
type Weak[K, V any] struct {
	m sync.Map // weak.Pointer[K] -> V
}

// Load returns the value stored for p.
func (c *Weak[K, V]) Load(p *K) (V, bool) {
	if v, ok := c.m.Load(weak.Make(p)); ok {
		return v.(V), true
	}
	var zero V
	return zero, false
}

// LoadOrStore returns the value stored for p if there is one, else stores v, to be released with
// release once p is unreachable; loaded is true when v was not stored and the caller still owns it.
func (c *Weak[K, V]) LoadOrStore(p *K, v V, release func(V)) (actual V, loaded bool) {
	k := weak.Make(p)
	a, loaded := c.m.LoadOrStore(k, v)
	if !loaded {
		runtime.AddCleanup(p, func(k weak.Pointer[K]) {
			if v, ok := c.m.LoadAndDelete(k); ok {
				release(v.(V))
			}
		}, k)
	}
	return a.(V), loaded
}

// Drain removes every entry and releases its value now; the cleanups of the keys find nothing left.
func (c *Weak[K, V]) Drain(release func(V)) {
	c.m.Range(func(k, _ any) bool {
		if v, ok := c.m.LoadAndDelete(k); ok {
			release(v.(V))
		}
		return true
	})
}

// Len is the number of entries.
func (c *Weak[K, V]) Len() int {
	n := 0
	c.m.Range(func(_, _ any) bool { n++; return true })
	return n
}
