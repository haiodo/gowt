package view

import (
	"math"
	"slices"
	"sync/atomic"
	"time"
)

const (
	scrollIdle     = 100 * time.Millisecond // a longer silence starts a new gesture
	scrollExtrap   = 4 * time.Millisecond   // longest extrapolation past the newest event
	scrollVelSpan  = 25 * time.Millisecond  // events this recent give the velocity
	scrollLeadInit = 16 * time.Millisecond
	scrollLeadMax  = 40 * time.Millisecond
	scrollHistory  = 64
)

type scrollSample struct {
	t    time.Duration
	x, y float64 // total delta of all events up to t, view points
}

// scrollAcc resamples scroll input: events come at 120-190 Hz, unlocked from the display, so a
// camera move per event or per frame sum jerks. The camera follows the total delta at frame time
// minus a lead.
type scrollAcc struct {
	hist   []scrollSample
	cx, cy float64 // total of all deltas
	ax, ay float64 // total applied to the camera
	lead   time.Duration
	events int         // events since the last tick
	wake   atomic.Bool // set while the camera has not reached the total; read by the display link thread
}

// add records a delta of an event stamped t on the frame clock.
func (a *scrollAcc) add(dx, dy float64, t time.Duration) {
	if n := len(a.hist); n == 0 || t-a.hist[n-1].t > scrollIdle {
		a.hist = append(a.hist[:0], scrollSample{t - 8*time.Millisecond, a.cx, a.cy})
	} else if t < a.hist[n-1].t {
		t = a.hist[n-1].t
	}
	a.cx, a.cy = a.cx+dx, a.cy+dy
	a.hist = append(a.hist, scrollSample{t, a.cx, a.cy})
	if len(a.hist) > scrollHistory {
		a.hist = a.hist[len(a.hist)-scrollHistory:]
	}
	a.events++
	a.wake.Store(true)
}

// position is the total delta the camera shows at time t.
func (a *scrollAcc) position(t time.Duration) (x, y float64) {
	h := a.hist
	last := h[len(h)-1]
	if t <= h[0].t {
		return h[0].x, h[0].y
	}
	if t < last.t {
		i := 1
		for h[i].t < t {
			i++
		}
		f := float64(t-h[i-1].t) / float64(h[i].t-h[i-1].t)
		return h[i-1].x + (h[i].x-h[i-1].x)*f, h[i-1].y + (h[i].y-h[i-1].y)*f
	}
	// the newest event is older than t: carry on at the recent velocity for a few ms, then settle
	// back to the total; h[0] is the start mark of the gesture, not an event
	k := len(h) - 2
	for k > 1 && last.t-h[k-1].t <= scrollVelSpan {
		k--
	}
	if k < 1 || last.t-h[k].t < time.Millisecond {
		return last.x, last.y
	}
	dt := float64(last.t - h[k].t)
	vx, vy := (last.x-h[k].x)/dt, (last.y-h[k].y)/dt
	over := t - last.t
	e := float64(min(over, scrollExtrap))
	if over > scrollExtrap {
		e *= math.Exp(-float64(over-scrollExtrap) / float64(20*time.Millisecond))
	}
	return last.x + vx*e, last.y + vy*e
}

// step returns the camera move for the frame shown at ft. The lead is the largest recent age of the
// newest event at a frame, so ft minus lead lies between events; it falls back slowly.
func (a *scrollAcc) step(ft time.Duration) (dx, dy float64) {
	if len(a.hist) == 0 {
		return 0, 0
	}
	last := a.hist[len(a.hist)-1]
	if age := ft - last.t; age < scrollIdle/2 {
		a.lead = max(age, a.lead-a.lead/100)
		if a.lead == 0 {
			a.lead = scrollLeadInit
		}
		a.lead = min(a.lead, scrollLeadMax)
	}
	if a.lead == 0 {
		a.lead = scrollLeadInit
	}
	if ft-last.t > scrollIdle+scrollExtrap {
		dx, dy = a.cx-a.ax, a.cy-a.ay
		a.ax, a.ay = a.cx, a.cy
		a.hist = a.hist[len(a.hist)-1:]
		a.wake.Store(false)
		return dx, dy
	}
	x, y := a.position(ft - a.lead)
	dx, dy = x-a.ax, y-a.ay
	a.ax, a.ay = x, y
	if i := slices.IndexFunc(a.hist, func(s scrollSample) bool { return s.t > ft-a.lead }); i > 1 {
		a.hist = a.hist[i-1:]
	}
	return dx, dy
}

// tick moves the camera for the frame shown at ft and returns how far it moved (view points) and
// how many scroll events came in since the last frame.
func (r *router) tick(ft time.Duration) (dx, dy float64, events int) {
	events, r.acc.events = r.acc.events, 0
	dx, dy = r.acc.step(ft)
	if dx != 0 || dy != 0 {
		r.pan(dx, dy)
	}
	return dx, dy, events
}
