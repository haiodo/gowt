package gfx

import (
	"math"
	"time"
)

// Easing maps linear progress 0..1 to eased progress; it must give 0 at 0 and 1 at 1.
type Easing func(t float64) float64

// The easings. Linear is constant speed; EaseIn starts slow, EaseOut ends slow, EaseInOut does
// both.
var (
	Linear    Easing = func(t float64) float64 { return t }
	EaseIn    Easing = func(t float64) float64 { return t * t * t }
	EaseOut   Easing = func(t float64) float64 { u := 1 - t; return 1 - u*u*u }
	EaseInOut Easing = func(t float64) float64 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		u := -2*t + 2
		return 1 - u*u*u/2
	}
)

// CubicBezier is the CSS timing function cubic-bezier(x1, y1, x2, y2).
func CubicBezier(x1, y1, x2, y2 float64) Easing {
	bez := func(a, b, t float64) float64 {
		u := 1 - t
		return 3*u*u*t*a + 3*u*t*t*b + t*t*t
	}
	return func(t float64) float64 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		// solve x(s) = t by bisection; x is monotonic for x1, x2 in 0..1
		lo, hi := 0.0, 1.0
		for i := 0; i < 40; i++ {
			mid := (lo + hi) / 2
			if bez(x1, x2, mid) < t {
				lo = mid
			} else {
				hi = mid
			}
		}
		return bez(y1, y2, (lo+hi)/2)
	}
}

type animProp uint8

const (
	propPos animProp = iota
	propOpacity
	propRotation
	propScale
	propAlong
)

// anim is one running animation of a sprite property. Time starts at the first frame that sees it,
// so an animation begun between frames runs in step with the display.
type anim struct {
	prop     animProp
	from, to Point // Pos; X alone for the scalar properties
	dur      time.Duration
	ease     Easing
	path     *Measure
	orient   bool
	start    time.Duration
	started  bool
	done     func()
}

// step applies the animation at frame time now and reports whether it has finished.
func (a *anim) step(now time.Duration, sp *spriteState) bool {
	if !a.started {
		a.start, a.started = now, true
		switch a.prop {
		case propPos:
			a.from = sp.pos
		case propOpacity:
			a.from.X = sp.opacity
		case propRotation:
			a.from.X = sp.rotation
		case propScale:
			a.from.X = sp.scale
		}
	}
	t := 1.0
	if a.dur > 0 {
		t = math.Min(math.Max(float64(now-a.start)/float64(a.dur), 0), 1)
	}
	e := a.ease(t)
	switch a.prop {
	case propPos:
		sp.pos = lerp(a.from, a.to, e)
	case propOpacity:
		sp.opacity = a.from.X + (a.to.X-a.from.X)*e
	case propRotation:
		sp.rotation = a.from.X + (a.to.X-a.from.X)*e
	case propScale:
		sp.scale = a.from.X + (a.to.X-a.from.X)*e
	case propAlong:
		p, dir := a.path.At(a.path.Length() * e)
		sp.pos = p
		if a.orient {
			sp.rotation = math.Atan2(dir.Y, dir.X)
		}
	}
	return t >= 1
}
