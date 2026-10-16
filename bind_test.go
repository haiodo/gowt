package gowt

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

func TestBindings(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		n := rx.New(1)
		calls := 0
		text := rx.Compute(func() string { calls++; return "n=" + strconv.Itoa(n.Get()) })
		lbl := put[*LabelView](w, Label(""))
		lbl.BindText(text)
		if lbl.l.GetText() != "n=1" {
			t.Errorf("BindText first run: %q", lbl.l.GetText())
		}
		n.Set(2)
		if lbl.l.GetText() != "n=1" {
			t.Error("BindText applied inside Set; effects run at Flush")
		}
		rx.Flush()
		if lbl.l.GetText() != "n=2" {
			t.Errorf("BindText after Flush: %q", lbl.l.GetText())
		}

		on := rx.New(true)
		cb := put[*ButtonView](w, Button("check", nil, Check()))
		cb.BindChecked(on)
		if !cb.Checked() {
			t.Error("BindChecked first run")
		}
		cb.b.SetSelection(false)
		notify(cb.Unwrap(), swt.Selection)
		if on.Peek() {
			t.Error("a click did not reach the state")
		}
		on.Set(true)
		rx.Flush()
		if !cb.Checked() {
			t.Error("Set did not check the button")
		}

		cb.BindEnabled(on)
		lbl.BindVisible(on)
		on.Set(false)
		rx.Flush()
		if cb.Enabled() || lbl.Visible() {
			t.Errorf("enabled %v visible %v after Set(false)", cb.Enabled(), lbl.Visible())
		}

		// Disposing the widget disposes the binding: the derived loses its last observer and stops.
		lbl.l.Dispose()
		before := calls
		n.Set(3)
		rx.Flush()
		if calls != before {
			t.Errorf("binding outlived its label: derived ran %d more times", calls-before)
		}

		// A binding made in a Scope dies with the Scope while the widget lives on.
		other := put[*LabelView](w, Label("keep"))
		stop := rx.Scope(func() { other.BindText(text) })
		stop()
		n.Set(4)
		rx.Flush()
		if other.l.GetText() != "n=3" || other.l.IsDisposed() {
			t.Errorf("label after its scope ended: %q", other.l.GetText())
		}
	})
}

func TestTextBindKeepsCaret(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		s := rx.New("hello")
		var seen []string
		stop := rx.Effect(func() { seen = append(seen, s.Get()) })
		defer stop()
		tx := put[*TextView](w, Text())
		tx.Bind(s)
		if tx.Text() != "hello" {
			t.Errorf("Text.Bind first run: %q", tx.Text())
		}
		// Typing in the middle: the state follows, the flush must not rewrite the field.
		tx.t.SetSelection(2)
		tx.t.Insert("XY")
		caret := tx.t.GetCaretPosition()
		if s.Peek() != "heXYllo" {
			t.Errorf("edit did not reach the state: %q", s.Peek())
		}
		rx.Flush()
		if tx.t.GetCaretPosition() != caret || caret != 4 {
			t.Errorf("caret moved %d -> %d", caret, tx.t.GetCaretPosition())
		}
		s.Set("reset")
		rx.Flush()
		if tx.Text() != "reset" || s.Peek() != "reset" {
			t.Errorf("Set did not replace the content: %q", tx.Text())
		}
		if !slices.Equal(seen, []string{"hello", "heXYllo", "reset"}) {
			t.Errorf("state changes %q: an echo loop or a lost edit", seen)
		}
	})
}

// A Post from a goroutine reaches an effect through the real event loop.
func TestPostReachesEffect(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		s := rx.New(0)
		got := make(chan int, 1)
		rx.Effect(func() {
			if v := s.Get(); v != 0 {
				got <- v
			}
		})
		go s.Post(42)
		deadline := time.Now().Add(5 * time.Second)
		for len(got) == 0 && time.Now().Before(deadline) {
			if !app.display.ReadAndDispatch() {
				time.Sleep(time.Millisecond)
			}
		}
		if len(got) == 0 || <-got != 42 {
			t.Error("the Post never reached the effect")
		}
	})
}

func guiBench(b *testing.B, setup func(app *App)) {
	b.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		b.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() { err = Run(setup) })
	if err != nil {
		b.Fatal(err)
	}
}

