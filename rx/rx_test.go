package rx

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func reset(t testing.TB) {
	t.Helper()
	if len(queue) != 0 || flushing || batchDepth != 0 || observer != nil || owner != nil || scheduled {
		t.Fatalf("leftover global state: queue=%d flushing=%v batch=%d scheduled=%v", len(queue), flushing, batchDepth, scheduled)
	}
	if wake != nil || uiThread.Load() != 0 {
		t.Fatal("a loop is still attached")
	}
}

// fakeLoop attaches rx to a channel the test drains by hand, standing in for Display.AsyncExec.
func fakeLoop(t testing.TB) chan func() {
	ui := make(chan func(), 1024)
	detach := Attach(func(tick func()) func() { return func() { ui <- tick } })
	t.Cleanup(func() {
		for len(ui) > 0 {
			(<-ui)()
		}
		detach()
	})
	return ui
}

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("no panic, want %q", want)
		}
		if s, _ := r.(string); !strings.Contains(s, want) {
			t.Fatalf("panic %v, want %q", r, want)
		}
	}()
	f()
}

func TestEffectRunsOncePerFlush(t *testing.T) {
	reset(t)
	a := New(1)
	runs, seen := 0, 0
	stop := Effect(func() { runs++; seen = a.Get() })
	defer stop()
	a.Set(2)
	a.Set(3)
	if runs != 1 {
		t.Fatalf("effect ran inside Set: runs=%d", runs)
	}
	Flush()
	if runs != 2 || seen != 3 {
		t.Fatalf("runs=%d seen=%d, want 2 and 3", runs, seen)
	}
	a.Set(3)
	Flush()
	if runs != 2 {
		t.Fatal("equal Set reran the effect")
	}
}

// Diamond a -> b, a -> c, (b, c) -> effect: one run, never a mixed old/new view.
func TestGlitchFree(t *testing.T) {
	reset(t)
	a := New(1)
	b := Compute(func() int { return a.Get() * 2 })
	c := Compute(func() int { return a.Get() + 10 })
	var log []string
	stop := Effect(func() {
		log = append(log, strconv.Itoa(b.Get())+"/"+strconv.Itoa(c.Get()))
	})
	defer stop()
	a.Set(5)
	Flush()
	a.Set(7) // the derived recomputed inside the last run must not leave the effect marked
	Flush()
	if !slices.Equal(log, []string{"2/11", "10/15", "14/17"}) {
		t.Fatalf("log=%v", log)
	}
}

// A deeper diamond: the two paths have different lengths and the sink still runs once.
func TestDiamondUnevenPaths(t *testing.T) {
	reset(t)
	a := New(1)
	b := Compute(func() int { return a.Get() + 1 })
	c := Compute(func() int { return b.Get() * 10 })
	sumCalls := 0
	d := Compute(func() int { sumCalls++; return a.Get() + c.Get() })
	var log []int
	stop := Effect(func() { log = append(log, d.Get()) })
	defer stop()
	a.Set(2)
	Flush()
	if !slices.Equal(log, []int{21, 32}) || sumCalls != 2 {
		t.Fatalf("log=%v sumCalls=%d", log, sumCalls)
	}
}

func TestDerivedIsLazyAndMemoized(t *testing.T) {
	reset(t)
	a := New(1)
	calls := 0
	d := Compute(func() int { calls++; return a.Get() % 2 })
	if calls != 0 {
		t.Fatal("derived computed before any read")
	}
	if d.Get() != 1 || d.Get() != 1 || calls != 1 {
		t.Fatalf("calls=%d after two reads", calls)
	}
	a.Set(3)
	if calls != 1 {
		t.Fatal("Set recomputed eagerly")
	}
	runs := 0
	stop := Effect(func() { d.Get(); runs++ })
	defer stop()
	a.Set(5) // parity unchanged: d recomputes, the effect must not rerun
	Flush()
	if runs != 1 || calls != 3 {
		t.Fatalf("runs=%d calls=%d, want 1 and 3", runs, calls)
	}
}

