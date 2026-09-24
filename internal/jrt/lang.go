package jrt

import (
	"bytes"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
