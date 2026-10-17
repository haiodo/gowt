package rx

import (
	"runtime/debug"
	"slices"
	"strings"
	"testing"
)

func fails(v int) {
	if v == 1 {
		panic("bad value")
	}
}

// A Catch takes the panic of an effect it owns; the round goes on and Flush does not panic.
func TestCatchTakesEffectPanic(t *testing.T) {
	reset(t)
	a := New(0)
	var got []any
	var stack string
	others := 0
	stop := Catch(func() {
		Effect(func() { fails(a.Get()) })
	}, func(r any) {
		got = append(got, r)
		stack = string(debug.Stack())
	})
	defer stop()
	stopOther := Effect(func() { a.Get(); others++ })
	defer stopOther()
	a.Set(1)
	Flush()
	if len(got) != 1 || got[0] != "bad value" || others != 2 {
		t.Fatalf("got %v, other effect ran %d times", got, others)
	}
	if !strings.Contains(stack, "rx.fails") {
		t.Errorf("handler stack does not show where the panic was raised:\n%s", stack)
	}
}

// The nearest Catch takes the panic; the outer one sees only what the inner handler raises.
func TestCatchNested(t *testing.T) {
	reset(t)
	a := New(0)
	var log []string
	var stopInner func()
	stop := Catch(func() {
		stopInner = Catch(func() {
			Effect(func() { fails(a.Get()) })
		}, func(r any) {
			log = append(log, "inner")
			stopInner()
			if a.Peek() == 1 {
				panic("again")
			}
		})
	}, func(r any) { log = append(log, "outer:"+r.(string)) })
	defer stop()
	a.Set(1)
	Flush()
	if !slices.Equal(log, []string{"inner", "outer:again"}) {
		t.Fatalf("log %v", log)
	}
	if len(a.observers) != 0 {
		t.Fatalf("disposed inner Catch left %d observers", len(a.observers))
	}
}

// A panic in handle with no Catch above goes to the caller of Flush.
func TestCatchHandlerPanics(t *testing.T) {
	reset(t)
	a := New(0)
	stop := Catch(func() {
		Effect(func() { fails(a.Get()) })
	}, func(r any) { panic("handler failed") })
	defer stop()
	a.Set(1)
	mustPanic(t, "handler failed", Flush)
}

// The first run of an Effect still panics in the code that makes it; effects of a disposed Catch stop.
func TestCatchFirstRunAndDispose(t *testing.T) {
	reset(t)
	a := New(1)
	handled := 0
	mustPanic(t, "bad value", func() {
		Catch(func() { Effect(func() { fails(a.Get()) }) }, func(any) { handled++ })
	})
	if handled != 0 || len(a.observers) != 0 {
		t.Fatalf("handled %d, observers %d", handled, len(a.observers))
	}
	runs, cleanups := 0, 0
	stop := Catch(func() {
		Effect(func() { a.Get(); runs++ })
		OnCleanup(func() { cleanups++ })
	}, func(any) { handled++ })
	stop()
	a.Set(2)
	Flush()
	if runs != 1 || cleanups != 1 || len(a.observers) != 0 {
		t.Fatalf("runs %d cleanups %d observers %d", runs, cleanups, len(a.observers))
	}
}
