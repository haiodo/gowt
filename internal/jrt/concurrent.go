package jrt

import (
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
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
// Ceiling: no stages beyond thenRunAsync/thenAccept, no join.
type CompletableFuture struct {
	mu      sync.Mutex
	done    bool
	failed  bool
	value   any
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
		f.value = supplier()
		failed = false
	}()
	return f
}

// CompletableFutureAllOf is CompletableFuture.allOf: done once every argument is.
func CompletableFutureAllOf(fs ...*CompletableFuture) *CompletableFuture {
	all := &CompletableFuture{}
	var mu sync.Mutex
	left, failed := len(fs), false
	if left == 0 {
		all.finish(false)
	}
	for _, f := range fs {
		f.whenDone(func() {
			mu.Lock()
			left--
			failed = failed || f.failed
			done := left == 0
			mu.Unlock()
			if done {
				all.finish(failed)
			}
		})
	}
	return all
}

func NewCompletableFuture() *CompletableFuture { return &CompletableFuture{} }

func CompletableFutureCompletedFuture(v any) *CompletableFuture {
	f := &CompletableFuture{value: v}
	f.finish(false)
	return f
}

// Complete is CompletableFuture.complete: false when the future was already done.
func (f *CompletableFuture) Complete(v any) bool {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return false
	}
	f.value = v
	f.mu.Unlock()
	f.finish(false)
	return true
}

// ThenAccept runs fn with the value as soon as the future is done; the returned future is done after fn.
func (f *CompletableFuture) ThenAccept(fn func(any)) *CompletableFuture {
	g := &CompletableFuture{}
	f.whenDone(func() {
		if f.failed {
			g.finish(true)
			return
		}
		failed := true
		defer func() { recover(); g.finish(failed) }()
		fn(f.value)
		failed = false
	})
	return g
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

// SystemProperties backs System.getProperty/setProperty/getProperties; values are strings. os.name is
// the running OS as a JDK names it (the shared generated files are translated once, for every OS).
var SystemProperties = func() *Map {
	m := NewMap()
	m.Put("os.name", javaOSName(runtime.GOOS))
	m.Put("line.separator", LineSeparator())
	return m
}()

// LineSeparator is System.lineSeparator(): "\r\n" on Windows (StyledText copies text with it).
func LineSeparator() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

// javaOSName is the JDK's os.name for a GOOS. Windows is always "Windows 10": the tests check only the "Windows" prefix.
func javaOSName(goos string) string {
	switch goos {
	case "darwin":
		return "Mac OS X"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows 10"
	}
	return goos
}

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

// GetInteger is Integer.getInteger(key, def): the property parsed as a number, def when unset or not one.
func GetInteger(key string, def int32) int32 {
	if v, ok := SystemProperties.Get(key).(string); ok {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32); err == nil {
			return int32(n)
		}
	}
	return def
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
