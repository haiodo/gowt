package view

import (
	"math"
	"math/rand"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
)

const frame120 = 8333333 * time.Nanosecond

// feed runs events against a 120 Hz display; the frame shown at ft runs lead earlier and sees only
// the events that arrived by then. It returns the camera step of every frame.
func feed(rg *rig, evs []time.Duration, delta func(i int) float64, frames int, lead time.Duration, rng *rand.Rand) []float64 {
	steps := make([]float64, 0, frames)
	next := 0
	for f := 1; f <= frames; f++ {
		ft := time.Duration(f) * frame120
		run := ft - lead + time.Duration(rng.Intn(2000))*time.Microsecond
		for next < len(evs) && evs[next]+500*time.Microsecond <= run {
			rg.r.scroll(pt(0, 0), 0, delta(next), true, PhaseChanged, PhaseNone, g.Mods{}, evs[next])
			next++
		}
		_, dy, _ := rg.r.tick(ft)
		steps = append(steps, dy)
	}
	return steps
}

// A finger at constant velocity: events at 120-190 Hz with jitter against a 120 Hz frame clock.
// The step per frame is even, and the total is the sum of the deltas.
func TestScrollConstantVelocityIsEven(t *testing.T) {
	for _, hz := range []float64{120, 150, 190} {
		rg := newRig(t)
		rg.cam.c.Zoom = 1
		rng := rand.New(rand.NewSource(int64(hz)))
		const vel = 0.8 // points per ms
		period := time.Duration(float64(time.Second) / hz)
		var evs []time.Duration
		var truth []time.Duration
		for i := 1; i <= int(hz); i++ { // one second
			tt := 100*time.Millisecond + time.Duration(i)*period
			truth = append(truth, tt)
			evs = append(evs, tt+time.Duration(rng.Intn(1000)-500)*time.Microsecond)
		}
		delta := func(i int) float64 {
			prev := 100 * time.Millisecond
			if i > 0 {
				prev = truth[i-1]
			}
			return vel * float64(truth[i]-prev) / 1e6
		}
		var want float64
		for i := range evs {
			want += delta(i)
		}
		steps := feed(rg, evs, delta, 160, 12*time.Millisecond, rng)
		var total float64
		for _, s := range steps {
			total += s
		}
		if math.Abs(total-want) > 1e-6 || math.Abs(rg.cam.c.Origin.Y-(50-want)) > 1e-6 {
			t.Fatalf("%v Hz: applied %v, camera moved %v, want %v", hz, total, 50-rg.cam.c.Origin.Y, want)
		}
		if rg.r.acc.wake.Load() {
			t.Fatalf("%v Hz: the display link stays woken after the camera arrived", hz)
		}
		// the middle of the gesture: past the start and the end of the ramp
		mid := steps[20:110]
		mean, sd := meanSd(mid)
		t.Logf("%v Hz: step mean %.3f sd %.3f (%.3f)", hz, mean, sd, sd/mean)
		if sd/mean > 0.1 {
			t.Fatalf("%v Hz: step sd/mean %.3f, want well under 0.2", hz, sd/mean)
		}
	}
}

// Momentum events carry their own timestamps and decay: the camera follows them the same way and
// the steps never jump back.
func TestScrollMomentumKeepsDirection(t *testing.T) {
	rg := newRig(t)
	rg.cam.c.Zoom = 1
	rng := rand.New(rand.NewSource(7))
	var evs []time.Duration
	for i := 1; i <= 120; i++ {
		evs = append(evs, 100*time.Millisecond+time.Duration(i)*7*time.Millisecond+time.Duration(rng.Intn(600)-300)*time.Microsecond)
	}
	delta := func(i int) float64 { return 12 * math.Pow(0.97, float64(i)) }
	steps := feed(rg, evs, delta, 160, 12*time.Millisecond, rng)
	// the first steps ramp up from rest; the decay runs from there on
	for i, s := range steps[3:] {
		if s < -0.1 { // the settle after the last event may step back a hair
			t.Fatalf("frame %d moved against the scroll by %v", i+3, s)
		}
	}
	var want float64
	for i := range evs {
		want += delta(i)
	}
	if math.Abs(rg.cam.c.Origin.Y-(50-want)) > 1e-6 {
		t.Fatalf("camera moved %v, want %v", 50-rg.cam.c.Origin.Y, want)
	}
}

func meanSd(v []float64) (m, sd float64) {
	for _, x := range v {
		m += x
	}
	m /= float64(len(v))
	for _, x := range v {
		sd += (x - m) * (x - m)
	}
	return m, math.Sqrt(sd / float64(len(v)))
}
