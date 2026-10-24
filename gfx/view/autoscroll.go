package view

import "time"

// AutoScrollOptions tune View.AutoScroll. Zero fields take the defaults.
type AutoScrollOptions struct {
	// Band is how far inside the view edge the scrolling starts, view points. Default 32.
	Band float64
	// Ramp is the distance past the band start at which the speed reaches Max. Default 120.
	Ramp float64
	// Max is the speed cap, view points per second. Default 1000.
	Max float64
	// Curve maps the distance fraction 0..1 to the speed fraction 0..1. Default linear.
	Curve func(t float64) float64
}

func (o *AutoScrollOptions) fill() {
	if o.Band <= 0 {
		o.Band = 32
	}
	if o.Ramp <= 0 {
		o.Ramp = 120
	}
	if o.Max <= 0 {
		o.Max = 1000
	}
	if o.Curve == nil {
		o.Curve = func(t float64) float64 { return t }
	}
}

// maxAutoStep bounds the frame interval one step uses, so a stalled frame does not jump the scene.
const maxAutoStep = 50 * time.Millisecond

// speed is the content speed along one axis for a pointer at c in a view of size n: positive moves
// the content toward larger coordinates (the camera toward the start edge).
func (o *AutoScrollOptions) speed(c, n float64) float64 {
	d := o.Band - c // distance into the start band, or past the start edge
	sign := 1.0
	if e := c - (n - o.Band); e > d {
		d, sign = e, -1
	}
	if d <= 0 {
		return 0
	}
	t := min(d/o.Ramp, 1)
	return sign * o.Max * min(max(o.Curve(t), 0), 1)
}

type autoScroll struct {
	opts AutoScrollOptions
	prev time.Duration
}

// startAuto begins auto-scroll for the press in progress; the returned function stops it. It ends
// by itself when the last button is released or the drag is cancelled.
func (r *router) startAuto(o AutoScrollOptions) (stop func()) {
	if r.held == 0 || r.size == nil {
		return func() {}
	}
	o.fill()
	a := &autoScroll{opts: o}
	r.auto = a
	r.autoWake.Store(true)
	return func() {
		if r.auto == a {
			r.stopAuto()
		}
	}
}

func (r *router) stopAuto() {
	r.auto = nil
	r.autoWake.Store(false)
}

// stepAuto pans for the frame at ft and repeats the last pointer position as a move, so the app
// puts the dragged content at the scene point now under the pointer.
func (r *router) stepAuto(ft time.Duration) {
	a := r.auto
	if a == nil || r.held == 0 {
		return
	}
	prev := a.prev
	a.prev = ft
	if prev == 0 {
		return
	}
	dt := min(ft-prev, maxAutoStep).Seconds()
	w, h := r.size()
	dx := a.opts.speed(r.last.X, w) * dt
	dy := a.opts.speed(r.last.Y, h) * dt
	if dx == 0 && dy == 0 {
		return
	}
	r.pan(dx, dy)
	r.pointer(PointerMove, r.last, 0, 0, r.lastMods)
}
