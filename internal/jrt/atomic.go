package jrt

import "sync"

// AtomicBoolean, AtomicInteger and AtomicReference are java.util.concurrent.atomic's cells; the
// constructors take Java's optional initial value.
type AtomicBoolean struct {
	mu sync.Mutex
	v  bool
}

func NewAtomicBoolean(initial ...bool) *AtomicBoolean {
	a := &AtomicBoolean{}
	if len(initial) > 0 {
		a.v = initial[0]
	}
	return a
}

func (a *AtomicBoolean) Get() bool       { a.mu.Lock(); defer a.mu.Unlock(); return a.v }
func (a *AtomicBoolean) GetPlain() bool  { return a.Get() }
func (a *AtomicBoolean) Set(v bool)      { a.mu.Lock(); a.v = v; a.mu.Unlock() }
func (a *AtomicBoolean) SetPlain(v bool) { a.Set(v) }
func (a *AtomicBoolean) GetAndSet(v bool) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.v
	a.v = v
	return old
}
func (a *AtomicBoolean) CompareAndSet(expected, v bool) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.v != expected {
		return false
	}
	a.v = v
	return true
}

type AtomicInteger struct {
	mu sync.Mutex
	v  int32
}

func NewAtomicInteger(initial ...int32) *AtomicInteger {
	a := &AtomicInteger{}
	if len(initial) > 0 {
		a.v = initial[0]
	}
	return a
}

func (a *AtomicInteger) Get() int32       { a.mu.Lock(); defer a.mu.Unlock(); return a.v }
func (a *AtomicInteger) GetPlain() int32  { return a.Get() }
func (a *AtomicInteger) IntValue() int32  { return a.Get() }
func (a *AtomicInteger) Set(v int32)      { a.mu.Lock(); a.v = v; a.mu.Unlock() }
func (a *AtomicInteger) SetPlain(v int32) { a.Set(v) }
func (a *AtomicInteger) AddAndGet(d int32) int32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.v += d
	return a.v
}
func (a *AtomicInteger) GetAndAdd(d int32) int32 { return a.AddAndGet(d) - d }
func (a *AtomicInteger) IncrementAndGet() int32  { return a.AddAndGet(1) }
func (a *AtomicInteger) GetAndIncrement() int32  { return a.AddAndGet(1) - 1 }
func (a *AtomicInteger) DecrementAndGet() int32  { return a.AddAndGet(-1) }
func (a *AtomicInteger) GetAndSet(v int32) int32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.v
	a.v = v
	return old
}
func (a *AtomicInteger) CompareAndSet(expected, v int32) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.v != expected {
		return false
	}
	a.v = v
	return true
}

type AtomicReference struct {
	mu sync.Mutex
	v  any
}

func NewAtomicReference(initial ...any) *AtomicReference {
	a := &AtomicReference{}
	if len(initial) > 0 {
		a.v = initial[0]
	}
	return a
}

func (a *AtomicReference) Get() any       { a.mu.Lock(); defer a.mu.Unlock(); return a.v }
func (a *AtomicReference) GetPlain() any  { return a.Get() }
func (a *AtomicReference) Set(v any)      { a.mu.Lock(); a.v = v; a.mu.Unlock() }
func (a *AtomicReference) SetPlain(v any) { a.Set(v) }
func (a *AtomicReference) GetAndSet(v any) any {
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.v
	a.v = v
	return old
}
