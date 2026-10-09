package jrt

import (
	"fmt"
	"strings"
	"sync"
)

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

type AtomicLong struct {
	mu sync.Mutex
	v  int64
}

func NewAtomicLong(initial ...int64) *AtomicLong {
	a := &AtomicLong{}
	if len(initial) > 0 {
		a.v = initial[0]
	}
	return a
}

func (a *AtomicLong) Get() int64       { a.mu.Lock(); defer a.mu.Unlock(); return a.v }
func (a *AtomicLong) GetPlain() int64  { return a.Get() }
func (a *AtomicLong) LongValue() int64 { return a.Get() }
func (a *AtomicLong) Set(v int64)      { a.mu.Lock(); a.v = v; a.mu.Unlock() }
func (a *AtomicLong) SetPlain(v int64) { a.Set(v) }
func (a *AtomicLong) AddAndGet(d int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.v += d
	return a.v
}
func (a *AtomicLong) GetAndAdd(d int64) int64 { return a.AddAndGet(d) - d }
func (a *AtomicLong) IncrementAndGet() int64  { return a.AddAndGet(1) }
func (a *AtomicLong) GetAndIncrement() int64  { return a.AddAndGet(1) - 1 }
func (a *AtomicLong) DecrementAndGet() int64  { return a.AddAndGet(-1) }
func (a *AtomicLong) GetAndSet(v int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.v
	a.v = v
	return old
}
func (a *AtomicLong) CompareAndSet(expected, v int64) bool {
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

// AtomicIntegerArray and AtomicReferenceArray are the fixed-size arrays of cells.
type AtomicIntegerArray struct {
	mu sync.Mutex
	v  []int32
}

func NewAtomicIntegerArray(n int32) *AtomicIntegerArray {
	return &AtomicIntegerArray{v: make([]int32, n)}
}

func (a *AtomicIntegerArray) Get(i int32) int32 { a.mu.Lock(); defer a.mu.Unlock(); return a.v[i] }
func (a *AtomicIntegerArray) Set(i, x int32)    { a.mu.Lock(); a.v[i] = x; a.mu.Unlock() }
func (a *AtomicIntegerArray) Length() int32     { return int32(len(a.v)) }
func (a *AtomicIntegerArray) ToString() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return arrayString(a.v)
}

type AtomicReferenceArray struct {
	mu sync.Mutex
	v  []any
}

func NewAtomicReferenceArray(n int32) *AtomicReferenceArray {
	return &AtomicReferenceArray{v: make([]any, n)}
}

func (a *AtomicReferenceArray) Get(i int32) any { a.mu.Lock(); defer a.mu.Unlock(); return a.v[i] }
func (a *AtomicReferenceArray) Set(i int32, x any) {
	a.mu.Lock()
	a.v[i] = x
	a.mu.Unlock()
}
func (a *AtomicReferenceArray) Length() int32 { return int32(len(a.v)) }
func (a *AtomicReferenceArray) ToString() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return arrayString(a.v)
}

// arrayString is the "[a, b, c]" of AtomicXArray.toString().
func arrayString[T any](v []T) string {
	parts := make([]string, len(v))
	for i, e := range v {
		parts[i] = fmt.Sprint(e)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
