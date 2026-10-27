package main

import (
	"math"
	"sort"
	"time"

	"github.com/haiodo/gowt/gfx"
)

// spline is a monotone cubic Hermite interpolation (Fritsch-Butland): smooth like a curve fit, but it
// never overshoots between samples, so a CPU chart does not dip below zero or past 100%.
type spline struct{ xs, ys, m []float64 }

func newSpline(xs, ys []float64) spline {
	n := len(xs)
	s := spline{xs: xs, ys: ys, m: make([]float64, n)}
	if n < 2 {
		return s
	}
	d := make([]float64, n-1)
	for i := range d {
		d[i] = (ys[i+1] - ys[i]) / (xs[i+1] - xs[i])
	}
	s.m[0], s.m[n-1] = d[0], d[n-2]
	for i := 1; i < n-1; i++ {
		if d[i-1]*d[i] <= 0 {
			continue
		}
		h0, h1 := xs[i]-xs[i-1], xs[i+1]-xs[i]
		s.m[i] = 3 * (h0 + h1) / ((2*h1+h0)/d[i-1] + (h1+2*h0)/d[i])
	}
	return s
}

// at evaluates the curve; x outside the samples gives the nearest end.
func (s spline) at(x float64) float64 {
	n := len(s.xs)
	switch {
	case n == 0:
		return 0
	case n == 1 || x <= s.xs[0]:
		return s.ys[0]
	case x >= s.xs[n-1]:
		return s.ys[n-1]
	}
	i := sort.SearchFloat64s(s.xs, x) - 1
	h := s.xs[i+1] - s.xs[i]
	t := (x - s.xs[i]) / h
	t2, t3 := t*t, t*t*t
	return (2*t3-3*t2+1)*s.ys[i] + (t3-2*t2+t)*h*s.m[i] + (-2*t3+3*t2)*s.ys[i+1] + (t3-t2)*h*s.m[i+1]
}

// forward appends the curve from the first sample to the last; the path is already at the first.
func (s spline) forward(p *gfx.Path) {
	for i := 0; i+1 < len(s.xs); i++ {
		h := (s.xs[i+1] - s.xs[i]) / 3
		p.CubicTo(s.xs[i]+h, s.ys[i]+s.m[i]*h, s.xs[i+1]-h, s.ys[i+1]-s.m[i+1]*h, s.xs[i+1], s.ys[i+1])
	}
}

// backward appends the curve from the last sample to the first; the path is already at the last.
func (s spline) backward(p *gfx.Path) {
	for i := len(s.xs) - 2; i >= 0; i-- {
		h := (s.xs[i+1] - s.xs[i]) / 3
		p.CubicTo(s.xs[i+1]-h, s.ys[i+1]-s.m[i+1]*h, s.xs[i]+h, s.ys[i]+s.m[i]*h, s.xs[i], s.ys[i])
	}
}

// tapeParams describes one chart tape: the scene is in view points, x in seconds times pps.
type tapeParams struct {
	pal      *palette
	colors   []gfx.Color
	stacked  bool
	w, h     float64   // view size in points
	margin   float64   // width of the column that holds the value labels, left of the plot
	pps      float64   // points per second
	yMax     float64   // value at the top gridline
	fracs    []float64 // gridlines, as fractions of yMax
	tickStep float64   // seconds between the vertical ticks
	base     time.Time // wall time of t = 0, for the tick labels
	xLo, xHi float64   // seconds the background and grid cover
	visLo    float64   // tape x of the plot's left and right edge once the step has settled; 0, 0 draws every label
	visHi    float64
}

const (
	plotTop = 10.0 // room above the highest gridline for half of its label
	strip   = 20.0 // the band under the plot that holds the time labels
)

func (p *tapeParams) y(v float64) float64 {
	return plotTop + (1-v/p.yMax)*(p.h-strip-plotTop)
}

// tapeResult is the picture and the curves it was drawn from, in value space, for the live dot.
type tapeResult struct {
	pic    *gfx.Picture
	curves []spline // top boundary of each series: cumulative when stacked
}