func TestReadAfterSetIsFresh(t *testing.T) {
	reset(t)
	a := New(1)
	d := Compute(func() int { return a.Get() + 1 })
	d.Get()
	a.Set(10)
	if d.Get() != 11 {
		t.Fatalf("pull after Set got %d", d.Get())
	}
}

func TestDynamicDependencies(t *testing.T) {
	reset(t)
	cond, x, y := New(true), New(1), New(2)
	runs := 0
	stop := Effect(func() {
		runs++
		if cond.Get() {
			x.Get()
		} else {
			y.Get()
		}
	})
	defer stop()
	cond.Set(false)
	Flush()
	if len(x.observers) != 0 || len(y.observers) != 1 {
		t.Fatalf("x obs=%d y obs=%d after the branch switched", len(x.observers), len(y.observers))
	}
	x.Set(5)
	Flush()
	if runs != 2 {
		t.Fatalf("dropped dependency still triggers: runs=%d", runs)
	}
}

func TestCleanupAndDispose(t *testing.T) {
	reset(t)
	a := New(0)
	var log []string
	stop := Effect(func() {
		v := strconv.Itoa(a.Get())
		log = append(log, "run"+v)
		OnCleanup(func() { log = append(log, "clean"+v) })
	})
	a.Set(1)
	Flush()
	stop()
	a.Set(2)
	Flush()
	want := []string{"run0", "clean0", "run1", "clean1"}
	if !slices.Equal(log, want) {
		t.Fatalf("log=%v want %v", log, want)
	}
	if len(a.observers) != 0 {
		t.Fatal("disposed effect still subscribed: the state would keep its widget alive")
	}
}

func TestEffectDisposesItself(t *testing.T) {
	reset(t)
	a := New(0)
	cleaned := false
	var stop func()
	stop = Effect(func() {
		if a.Get() == 1 {
			stop()
		}
		OnCleanup(func() { cleaned = true })
	})
	a.Set(1)
	Flush()
	if !cleaned || len(a.observers) != 0 {
		t.Fatalf("cleaned=%v observers=%d", cleaned, len(a.observers))
	}
}

func TestNestedEffectsDieWithParent(t *testing.T) {
	reset(t)
	outer, inner := New(0), New(0)
	innerRuns := 0
	stop := Effect(func() {
		outer.Get()
		Effect(func() { inner.Get(); innerRuns++ })
	})
	outer.Set(1) // the old inner effect is disposed, a new one is made
	Flush()
	if len(inner.observers) != 1 {
		t.Fatalf("inner observers=%d, want 1", len(inner.observers))
	}
	stop()
	if len(inner.observers) != 0 {
		t.Fatal("inner effect outlived its parent")
	}
	if innerRuns != 2 {
		t.Fatalf("innerRuns=%d", innerRuns)
	}
}

func TestScopeDispose(t *testing.T) {
	reset(t)
	a := New(0)
	d := Compute(func() int { return a.Get() * 2 })
	runs, disposed := 0, 0
	stop := Scope(func() {
		OnCleanup(func() { disposed++ })
		for range 3 {
			Effect(func() { d.Get(); runs++ })
		}
	})
	a.Set(1)
	Flush()
	if runs != 6 {
		t.Fatalf("runs=%d, want 6", runs)
	}
	stop()
	stop()
	a.Set(2)
	Flush()
	if runs != 6 || disposed != 1 {
		t.Fatalf("after dispose runs=%d disposed=%d", runs, disposed)
	}
	// The derived lost its last observer, so it let go of a: the state keeps nothing alive.
	if len(a.observers) != 0 || len(d.observers) != 0 {
		t.Fatalf("a observers=%d d observers=%d", len(a.observers), len(d.observers))
	}
	if d.Get() != 4 {
		t.Fatal("an unlinked derived must recompute on read")
	}
}

