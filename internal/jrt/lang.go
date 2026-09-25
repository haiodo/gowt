package jrt

import (
	"bytes"
	"math"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

// Runnable is java.lang.Runnable. Lambdas and anonymous classes become *RunnableFunc: a pointer,
// so two runnables compare by identity (Display.timerExec looks a runnable up with ==).
type Runnable interface {
	Run()
}

type RunnableFunc struct {
	Fn func()
}

func NewRunnable(fn func()) *RunnableFunc { return &RunnableFunc{Fn: fn} }

func (r *RunnableFunc) Run() { r.Fn() }

// Cast is a Java reference cast of an erased generic result (Map.get, Queue.poll): nil stays
// the zero value instead of panicking like a bare Go type assertion would.
func Cast[T any](x any) T {
	v, _ := x.(T)
	return v
}

// Every Java monitor collapses into one reentrant process-wide lock. Ceiling: unrelated blocks
// serialize; switch to a per-object monitor map if that ever matters.
var monitor struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner uint64
	depth int
}

func init() { monitor.cond = sync.NewCond(&monitor.mu) }

// goid parses the current goroutine id: Go has no public API for it, and a reentrant lock
// needs to recognise its own holder.
func goid() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	id, _ := strconv.ParseUint(string(b[:bytes.IndexByte(b, ' ')]), 10, 64)
	return id
}

func MonitorEnter() {
	g := goid()
	monitor.mu.Lock()
	for monitor.depth > 0 && monitor.owner != g {
		monitor.cond.Wait()
	}
	monitor.owner = g
	monitor.depth++
	monitor.mu.Unlock()
}

func MonitorExit() {
	monitor.mu.Lock()
	monitor.depth--
	if monitor.depth == 0 {
		monitor.cond.Broadcast()
	}
	monitor.mu.Unlock()
}

// MonitorWait is Object.wait(): releases the monitor fully, sleeps until notified, re-acquires.
func MonitorWait() {
	g := goid()
	monitor.mu.Lock()
	saved := monitor.depth
	monitor.depth = 0
	monitor.cond.Broadcast()
	monitor.cond.Wait()
	for monitor.depth > 0 && monitor.owner != g {
		monitor.cond.Wait()
	}
	monitor.owner = g
	monitor.depth = saved
	monitor.mu.Unlock()
}

func MonitorNotifyAll() {
	monitor.mu.Lock()
	monitor.cond.Broadcast()
	monitor.mu.Unlock()
}

// ClassForName is Class.forName, which SWT only calls to force a class's static initializer -
// Go package init has already run all of them.
func ClassForName(name string) reflect.Type { return nil }

// CurrentThread is Thread.currentThread() as a comparable identity: the goroutine id (the UI
// goroutine is locked to the main OS thread, so this is also the AppKit thread check).
func CurrentThread() any { return goid() }

// LocaleLanguage is locale.getLanguage() for the default (only) locale: the ISO code from $LANG.
func LocaleLanguage(locale any) string {
	lang := os.Getenv("LANG")
	if i := strings.IndexAny(lang, "_.@"); i >= 0 {
		lang = lang[:i]
	}
	if lang == "" || lang == "C" {
		return "en"
	}
	return lang
}

// JavaVersionFeature is Runtime.version().feature(): there is no JVM, so no Java version.
func JavaVersionFeature(version any) int32 { return 0 }

// IdentityHashCode is Object.hashCode: Go's heap objects don't move, so the address is stable.
func IdentityHashCode(x any) int32 { return int32(reflect.ValueOf(x).Pointer()) }

// StringHashCode is String.hashCode(): h = 31*h + c over the UTF-16 code units.
func StringHashCode(s string) int32 {
	var h int32
	for _, c := range utf16.Encode([]rune(s)) {
		h = 31*h + int32(c)
	}
	return h
}

// DoubleHashCode is Double.hashCode(d).
func DoubleHashCode(d float64) int32 {
	b := math.Float64bits(d)
	return int32(b ^ b>>32)
}

// ObjectsHash is Objects.hash(values...), i.e. Arrays.hashCode of the boxed values.
func ObjectsHash(values ...any) int32 {
	h := int32(1)
	for _, v := range values {
		var e int32
		switch x := v.(type) {
		case nil:
		case interface{ HashCode() int32 }:
			e = x.HashCode()
		case int32:
			e = x
		case string:
			e = StringHashCode(x)
		case float64:
			e = DoubleHashCode(x)
		case bool:
			e = 1237
			if x {
				e = 1231
			}
		default:
			e = IdentityHashCode(x)
		}
		h = 31*h + e
	}
	return h
}
