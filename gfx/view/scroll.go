package view

import (
	"math"
	"sync/atomic"
	"time"
)

const (
	scrollGapMin = 8 * time.Millisecond  // floor of the smoothing time, one 120 Hz frame
	scrollGapMax = 40 * time.Millisecond // above this a slow wheel would trail behind the fingers
	scrollMaxDt  = 100 * time.Millisecond
	scrollRest   = 0.05 // points: a remainder below this is applied at once
)

// scrollAcc collects pan deltas; the camera moves once per frame. Events come 0, 1 or 2 per
// display frame, and moving the camera by each one shows that as jerks.
type scrollAcc struct {
	x, y     float64       // view points not yet applied
	gap      time.Duration // smoothed time between events
	lastEv   time.Duration
	lastTick time.Duration
	events   int         // events since the last tick
	wake     atomic.Bool // set while something is left to apply; read by the display link thread
}

// add collects a delta of an event that arrived at now on the event clock.
func (a *scrollAcc) add(dx, dy float64, now time.Duration) {
	if g := now - a.lastEv; a.lastEv != 0 && g < scrollMaxDt {
		a.gap = (a.gap*4 + g) / 5
	}
	a.lastEv = now
	a.x, a.y = a.x+dx, a.y+dy
	a.events++
	a.wake.Store(true)
}

// step takes 1-exp(-dt/gap) of the collected delta: the camera trails the input by about one event
// interval and the rest goes to the next frames.
func (a *scrollAcc) step(now time.Duration) (dx, dy float64) {
	dt := min(max(now-a.lastTick, 0), scrollMaxDt)
	a.lastTick = now
	gap := a.gap
	if gap == 0 {
		gap = 16667 * time.Microsecond
	}
	gap = min(max(gap, scrollGapMin), scrollGapMax)
	f := 1 - math.Exp(-float64(dt)/float64(gap))
	dx, dy = a.x*f, a.y*f
	a.x, a.y = a.x-dx, a.y-dy
	if math.Abs(a.x) < scrollRest && math.Abs(a.y) < scrollRest {
		dx, dy = dx+a.x, dy+a.y
		a.x, a.y = 0, 0
		a.wake.Store(false)
	}
	return dx, dy
}

// tick moves the camera for the frame at now on the frame clock and returns how far it moved
// (view points) and how many scroll events it took in.
func (r *router) tick(now time.Duration) (dx, dy float64, events int) {
	events, r.acc.events = r.acc.events, 0
	dx, dy = r.acc.step(now)
	if dx != 0 || dy != 0 {
		r.pan(dx, dy)
	}
	return dx, dy, events
}