func TestUntrack(t *testing.T) {
	reset(t)
	a, b := New(0), New(0)
	runs := 0
	stop := Effect(func() {
		a.Get()
		Untrack(func() { b.Get() })
		b.Peek()
		runs++
	})
	defer stop()
	b.Set(1)
	Flush()
	if runs != 1 {
		t.Fatal("untracked read subscribed")
	}
	a.Set(1)
	Flush()
	if runs != 2 {
		t.Fatal("tracked read lost")
	}
}

func TestBatchFlushesAtEnd(t *testing.T) {
	reset(t)
	a := New(0)
	seen, runs := -1, 0
	stop := Effect(func() { seen = a.Get(); runs++ })
	defer stop()
	Batch(func() {
		a.Set(1)
		Batch(func() { a.Set(2) })
		if seen != 0 {
			t.Fatal("effect ran inside Batch")
		}
	})
	if seen != 2 || runs != 2 {
		t.Fatalf("seen=%d runs=%d after Batch", seen, runs)
	}
}

func TestOneTickPerBatch(t *testing.T) {
	reset(t)
	ui := fakeLoop(t)
	a, b := New(0), New(0)
	runs := 0
	stop := Effect(func() { a.Get(); b.Get(); runs++ })
	defer stop()
	a.Set(1)
	b.Set(1)
	a.Set(2)
	if len(ui) != 1 {
		t.Fatalf("ticks queued: %d", len(ui))
	}
	(<-ui)()
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}
	a.Set(3)
	if len(ui) != 1 {
		t.Fatal("no tick after the flush")
	}
	(<-ui)()
}

func TestSetFlushDoesNotAllocate(t *testing.T) {
	reset(t)
	a := New(0)
	d1 := Compute(func() int { return a.Get() * 2 })
	d2 := Compute(func() int { return d1.Get() + 1 })
	sink := 0
	stop := Effect(func() { sink = d2.Get() })
	defer stop()
	i := 0
	if n := testing.AllocsPerRun(1000, func() { i++; a.Set(i); Flush() }); n != 0 {
		t.Fatalf("Set -> Derived -> Derived -> Effect allocates %v per run", n)
	}
	_ = sink
}

// An effect that panics must not cost the others their run, and the graph must keep working.
func TestPanicInEffectKeepsTheRound(t *testing.T) {
	reset(t)
	a := New(0)
	var log []string
	for _, name := range []string{"x", "boom", "y"} {
		stop := Effect(func() {
			v := a.Get()
			if name == "boom" && v == 1 {
				panic("boom")
			}
			log = append(log, name+strconv.Itoa(v))
		})
		defer stop()
	}
	a.Set(1)
	mustPanic(t, "boom", Flush)
	if !slices.Contains(log, "x1") || !slices.Contains(log, "y1") {
		t.Fatalf("a panic lost the rest of the round: %v", log)
	}
	a.Set(2)
	Flush()
	if !slices.Contains(log, "boom2") || !slices.Contains(log, "x2") {
		t.Fatalf("graph stuck after a panic: %v", log)
	}
}

// A Derived that panics leaves the chain clean: when its input is fixed the effect runs again.
func TestPanicInDerivedRecovers(t *testing.T) {
	reset(t)
	a := New(1)
	inv := Compute(func() int {
		if a.Get() == 0 {
			panic("division by zero")
		}
		return 100 / a.Get()
	})
	twice := Compute(func() int { return inv.Get() * 2 })
	other := Compute(func() int { return a.Get() + 1 })
	var log []int
	stop := Effect(func() { log = append(log, twice.Get()) })
	defer stop()
	seen := 0
	stop2 := Effect(func() { seen = other.Get() })
	defer stop2()
	a.Set(0)
	mustPanic(t, "division", Flush)
	if seen != 1 {
		t.Fatalf("the other effect missed the change: %d", seen)
	}
	a.Set(4)
	Flush()
	if !slices.Equal(log, []int{200, 50}) || seen != 5 {
		t.Fatalf("log=%v seen=%d", log, seen)
	}
}

func TestPanicInFirstRunDisposes(t *testing.T) {
	reset(t)
	a := New(0)
	mustPanic(t, "first", func() {
		Effect(func() { a.Get(); panic("first") })
	})
	if len(a.observers) != 0 {
		t.Fatal("an effect that failed to start is still subscribed")
	}
}

