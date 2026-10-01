package jrt

import (
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"
	"weak"
)

// CountDownLatch is java.util.concurrent.CountDownLatch.
type CountDownLatch struct {
	mu    sync.Mutex
	count int32
	done  chan struct{}
}

func NewCountDownLatch(count int32) *CountDownLatch {
	l := &CountDownLatch{count: count, done: make(chan struct{})}
	if count <= 0 {
		close(l.done)
	}
	return l
}

func (l *CountDownLatch) CountDown() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count > 0 {
		if l.count--; l.count == 0 {
			close(l.done)
		}
	}
}

func (l *CountDownLatch) Await(timeout int64, unit TimeUnit) bool {
	select {
	case <-l.done:
		return true
	case <-time.After(unit.duration(timeout)):
		return false
	}
}

// TimeUnit is java.util.concurrent.TimeUnit; only the units below are modeled.
type TimeUnit int32

const (
	TimeUnitMILLISECONDS TimeUnit = iota
	TimeUnitSECONDS
	TimeUnitNANOSECONDS
)

func (u TimeUnit) duration(n int64) time.Duration {
	switch u {
	case TimeUnitSECONDS:
		return time.Duration(n) * time.Second
	case TimeUnitNANOSECONDS:
		return time.Duration(n)
	}
	return time.Duration(n) * time.Millisecond
}

// Executor is java.util.concurrent.Executor (translated Display has Execute).
type Executor interface{ Execute(Runnable) }

// CompletableFuture covers supplyAsync/thenRunAsync and the state queries the tests read.
// Ceiling: no other stages, no join/get.
type CompletableFuture struct {
	mu      sync.Mutex
	done    bool
	failed  bool
	waiters []func()
}

func (f *CompletableFuture) finish(failed bool) {
	f.mu.Lock()
	f.done, f.failed = true, failed
	w := f.waiters
	f.waiters = nil
	f.mu.Unlock()
	for _, fn := range w {
		fn()
	}
}

func (f *CompletableFuture) whenDone(fn func()) {
	f.mu.Lock()
	if !f.done {
		f.waiters = append(f.waiters, fn)
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()
	fn()
}

func CompletableFutureSupplyAsync[T any](supplier func() T) *CompletableFuture {
	f := &CompletableFuture{}
	go func() {
		failed := true
		defer func() { recover(); f.finish(failed) }()
		supplier()
		failed = false
	}()
	return f
}

func (f *CompletableFuture) ThenRunAsync(r Runnable, executor any) *CompletableFuture {
	g := &CompletableFuture{}
	f.whenDone(func() {
		if f.failed {
			g.finish(true)
			return
		}
		executor.(Executor).Execute(NewRunnable(func() {
			failed := true
			defer func() { recover(); g.finish(failed) }()
			r.Run()
			failed = false
		}))
	})
	return g
}

func (f *CompletableFuture) IsDone() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.done
}

func (f *CompletableFuture) IsCancelled() bool { return false }

func (f *CompletableFuture) IsCompletedExceptionally() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.done && f.failed
}

// WeakReference is java.lang.ref.WeakReference over a pointer boxed in any.
type WeakReference struct {
	typ reflect.Type
	wp  weak.Pointer[byte]
}

func NewWeakReference(v any) *WeakReference {
	rv := reflect.ValueOf(v)
	if v == nil || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &WeakReference{}
	}
	return &WeakReference{typ: rv.Type(), wp: weak.Make((*byte)(unsafe.Pointer(rv.Pointer())))}
}

// Get is nil once the referent is collected; the caller casts the result.
func (w *WeakReference) Get() any {
	if w.typ == nil {
		return nil
	}
	p := w.wp.Value()
	if p == nil {
		return nil
	}
	return reflect.NewAt(w.typ.Elem(), unsafe.Pointer(p)).Interface()
}

// GC is System.gc().
func GC() {
	runtime.GC()
	runtime.GC()
}

// SystemProperties backs System.getProperty/setProperty/getProperties; values are strings.
var SystemProperties = NewMap()

// GetProperty is System.getProperty(key, def); an unset key with no default is the port's null String, "".
func GetProperty(key, def string) string {
	if v, ok := SystemProperties.Get(key).(string); ok {
		return v
	}
	return def
}

// SetProperty is System.setProperty: the previous value, "" when unset.
func SetProperty(key, value string) string {
	prev, _ := SystemProperties.Put(key, value).(string)
	return prev
}

// Getenv is System.getenv(name): "" for an unset variable.
func Getenv(name string) string { return os.Getenv(name) }

// Matches is String.matches: the whole string must match.
func Matches(s, regex string) bool {
	re, err := regexp.Compile("^(?:" + regex + ")$")
	return err == nil && re.MatchString(s)
}

// Environ is System.getenv(): a snapshot of the process environment.
func Environ() *Map {
	m := NewMap()
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m.Put(k, v)
		}
	}
	return m
}
