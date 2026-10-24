//go:build darwin && gfxmock

package main

import (
	"fmt"
	"math"
	"time"

	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/gfx/view"
)

// scrollStats prints, once per second while the view is moving, how the frames and the scroll
// steps went. With GFXDEMO_TRACE=1 every frame goes to stderr too.
type scrollStats struct {
	since  time.Time
	frames []view.FrameReport
}

func (s *scrollStats) add(r view.FrameReport) {
	cg.Tracef("frame t=%.1fms int=%.2f cpu=%.2f ev=%d d=(%.2f,%.2f) origin=(%.1f,%.1f) z=%.3f tiles=%d",
		float64(r.Time)/1e6, ms(r.Interval), ms(r.CPU), r.Events, r.DX, r.DY, r.Origin.X, r.Origin.Y, r.Zoom, r.Tiles)
	if s.since.IsZero() {
		s.since = time.Now()
	}
	s.frames = append(s.frames, r)
	if time.Since(s.since) < time.Second {
		return
	}
	s.print()
	s.since, s.frames = time.Now(), s.frames[:0]
}

func ms(d time.Duration) float64 { return float64(d) / 1e6 }

func (s *scrollStats) print() {
	var iv, cpu, step []float64
	var ev [3]int
	over, tiles, events := 0, 0, 0
	for _, f := range s.frames {
		if f.Interval > 0 && f.Interval < 100*time.Millisecond {
			iv = append(iv, ms(f.Interval))
			if f.Interval > 12*time.Millisecond {
				over++
			}
		}
		cpu = append(cpu, ms(f.CPU))
		tiles += f.Tiles
		events += f.Events
		ev[min(f.Events, 2)]++
		if f.Events > 0 || f.DX != 0 || f.DY != 0 {
			step = append(step, math.Hypot(f.DX, f.DY))
		}
	}
	if events == 0 && len(step) == 0 {
		return // idle: tile arrivals only
	}
	var mean, sd float64
	for _, x := range step {
		mean += x
	}
	if len(step) > 0 {
		mean /= float64(len(step))
	}
	for _, x := range step {
		sd += (x - mean) * (x - mean)
	}
	if len(step) > 0 {
		sd = math.Sqrt(sd / float64(len(step)))
	}
	fmt.Printf("input 1s: frames %d interval ms p50 %.2f p99 %.2f max %.2f (%d over 12) cpu ms p50 %.2f max %.2f tiles %d | events %d per frame 0:%d 1:%d 2+:%d | step pt mean %.2f sd %.2f max %.2f\n",
		len(s.frames), pct(iv, .5), pct(iv, .99), pct(iv, 1), over, pct(cpu, .5), pct(cpu, 1), tiles,
		events, ev[0], ev[1], ev[2], mean, sd, pct(step, 1))
}