func TestPanicInBatchLeavesEffectsQueued(t *testing.T) {
	reset(t)
	a := New(0)
	seen := 0
	stop := Effect(func() { seen = a.Get() })
	defer stop()
	mustPanic(t, "fn", func() { Batch(func() { a.Set(1); panic("fn") }) })
	if seen != 0 {
		t.Fatal("effects ran while Batch was unwinding")
	}
	Flush()
	if seen != 1 {
		t.Fatalf("seen=%d", seen)
	}
}

func TestCycleIsReported(t *testing.T) {
	reset(t)
	a := New(0)
	stop := Effect(func() { a.Set(a.Get() + 1) })
	defer stop()
	mustPanic(t, "did not settle after 100 flush rounds", Flush)
	if len(queue) != 0 || a.Peek() < 100 {
		t.Fatalf("queue=%d a=%d", len(queue), a.Peek())
	}
}

func TestDerivedReadingItselfPanics(t *testing.T) {
	reset(t)
	var d *Derived[int]
	d = Compute(func() int { return d.Get() + 1 })
	mustPanic(t, "reads itself", func() { d.Get() })
}

func TestSetInsideDerivedPanics(t *testing.T) {
	reset(t)
	a, b := New(0), New(0)
	d := Compute(func() int { b.Set(1); return a.Get() })
	mustPanic(t, "inside a Derived", func() { d.Get() })
}

func TestSetOffTheUIThreadPanics(t *testing.T) {
	reset(t)
	fakeLoop(t)
	a := New(0)
	got := make(chan any)
	go func() {
		defer func() { got <- recover() }()
		a.Set(1)
	}()
	r := <-got
	if s, _ := r.(string); !strings.Contains(s, "State.Set called off the UI thread") {
		t.Fatalf("panic=%v", r)
	}
	if a.Peek() != 0 {
		t.Fatal("the off-thread Set was applied")
	}
}

func TestGetOffTheUIThreadPanicsUnderRxdebug(t *testing.T) {
	if !checkReads {
		t.Skip("build with -tags rxdebug")
	}
	reset(t)
	fakeLoop(t)
	a := New(0)
	got := make(chan any)
	go func() {
		defer func() { got <- recover() }()
		a.Get()
	}()
	if s, _ := (<-got).(string); !strings.Contains(s, "State.Get called off the UI thread") {
		t.Fatalf("panic=%q", s)
	}
}

func TestPostFromGoroutines(t *testing.T) {
	reset(t)
	ui := fakeLoop(t)
	a, b := New(0), New(0)
	runs := 0
	stop := Effect(func() { a.Get(); b.Get(); runs++ })
	defer stop()
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 1000 {
				if g%2 == 0 {
					a.Post(1)
				} else {
					b.Post(g*1000 + i + 1)
				}
			}
		})
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	hops := 0
	for {
		select {
		case f := <-ui:
			hops++
			f()
			continue
		case <-done:
		}
		break
	}
	for len(ui) > 0 {
		hops++
		(<-ui)()
	}
	if a.Peek() != 1 || b.Peek()%1000 != 0 || hops == 0 || hops > 8000 || runs > hops+1 {
		t.Fatalf("a=%d b=%d hops=%d runs=%d", a.Peek(), b.Peek(), hops, runs)
	}
	t.Logf("8000 posts reached the UI thread in %d hops, %d effect runs", hops, runs)
}

func TestPostBeforeAttachWaits(t *testing.T) {
	reset(t)
	a := New(0)
	a.Post(7)
	if a.Peek() != 0 {
		t.Fatal("Post applied without a loop")
	}
	ui := fakeLoop(t)
	if len(ui) != 1 {
		t.Fatalf("Attach queued %d ticks for a waiting Post", len(ui))
	}
	(<-ui)()
	if a.Peek() != 7 {
		t.Fatalf("a=%d", a.Peek())
	}
}