// Post -> AsyncExec -> tick -> effect -> back to the posting goroutine: one full round trip.
func BenchmarkRxPostRoundTrip(b *testing.B) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		w.Show()
		s := rx.New(0)
		back := make(chan int, 1)
		rx.Effect(func() {
			if v := s.Get(); v != 0 {
				back <- v
			}
		})
		go func() {
			lat := make([]time.Duration, 0, b.N)
			b.ResetTimer()
			for i := range b.N {
				t0 := time.Now()
				s.Post(i + 1)
				<-back
				lat = append(lat, time.Since(t0))
			}
			b.StopTimer()
			slices.Sort(lat)
			b.ReportMetric(float64(lat[len(lat)/2].Nanoseconds()), "p50-ns")
			b.ReportMetric(float64(lat[len(lat)*99/100].Nanoseconds()), "p99-ns")
			app.Async(w.Close)
		}()
	})
}

// The same round trip with App.Async, the path dockerdesk used before rx.
func BenchmarkAsyncRoundTrip(b *testing.B) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		w.Show()
		back := make(chan int, 1)
		go func() {
			b.ResetTimer()
			for i := range b.N {
				app.Async(func() { back <- i })
				<-back
			}
			b.StopTimer()
			app.Async(w.Close)
		}()
	})
}

// 8 goroutines post as fast as they can: posts per hop and effect runs per second.
func BenchmarkRxPostThroughput(b *testing.B) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		w.Show()
		s := rx.New(0)
		runs := 0
		rx.Effect(func() { s.Get(); runs++ })
		go func() {
			var wg sync.WaitGroup
			per := b.N/8 + 1
			b.ResetTimer()
			t0 := time.Now()
			for g := range 8 {
				wg.Go(func() {
					for i := range per {
						s.Post(g*per + i + 1)
					}
				})
			}
			wg.Wait()
			done := make(chan int)
			app.Async(func() { rx.Flush(); done <- runs })
			r := <-done
			el := time.Since(t0)
			b.StopTimer()
			b.ReportMetric(float64(8*per)/float64(r), "posts/run")
			b.ReportMetric(float64(r)/el.Seconds(), "runs/s")
			app.Async(w.Close)
		}()
	})
}

type benchCtr struct {
	running bool
	cpu     float64
	mem     int64
}

func benchSnaps(n int) [2][]benchCtr {
	var s [2][]benchCtr
	for k := range s {
		for i := range n {
			s[k] = append(s[k], benchCtr{i%3 != 0, float64(i+k) * 0.5, int64(i+k) << 20})
		}
	}
	return s
}

func benchUsage(cs []benchCtr) (cpu float64, mem int64) {
	for _, c := range cs {
		if c.running {
			cpu += c.cpu
			mem += c.mem
		}
	}
	return
}

// dockerdesk's stats header and footer on real labels: one snapshot per op, three labels change.
func BenchmarkHeaderRx(b *testing.B) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		cpuL, memL, footL := put[*LabelView](w, Label("-")), put[*LabelView](w, Label("-")), put[*LabelView](w, Label("-"))
		w.Show()
		snaps := benchSnaps(50)
		snap := rx.NewFunc(snaps[0], nil)
		type usage struct {
			cpu float64
			mem int64
		}
		use := rx.Compute(func() usage { c, m := benchUsage(snap.Get()); return usage{c, m} })
		cpuL.BindText(rx.Compute(func() string { return fmt.Sprintf("%.2f%% / %d00%%", use.Get().cpu, 8) }))
		memL.BindText(rx.Compute(func() string { return strconv.FormatInt(use.Get().mem>>20, 10) + " MiB" }))
		footL.BindText(rx.Compute(func() string { return fmt.Sprintf("RAM %d CPU %.2f%%", use.Get().mem>>20, use.Get().cpu/8) }))
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			snap.Set(snaps[i&1])
			rx.Flush()
			app.display.RunDeferredLayouts() // what the next event-loop turn does with BindText's DEFER layout
		}
		b.StopTimer()
		w.Close()
	})
}

func BenchmarkHeaderManual(b *testing.B) {
	guiBench(b, func(app *App) {
		w := app.Window("bench")
		cpuL, memL, footL := put[*LabelView](w, Label("-")), put[*LabelView](w, Label("-")), put[*LabelView](w, Label("-"))
		w.Show()
		snaps := benchSnaps(50)
		var last [3]string
		set := func(i int, l *LabelView, s string) bool {
			if s == last[i] {
				return false
			}
			last[i] = s
			l.SetText(s)
			return true
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			cpu, mem := benchUsage(snaps[i&1])
			changed := set(0, cpuL, fmt.Sprintf("%.2f%% / %d00%%", cpu, 8))
			changed = set(1, memL, strconv.FormatInt(mem>>20, 10)+" MiB") || changed
			changed = set(2, footL, fmt.Sprintf("RAM %d CPU %.2f%%", mem>>20, cpu/8)) || changed
			if changed {
				w.Unwrap().LayoutOverload2(true, true)
			}
		}
		b.StopTimer()
		w.Close()
	})
}
