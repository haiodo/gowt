package rx

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

var sinkInt int

func BenchmarkSetFlush1Effect(b *testing.B) {
	reset(b)
	a := New(0)
	stop := Effect(func() { sinkInt = a.Get() })
	defer stop()
	b.ReportAllocs()
	for i := range b.N {
		a.Set(i + 1)
		Flush()
	}
}

func BenchmarkSetEqualNoop(b *testing.B) {
	reset(b)
	a := New(7)
	stop := Effect(func() { sinkInt = a.Get() })
	defer stop()
	b.ReportAllocs()
	for range b.N {
		a.Set(7)
		Flush()
	}
}

func BenchmarkGetTracked(b *testing.B) {
	reset(b)
	a := New(1)
	d := Compute(func() int {
		s := 0
		for range 100 {
			s += a.Get()
		}
		return s
	})
	b.ReportAllocs()
	for i := range b.N {
		a.Set(i + 2)
		sinkInt = d.Get()
	}
}

// State -> Derived -> Derived -> Effect, the shape of "containers -> usage -> text -> label".
func BenchmarkSetFlushChain2(b *testing.B) {
	reset(b)
	a := New(0)
	d1 := Compute(func() int { return a.Get() * 2 })
	d2 := Compute(func() int { return d1.Get() + 1 })
	stop := Effect(func() { sinkInt = d2.Get() })
	defer stop()
	b.ReportAllocs()
	for i := range b.N {
		a.Set(i + 1)
		Flush()
	}
}

func BenchmarkFanout100Effects(b *testing.B) {
	reset(b)
	a := New(0)
	for range 100 {
		stop := Effect(func() { sinkInt += a.Get() })
		defer stop()
	}
	b.ReportAllocs()
	for i := range b.N {
		a.Set(i + 1)
		Flush()
	}
}

func BenchmarkEffectCreateDispose(b *testing.B) {
	reset(b)
	a := New(0)
	b.ReportAllocs()
	for range b.N {
		Effect(func() { sinkInt = a.Get() })()
	}
}

// One Post and its tick, the loop replaced by a direct call: the rx share of a hop.
func BenchmarkPostTick(b *testing.B) {
	reset(b)
	var pending func()
	detach := Attach(func(tick func()) func() { return func() { pending = tick } })
	defer detach()
	a := New(0)
	stop := Effect(func() { sinkInt = a.Get() })
	defer stop()
	b.ReportAllocs()
	for i := range b.N {
		a.Post(i + 1)
		pending()
	}
}

// Post contended by 8 goroutines while the UI side drains: ns per Post.
func BenchmarkPostContended(b *testing.B) {
	reset(b)
	ui := make(chan func(), 1)
	detach := Attach(func(tick func()) func() { return func() { ui <- tick } })
	defer detach()
	a := New(0)
	stop := Effect(func() { sinkInt = a.Get() })
	defer stop()
	done := make(chan struct{})
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	per := b.N/8 + 1
	for g := range 8 {
		wg.Go(func() {
			for i := range per {
				a.Post(g*per + i + 1)
			}
		})
	}
	go func() { wg.Wait(); close(done) }()
	for {
		select {
		case f := <-ui:
			f()
			continue
		case <-done:
		}
		break
	}
	for len(ui) > 0 {
		(<-ui)()
	}
}

type ctr struct {
	running bool
	cpu     float64
	mem     int64
}

type usage struct {
	cpu float64
	mem int64
}

func sum(cs []*ctr) usage {
	var u usage
	for _, c := range cs {
		if c.running {
			u.cpu += c.cpu
			u.mem += c.mem
		}
	}
	return u
}

// label mimics dockerdesk's lbl: the widget is touched only when the text changed.
type label struct {
	last  string
	calls int
}

func (l *label) set(s string) {
	if s != l.last {
		l.last = s
		l.calls++
	}
}

func snapshots(n int) [2][]*ctr {
	var s [2][]*ctr
	for k := range s {
		for i := range n {
			s[k] = append(s[k], &ctr{running: i%3 != 0, cpu: float64(i+k) * 0.5, mem: int64(i+k) << 20})
		}
	}
	return s
}

// The stats header: one poll result feeds three labels through a shared usage derived.
func BenchmarkStatsHeaderRx(b *testing.B) {
	reset(b)
	snaps := snapshots(50)
	ctrs := NewFunc(snaps[0], nil)
	use := Compute(func() usage { return sum(ctrs.Get()) })
	cpuText := Compute(func() string { return fmt.Sprintf("%.2f%% / %d00%%", use.Get().cpu, 8) })
	memText := Compute(func() string { return strconv.FormatInt(use.Get().mem>>20, 10) + " MiB" })
	foot := Compute(func() string { return fmt.Sprintf("RAM %d CPU %.2f%%", use.Get().mem>>20, use.Get().cpu/8) })
	var l1, l2, l3 label
	for _, x := range []struct {
		d *Derived[string]
		l *label
	}{{cpuText, &l1}, {memText, &l2}, {foot, &l3}} {
		stop := Effect(func() { x.l.set(x.d.Get()) })
		defer stop()
	}
	b.ReportAllocs()
	for i := range b.N {
		ctrs.Set(snaps[i&1])
		Flush()
	}
}

// The same work written by hand, as ui.apply in dockerdesk did it.
func BenchmarkStatsHeaderManual(b *testing.B) {
	snaps := snapshots(50)
	var l1, l2, l3 label
	b.ReportAllocs()
	for i := range b.N {
		u := sum(snaps[i&1])
		l1.set(fmt.Sprintf("%.2f%% / %d00%%", u.cpu, 8))
		l2.set(strconv.FormatInt(u.mem>>20, 10) + " MiB")
		l3.set(fmt.Sprintf("RAM %d CPU %.2f%%", u.mem>>20, u.cpu/8))
	}
}

func TestNodeSizes(t *testing.T) {
	t.Logf("node %d B, State[int] %d B, State[string] %d B, Derived[string] %d B",
		unsafe.Sizeof(node{}), unsafe.Sizeof(State[int]{}), unsafe.Sizeof(State[string]{}), unsafe.Sizeof(Derived[string]{}))
}
