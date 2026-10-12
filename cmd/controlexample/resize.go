package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/haiodo/gowt/swt"
)

// resizeCounter counts Resize events per widget type through a display filter.
type resizeCounter map[string]int

func (c resizeCounter) HandleEvent(e *swt.Event) { c[fmt.Sprintf("%T", e.Widget)]++ }

// resizeBench resizes the shell n times through Shell.SetSize, the call path a window drag reaches
// through windowDidResize and setFrameSize, and prints per-step timings. GOWT_PROFILE=<file> writes
// a CPU profile of the measured steps. The width sweeps 200 px out and back in 2 px steps.
func resizeBench(display *swt.Display, shell *swt.Shell, n int) {
	for i := 0; i < 50 && display.ReadAndDispatch(); i++ {
	}
	size := shell.GetSize()
	step := func(i int) {
		k := i % 200
		if k > 100 {
			k = 200 - k
		}
		shell.SetSize(size.X+int32(2*k), size.Y)
	}
	for i := 0; i < 20; i++ {
		step(i)
	}

	counter := resizeCounter{}
	display.AddFilter(swt.Resize, counter)

	if path := os.Getenv("GOWT_PROFILE"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "GOWT_PROFILE:", err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintln(os.Stderr, "GOWT_PROFILE:", err)
			os.Exit(1)
		}
		defer func() { pprof.StopCPUProfile(); f.Close() }()
	}

	times := make([]time.Duration, n)
	start := time.Now()
	for i := 0; i < n; i++ {
		t := time.Now()
		step(i)
		times[i] = time.Since(t)
	}
	total := time.Since(start)
	display.RemoveFilter(swt.Resize, counter)

	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	fmt.Printf("resize steps=%d total=%v mean=%v p50=%v p95=%v max=%v\n",
		n, total, total/time.Duration(n), times[n/2], times[n*95/100], times[n-1])
	type kv struct {
		k string
		v int
	}
	var rows []kv
	events := 0
	for k, v := range counter {
		rows = append(rows, kv{k, v})
		events += v
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].v > rows[j].v })
	fmt.Printf("Resize events per step: %.1f\n", float64(events)/float64(n))
	for i, r := range rows {
		if i == 8 {
			break
		}
		fmt.Printf("  %-28s %.1f/step\n", r.k, float64(r.v)/float64(n))
	}
}
