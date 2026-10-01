//go:build linux

package gtk

import (
	"reflect"
	"sync"

	"github.com/ebitengine/purego"
)

// NewCallbackAny registers fn, whose Go parameter types mirror the C signature, as a C function
// pointer. purego hands out a fixed pool of callbacks that is never freed.
func NewCallbackAny(fn any) int64 { return int64(purego.NewCallback(fn)) }

// A slot is a purego trampoline whose Go function can be swapped: purego never frees callbacks
// (about 2000 per process), so Callback.dispose pools the slot for the next NewCallbackN.
type cbSlot struct {
	addr int64
	argc int
	fn   func(args []int64) int64
}

var (
	cbMu     sync.Mutex
	cbFree   = map[int][]*cbSlot{}
	cbByAddr = map[int64]*cbSlot{}
)

func newTrampoline(argCount int) *cbSlot {
	s := &cbSlot{argc: argCount}
	in := make([]reflect.Type, argCount)
	for i := range in {
		in[i] = reflect.TypeOf(uintptr(0))
	}
	ft := reflect.FuncOf(in, []reflect.Type{reflect.TypeOf(uintptr(0))}, false)
	f := reflect.MakeFunc(ft, func(a []reflect.Value) []reflect.Value {
		fn := s.fn
		var r int64
		if fn != nil {
			args := make([]int64, len(a))
			for i, v := range a {
				args[i] = int64(v.Uint())
			}
			r = fn(args)
		}
		return []reflect.Value{reflect.ValueOf(uintptr(r))}
	})
	s.addr = int64(purego.NewCallback(f.Interface()))
	return s
}

// NewCallbackN registers a C function taking argCount word-sized arguments and returning one word.
func NewCallbackN(argCount int, fn func(args []int64) int64) int64 {
	cbMu.Lock()
	defer cbMu.Unlock()
	var s *cbSlot
	if free := cbFree[argCount]; len(free) > 0 {
		s, cbFree[argCount] = free[len(free)-1], free[:len(free)-1]
	} else {
		s = newTrampoline(argCount)
		cbByAddr[s.addr] = s
	}
	s.fn = fn
	return s.addr
}

// FreeCallback returns the slot behind addr to the pool; calling the address afterwards is a no-op.
func FreeCallback(addr int64) {
	cbMu.Lock()
	defer cbMu.Unlock()
	if s := cbByAddr[addr]; s != nil && s.fn != nil {
		s.fn = nil
		cbFree[s.argc] = append(cbFree[s.argc], s)
	}
}

// constructorWrappers are made once per original constructor: Display restores the original on
// release, so each new Display asks for the same wrapper again.
var constructorWrappers = map[int64]int64{}

// constructorWrapper makes a GObjectClass.constructor that calls orig and records the object it made.
func constructorWrapper(orig int64, last *int64) int64 {
	cbMu.Lock()
	w, ok := constructorWrappers[orig]
	cbMu.Unlock()
	if ok {
		return w
	}
	w = NewCallbackAny(func(typ, n, props uintptr) uintptr {
		r := cCall(orig, typ, n, props)
		if last != nil {
			*last = r
		}
		return uintptr(r)
	})
	cbMu.Lock()
	constructorWrappers[orig] = w
	cbMu.Unlock()
	return w
}

var lastIMContext int64

// OSImContextLast is the GtkIMContext the most recent GtkIMMulticontext construction produced
// (GtkEntry creates its own, and SWT needs the handle).
func OSImContextLast() int64 { return lastIMContext }

func OSImContextNewProc_CALLBACK(orig int64) int64 { return constructorWrapper(orig, &lastIMContext) }
func OSPangoLayoutNewProc_CALLBACK(orig int64) int64 {
	return constructorWrapper(orig, nil)
}
func OSPangoFontFamilyNewProc_CALLBACK(orig int64) int64 { return constructorWrapper(orig, nil) }
func OSPangoFontFaceNewProc_CALLBACK(orig int64) int64   { return constructorWrapper(orig, nil) }
func OSPrinterOptionWidgetNewProc_CALLBACK(orig int64) int64 {
	return constructorWrapper(orig, nil)
}
