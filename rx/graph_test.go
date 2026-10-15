package rx

import (
	"strings"
	"testing"
)

// Children subscribe before their parent, so a Set queues them first; the flush still runs the
// parent first, and it disposes the children the new list no longer has.
func TestParentRunsBeforeChildren(t *testing.T) {
	reset(t)
	list := NewFunc([]int{10, 20, 30}, nil)
	var bad any
	stop := Effect(func() {
		l := list.Get()
		for i := range l {
			Effect(func() {
				defer func() {
					if r := recover(); r != nil && bad == nil {
						bad = r
					}
				}()
				_ = list.Get()[i]
			})
		}
	})
	defer stop()
	list.Set([]int{1})
	Flush()
	if bad != nil {
		t.Fatalf("a child of the old list ran with the new list: %v", bad)
	}
}

// The effect writes the source of a Derived it read in the same run, before it is subscribed:
// it must run again and keep hearing the Derived.
func TestWriteAfterDerivedReadInFirstRun(t *testing.T) {
	reset(t)
	a := New(0)
	d := Compute(func() int { return a.Get() * 2 })
	seen := -1
	stop := Effect(func() {
		seen = d.Get()
		if a.Peek() == 0 {
			a.Set(1)
		}
	})
	defer stop()
	Flush()
	if seen != 2 {
		t.Errorf("after first run + Flush seen=%d want 2", seen)
	}
	a.Set(5)
	Flush()
	if seen != 10 {
		t.Fatalf("effect deaf to its derived: seen=%d want 10 (d.state=%d)", seen, d.state)
	}
}

// The same write made by a child effect created in the parent's run.
func TestChildWriteDuringParentFirstRun(t *testing.T) {
	reset(t)
	a := New(0)
	d := Compute(func() int { return a.Get() + 100 })
	seen := -1
	stop := Effect(func() {
		seen = d.Get()
		Effect(func() {
			if a.Peek() == 0 {
				a.Set(1)
			}
		})
	})
	defer stop()
	Flush()
	if seen != 101 {
		t.Errorf("after first run + Flush seen=%d want 101", seen)
	}
	a.Set(7)
	Flush()
	if seen != 107 {
		t.Fatalf("seen=%d want 107", seen)
	}
}

// A write under Untrack to a State the effect read reruns the effect (gowt.Bind applies under Untrack).
func TestUntrackedSelfWriteFirstRun(t *testing.T) {
	reset(t)
	a := New(0)
	seen := -1
	stop := Effect(func() {
		v := a.Get()
		Untrack(func() {
			if v == 0 {
				a.Set(1)
			}
		})
		seen = v
	})
	defer stop()
	Flush()
	if seen != 1 {
		t.Fatalf("seen=%d want 1, a=%d", seen, a.Peek())
	}
}

// A Derived read outside any effect must not stay subscribed to its long-lived sources.
func TestUnobservedDerivedDoesNotSubscribe(t *testing.T) {
	reset(t)
	a := New(0)
	for range 1000 {
		d := Compute(func() int { return a.Get() + 1 })
		_ = d.Get()
	}
	if n := len(a.observers); n != 0 {
		t.Fatalf("a long-lived State holds %d throwaway deriveds", n)
	}
}

// What a Derived creates is released when it recomputes.
func TestEffectInsideDerived(t *testing.T) {
	reset(t)
	a := New(0)
	runs := 0
	d := Compute(func() int {
		v := a.Get()
		Effect(func() { a.Get(); runs++ })
		return v
	})
	stop := Effect(func() { d.Get() })
	defer stop()
	for i := 1; i <= 3; i++ {
		a.Set(i)
		Flush()
	}
	runs = 0
	a.Set(10)
	Flush()
	if len(d.children) > 1 || runs > 1 {
		t.Fatalf("derived keeps %d children, %d inner effects ran for one Set", len(d.children), runs)
	}
}

// A panicking cleanup does not skip the cleanups after it.
func TestCleanupPanicRunsTheRest(t *testing.T) {
	reset(t)
	a := New(0)
	second := 0
	stop := Effect(func() {
		a.Get()
		OnCleanup(func() { panic("c1") })
		OnCleanup(func() { second++ })
	})
	a.Set(1)
	func() {
		defer func() { recover() }()
		Flush()
	}()
	stop()
	if second == 0 {
		t.Fatal("second cleanup never ran")
	}
}

// Batch inside an effect only schedules: running the effect again inside itself would let the
// outer, older run finish last and its value win.
func TestBatchInsideEffectDoesNotReenter(t *testing.T) {
	reset(t)
	a := New(0)
	var shown []int
	stop := Effect(func() {
		v := a.Get()
		if v == 0 {
			a.Set(1)
		}
		Batch(func() {})
		shown = append(shown, v)
	})
	defer stop()
	Flush()
	if shown[len(shown)-1] != a.Peek() {
		t.Fatalf("last applied %v, state %d", shown, a.Peek())
	}
}

// A child effect created in the parent's first run writes a State the parent read.
func TestChildWritesParentStateFirstRun(t *testing.T) {
	reset(t)
	a := New(0)
	seen := -1
	stop := Effect(func() {
		seen = a.Get()
		Effect(func() {
			if a.Peek() == 0 {
				a.Set(1)
			}
		})
	})
	defer stop()
	Flush()
	if seen != 1 {
		t.Fatalf("seen=%d want 1", seen)
	}
}

func TestComputeOffTheUIThreadPanics(t *testing.T) {
	reset(t)
	fakeLoop(t)
	got := make(chan any)
	go func() {
		defer func() { got <- recover() }()
		Compute(func() int { return 1 })
	}()
	if s, _ := (<-got).(string); !strings.Contains(s, "Compute called off the UI thread") {
		t.Fatalf("panic=%q", s)
	}
}

// An unobserved Derived is memoized until the next Set, and subscribes once an effect reads it.
func TestUnobservedDerivedThenObserved(t *testing.T) {
	reset(t)
	a := New(1)
	calls := 0
	d := Compute(func() int { calls++; return a.Get() * 10 })
	if d.Get() != 10 || d.Get() != 10 || calls != 1 || len(a.observers) != 0 {
		t.Fatalf("calls=%d observers=%d", calls, len(a.observers))
	}
	a.Set(2)
	seen := 0
	stop := Effect(func() { seen = d.Get() })
	defer stop()
	a.Set(3)
	Flush()
	if seen != 30 || len(a.observers) != 1 {
		t.Fatalf("seen=%d observers=%d", seen, len(a.observers))
	}
	stop()
	if len(a.observers) != 0 {
		t.Fatal("the derived kept its source after its last observer left")
	}
}

func TestDerivedPeekDoesNotSubscribe(t *testing.T) {
	reset(t)
	a := New(1)
	d := Compute(func() int { return a.Get() + 1 })
	runs := 0
	stop := Effect(func() { d.Peek(); runs++ })
	defer stop()
	a.Set(2)
	Flush()
	if runs != 1 || d.Peek() != 3 {
		t.Fatalf("runs=%d d=%d", runs, d.Peek())
	}
}