// recordTape draws the whole tape: background, grid, time ticks with labels, then the series.
// ts are seconds from the tape origin; vals[i][k] is series i at sample k.
func recordTape(p *tapeParams, ts []float64, vals [][]float64) tapeResult {
	pal := p.pal
	r := gfx.NewRecorder()
	x0, x1 := p.xLo*p.pps, p.xHi*p.pps
	r.FillRect(gfx.R(x0, 0, x1-x0, p.h), pal.card)

	tick := gfx.StrokeStyle{Width: 1}
	off := float64(p.base.UnixNano()) / 1e9 // ticks fall on whole minutes and seconds of the clock
	layout := "15:04"
	if p.tickStep < 60 {
		layout = "15:04:05"
	}
	every := labelEvery(p.tickStep*p.pps, gfx.DefaultMeasurer.MeasureText(layout, pal.small).Width+12)
	lo, hi := math.Inf(-1), math.Inf(1)
	if p.visHi != 0 {
		lo, hi = p.visLo, p.visHi
	}
	var labels []timeLabel
	for t := math.Ceil((p.xLo+off)/p.tickStep)*p.tickStep - off; t <= p.xHi; t += p.tickStep {
		x := math.Round(t*p.pps) + 0.5
		var l gfx.Path
		l.MoveTo(x, plotTop)
		l.LineTo(x, p.h-strip+4)
		r.StrokePath(&l, tick, pal.tick)
		at := p.base.Add(time.Duration(t * float64(time.Second)))
		// a label on every second tick at most: Pango text is the slow part of a Linux tile
		if k := math.Round((float64(at.UnixNano())/1e9)/p.tickStep) * p.tickStep; math.Mod(k/p.tickStep, every) == 0 {
			if s := at.Format(layout); x >= lo && x <= hi {
				w := gfx.DefaultMeasurer.MeasureText(s, pal.small).Width
				labels = append(labels, timeLabel{s, x - w/2, w})
			}
		}
	}
	for _, l := range placeLabels(labels, lo, hi) {
		r.Text(l.s, l.x, p.h-5, pal.small, pal.label)
	}
	for _, f := range p.fracs {
		y := math.Round(p.y(f*p.yMax)) + 0.5
		var l gfx.Path
		l.MoveTo(x0, y)
		l.LineTo(x1, y)
		r.StrokePath(&l, tick, pal.grid)
	}

	res := tapeResult{curves: make([]spline, len(vals))}
	n := len(ts)
	if n >= 2 {
		xs := make([]float64, n)
		for i, t := range ts {
			xs[i] = t * p.pps
		}
		tops := make([][]float64, len(vals)) // value space, cumulative when stacked
		for i := range vals {
			tops[i] = make([]float64, n)
			for k := range ts {
				tops[i][k] = vals[i][k]
				if p.stacked && i > 0 {
					tops[i][k] += tops[i-1][k]
				}
			}
			res.curves[i] = newSpline(ts, tops[i])
		}
		pix := func(v []float64) []float64 {
			out := make([]float64, n)
			for k := range v {
				out[k] = p.y(v[k])
			}
			return out
		}
		base := make([]float64, n)
		for k := range base {
			base[k] = p.y(0)
		}
		// the first series is drawn last so that, unstacked, it lies on top of the others
		for i := len(vals) - 1; i >= 0; i-- {
			top := newSpline(xs, pix(tops[i]))
			var area gfx.Path
			area.MoveTo(xs[0], top.ys[0])
			top.forward(&area)
			if p.stacked && i > 0 {
				low := newSpline(xs, pix(tops[i-1]))
				area.LineTo(xs[n-1], low.ys[n-1])
				low.backward(&area)
			} else {
				area.LineTo(xs[n-1], base[0])
				area.LineTo(xs[0], base[0])
			}
			area.Close()
			c := p.colors[i]
			r.FillGradient(&area, gfx.NewLinearGradient(gfx.Point{Y: plotTop}, gfx.Point{Y: p.h - strip},
				gfx.Stop{Offset: 0, Color: c.WithAlpha(0.62)}, gfx.Stop{Offset: 1, Color: c.WithAlpha(0.07)}))
			var line gfx.Path
			line.MoveTo(xs[0], top.ys[0])
			top.forward(&line)
			r.StrokePath(&line, gfx.StrokeStyle{Width: 1.75, Cap: gfx.CapRound, Join: gfx.JoinRound}, c)
		}
	}
	res.pic = r.Finish()
	return res
}

// recordAxis draws the column of value labels left of the plot. It covers the tape under it, so the data
// never runs under a label, and stays put while the tape moves.
func recordAxis(pal *palette, p *tapeParams, label func(float64) string) *gfx.Picture {
	r := gfx.NewRecorder()
	r.FillRect(gfx.R(0, 0, p.margin, p.h), pal.card)
	// from the top down, so the highest gridline keeps its label; a label that touches the one above is left out
	last := math.Inf(-1)
	for i := len(p.fracs) - 1; i >= 0; i-- {
		f := p.fracs[i]
		y := math.Round(p.y(f * p.yMax))
		if f == 0 || y-last < pal.small.Size*1.25 {
			continue
		}
		last = y
		s := label(f * p.yMax)
		r.Text(s, p.margin-8-gfx.DefaultMeasurer.MeasureText(s, pal.small).Width, y+pal.small.Size*0.36, pal.small, pal.label)
	}
	return r.Finish()
}

// labelEvery is how many ticks apart the time labels are, so that labels labelW wide do not touch
// on ticks gap apart; the steps keep the labels on round times (20, 30, 60 s for 10 s ticks).
func labelEvery(gap, labelW float64) float64 {
	for _, n := range []float64{2, 3, 6} {
		if n*gap >= labelW {
			return n
		}
	}
	return 12
}

// timeLabel is a time label at x, w wide, centred under its tick.
type timeLabel struct {
	s    string
	x, w float64
}

// placeLabels moves a label the plot edge [lo, hi] would cut inside it (dropped, a narrow plot kept
// one label) and leaves out a label that would touch the one before it.
func placeLabels(ls []timeLabel, lo, hi float64) []timeLabel {
	var out []timeLabel
	right := math.Inf(-1)
	for _, l := range ls {
		l.x = max(min(l.x, hi-l.w), lo)
		if l.x < right+4 {
			continue
		}
		out = append(out, l)
		right = l.x + l.w
	}
	return out
}

// niceCeil rounds v up to 1, 2, 2.5, 5 or 10 times a power of ten: the axis moves in readable steps.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	e := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if v <= m*e {
			return m * e
		}
	}
	return 10 * e
}
