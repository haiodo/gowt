package main

import (
	"math"
	"sort"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/swt"
)

// chartSpec says what a history chart shows.
type chartSpec struct {
	names    []string
	stacked  bool
	total    string // tooltip row with the sum of a stacked chart
	fixedMax float64
	minMax   float64 // auto axis: the smallest top value
	fracs    []float64
	widest   string // the widest axis label, to size the label column
	color    func(p *palette, i int) gfx.Color
	axis     func(float64) string // gridline labels
	value    func(float64) string // tooltip values
}

const (
	animDur  = 350 * time.Millisecond // one sample step, and the pulse that marks it
	rightPad = 14.0                   // empty space right of the newest point, points
)

// frameGap is the least time between two frames that move the chart.
const frameGap = 20 * time.Millisecond

var stepEase = gfx.CubicBezier(0.25, 0.0, 0.2, 1)

// axisKey is what the axis labels depend on; the picture is recorded again only when it changes.
type axisKey struct {
	w, h, yMax float64
	pal        *palette
}

// chart is a scrolling history chart: the tape is one scene node wider than the view and scrolling
// is the camera moving over it. Labels, live dots and the crosshair are overlay sprites.
type chart struct {
	spec    chartSpec
	pal     *palette
	scene   *gfx.Scene
	node    *gfx.Node
	v       *view.View
	w, h    float64
	margin  float64 // the label column left of the plot
	window  time.Duration
	t0      time.Time
	ts      []float64
	vals    [][]float64
	curves  []spline
	pic     *gfx.Picture
	axis    *gfx.Picture
	axisKey axisKey

	// animation, in seconds on the tape and in frame time
	tr, trFrom, trTo float64
	animStart        time.Duration
	animFresh        bool
	yMax, yTarget    float64
	lastFrame        time.Duration
	lastPlaced       time.Duration
	ticking          bool

	axisSp, lineSp, bubbleSp, ringSp *gfx.Sprite
	live, dots                       []*gfx.Sprite
	hover                            *gfx.Point
	hoverKey, bubbleW                float64
	hoverShown, liveShown            bool
}

func newChart(spec chartSpec, pal *palette, window time.Duration) *chart {
	c := &chart{spec: spec, pal: pal, window: window, yMax: max(spec.fixedMax, spec.minMax)}
	if c.yMax == 0 {
		c.yMax = 1
	}
	c.yTarget = c.yMax
	c.vals = make([][]float64, len(spec.names))
	c.scene = gfx.NewScene()
	c.node = gfx.NewNode(1)
	c.scene.Update(func(tx *gfx.Tx) { tx.Add(nil, c.node) })
	return c
}

// option is the widget; it fills its cell.
func (c *chart) option(minW, minH int) g.Option {
	return view.New(c.scene, c.attach, g.Grow(), g.Size(minW, minH))
}

func (c *chart) attach(v *view.View) {
	c.v = v
	v.SetZoomLimits(1, 1)
	// the built-in pan and zoom would move the tape under the user's fingers
	v.OnScroll(func(*view.Scroll) bool { return true })
	v.OnMagnify(func(*view.Magnify) bool { return true })
	v.OnSmartZoom(func(*view.SmartZoom) bool { return true })
	v.OnPointer(func(p *view.Pointer) {
		if p.Kind == view.PointerMove {
			c.setHover(&gfx.Point{X: p.Pos.X, Y: p.Pos.Y})
		}
	})
	cv := v.Canvas().Unwrap()
	cv.AddMouseTrackListener(swt.MouseTrackListenerMouseExitAdapter(func(*swt.MouseEvent) { c.setHover(nil) }))
	cv.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { c.resize() }))
	comp := v.Compositor()
	n := len(c.spec.names)
	c.axisSp = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
	c.lineSp = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
	c.bubbleSp = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
	for i := 0; i < n; i++ {
		c.dots = append(c.dots, comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{}))
		c.live = append(c.live, comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{}))
	}
	c.ringSp = comp.NewSprite(nil, gfx.SpaceScreen, gfx.Point{})
	for _, s := range append([]*gfx.Sprite{c.lineSp, c.bubbleSp, c.ringSp}, append(c.dots, c.live...)...) {
		s.SetOpacity(0)
	}
	c.resize()
}

func (c *chart) resize() {
	w, h := c.v.Canvas().Size()
	if float64(w) == c.w && float64(h) == c.h || w <= 0 || h <= 0 {
		return
	}
	c.w, c.h = float64(w), float64(h)
	c.restyle()
}

func (c *chart) pps() float64 { return (c.w - c.margin - rightPad) / c.window.Seconds() }

// setPalette swaps colours; pictures are recorded again.
func (c *chart) setPalette(p *palette) {
	c.pal = p
	c.restyle()
}

func (c *chart) setWindow(d time.Duration) {
	c.window = d
	c.pruneFor()
	c.updateTarget(true)
	c.restyle()
}

// restyle records everything that depends on size, palette or window again.
func (c *chart) restyle() {
	if c.w == 0 {
		return
	}
	if c.v != nil {
		c.pal.setCanvasBackground(c.v.Canvas().Unwrap())
	}
	c.margin = gfx.DefaultMeasurer.MeasureText(c.spec.widest, c.pal.small).Width + 18
	c.rerecord()
	if c.v == nil || c.axisSp == nil {
		return
	}
	n := len(c.spec.names)
	dot := func(r float64, fill gfx.Color, ring float64, edge gfx.Color) *gfx.Picture {
		rec := gfx.NewRecorder()
		rec.FillEllipse(gfx.R(-r, -r, 2*r, 2*r), fill)
		if ring > 0 {
			rec.StrokeEllipse(gfx.R(-r, -r, 2*r, 2*r), gfx.StrokeStyle{Width: ring}, edge)
		}
		return rec.Finish()
	}
	for i := 0; i < n; i++ {
		col := c.spec.color(c.pal, i)
		c.live[i].SetPicture(dot(4, col, 2, c.pal.card))
		c.dots[i].SetPicture(dot(4.5, col, 2, c.pal.bubble))
	}
	last := c.spec.color(c.pal, n-1)
	c.ringSp.SetPicture(dot(5, gfx.Color{}, 1.5, last))
	rec := gfx.NewRecorder()
	var l gfx.Path
	l.MoveTo(0, 0)
	l.LineTo(0, c.h)
	rec.StrokePath(&l, gfx.StrokeStyle{Width: 1}, c.pal.label)
	c.lineSp.SetPicture(rec.Finish())
	c.hoverKey = math.NaN()
	c.place()
}

func (c *chart) params() *tapeParams {
	pps := c.pps()
	step := 10.0
	switch {
	case c.window > 10*time.Minute:
		step = 180
	case c.window > 2*time.Minute:
		step = 60
	}
	cols := make([]gfx.Color, len(c.spec.names))
	for i := range cols {
		cols[i] = c.spec.color(c.pal, i)
	}
	return &tapeParams{
		pal: c.pal, colors: cols, stacked: c.spec.stacked, w: c.w, h: c.h, margin: c.margin, pps: pps, yMax: c.yMax,
		fracs: c.spec.fracs, tickStep: step, base: c.t0,
		xLo: min(c.trFrom, c.trTo) - (c.w-rightPad)/pps - 2, xHi: c.trTo + rightPad/pps + 2,
		visLo: c.trTo*pps - (c.w - rightPad) + c.margin, visHi: c.trTo*pps + rightPad,
	}
}

// rerecord makes the tape picture from the samples and puts it on the node.
func (c *chart) rerecord() {
	p := c.params()
	res := recordTape(p, c.ts, c.vals)
	c.pic, c.curves = res.pic, res.curves
	pic := c.pic
	c.scene.Update(func(tx *gfx.Tx) { tx.SetPicture(c.node, pic) })
	if key := (axisKey{c.w, c.h, c.yMax, c.pal}); key != c.axisKey || c.axis == nil {
		c.axisKey = key
		c.axis = recordAxis(c.pal, p, c.spec.axis)
		if c.axisSp != nil {
			c.axisSp.SetPicture(c.axis)
		}
	}
}

// push adds a sample taken at t and starts the step to it.
func (c *chart) push(t time.Time, vals ...float64) {
	if c.t0.IsZero() {
		c.t0 = t
	}
	s := t.Sub(c.t0).Seconds()
	c.ts = append(c.ts, s)
	for i, v := range vals {
		c.vals[i] = append(c.vals[i], v)
	}
	if len(c.ts) == 1 {
		c.tr, c.trFrom = s, s
	} else {
		c.trFrom = c.tr
	}
	c.trTo = s
	// the tape coordinates stay small enough for float32 pictures
	if c.trTo*c.pps() > 2e5 {
		c.rebase()
	}
	c.pruneFor()
	c.updateTarget(false)
	if c.w > 0 {
		c.rerecord()
	}
	c.animFresh = true
	if c.ringSp != nil {
		c.ringSp.SetScale(1)
		c.ringSp.SetOpacity(0.9)
		c.ringSp.ScaleTo(2.6, animDur, gfx.EaseOut, nil)
		c.ringSp.FadeTo(0, animDur, gfx.EaseOut, nil)
	}
	c.wake()
}

func (c *chart) rebase() {
	d := c.trTo
	c.t0 = c.t0.Add(time.Duration(d * float64(time.Second)))
	for i := range c.ts {
		c.ts[i] -= d
	}
	c.tr, c.trFrom, c.trTo = c.tr-d, c.trFrom-d, 0
}

// pruneFor drops the samples that left the window, keeping one before the left edge.
func (c *chart) pruneFor() {
	if len(c.ts) == 0 {
		return
	}
	edge := c.trTo - c.window.Seconds() - 2
	k := sort.SearchFloat64s(c.ts, edge)
	if k > 1 {
		k--
		c.ts = append(c.ts[:0], c.ts[k:]...)
		for i := range c.vals {
			c.vals[i] = append(c.vals[i][:0], c.vals[i][k:]...)
		}
	}
}

// updateTarget sets the axis top for an auto-scaled chart: it grows at once and shrinks only when
// the data would fill under half of it, so the axis does not breathe.
func (c *chart) updateTarget(force bool) {
	if c.spec.fixedMax > 0 {
		return
	}
	top := 0.0
	for k := range c.ts {
		for i := range c.vals {
			top = max(top, c.vals[i][k])
		}
	}
	want := niceCeil(max(top*1.15, c.spec.minMax))
	if want > c.yTarget || want < c.yTarget/2 || force {
		c.yTarget = want
	}
	if want := c.yTarget; c.yMax < want || force && c.v == nil {
		c.yMax = want
	}
}

// wake makes the view run frames until the step and the axis have settled.
func (c *chart) wake() {
	if c.v == nil || c.ticking {
		return
	}
	c.ticking = true
	c.v.Compositor().OnFrame(c.frame)
}

func (c *chart) frame(fi gfx.FrameInfo) {
	if c.animFresh {
		c.animStart, c.animFresh = fi.Time, false
	}
	t := min(float64(fi.Time-c.animStart)/float64(animDur), 1)
	if t < 1 && c.yMax == c.yTarget && fi.Time-c.lastPlaced < frameGap {
		return // a step needs no more than about 45 frames a second; a skipped frame costs nothing
	}
	c.lastPlaced = fi.Time
	c.tr = c.trFrom + (c.trTo-c.trFrom)*stepEase(t)
	settled := t >= 1
	if c.yMax != c.yTarget {
		dt := (fi.Time - c.lastFrame).Seconds()
		if dt <= 0 || dt > 0.1 {
			dt = 1.0 / 60
		}
		c.yMax += (c.yTarget - c.yMax) * (1 - math.Exp(-dt/0.12))
		if math.Abs(c.yMax-c.yTarget) < 0.002*c.yTarget {
			c.yMax = c.yTarget
		}
		c.rerecord()
		settled = false
	}
	c.lastFrame = fi.Time
	c.place()
	if settled && !c.animFresh {
		c.ticking = false
		c.v.Compositor().OnFrame(nil)
	}
}

// place puts the camera and the sprites where the animation is.
func (c *chart) place() {
	if c.v == nil {
		return
	}
	pps := c.pps()
	right := c.w - rightPad
	c.v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: c.tr*pps - right}, Zoom: 1})
	for i, s := range c.live {
		if i < len(c.curves) && len(c.ts) > 1 {
			s.SetPos(gfx.Point{X: right, Y: c.yOf(c.curves[i].at(c.tr))})
			if !c.liveShown { // before the first samples the dots would sit in the plot's corner
				s.SetOpacity(1)
			}
		}
	}
	c.liveShown = c.liveShown || len(c.ts) > 1
	if c.ringSp != nil && len(c.curves) > 0 && len(c.ts) > 1 {
		c.ringSp.SetPos(c.live[len(c.live)-1].Pos())
	}
	c.placeHover()
}

func (c *chart) yOf(v float64) float64 {
	return plotTop + (1-v/c.yMax)*(c.h-strip-plotTop)
}

func (c *chart) setHover(p *gfx.Point) {
	if p != nil && snapping {
		return // a screenshot shows no crosshair, whatever the pointer rests on
	}
	c.hover = p
	if p == nil {
		c.hoverShown = false
		for _, s := range append([]*gfx.Sprite{c.lineSp, c.bubbleSp}, c.dots...) {
			s.FadeTo(0, 120*time.Millisecond, gfx.EaseOut, nil)
		}
		return
	}
	c.placeHover() // sprite changes ask for their own frame; no frame loop while the pointer rests
}

// placeHover moves the crosshair to the sample nearest the pointer. The data scrolls under a
// standing pointer, so this runs on every frame while the pointer is over the chart.
func (c *chart) placeHover() {
	if c.hover == nil || len(c.ts) == 0 {
		return
	}
	pps := c.pps()
	right := c.w - rightPad
	at := c.tr - (right-c.hover.X)/pps
	k := sort.SearchFloat64s(c.ts, at)
	if k > 0 && (k == len(c.ts) || at-c.ts[k-1] < c.ts[k]-at) {
		k--
	}
	x := right - (c.tr-c.ts[k])*pps
	if x < c.margin || x > right+1 {
		if c.hoverShown {
			c.hoverShown = false
			for _, s := range append([]*gfx.Sprite{c.lineSp, c.bubbleSp}, c.dots...) {
				s.FadeTo(0, 120*time.Millisecond, gfx.EaseOut, nil)
			}
		}
		return
	}
	if !c.hoverShown {
		c.hoverShown = true
		for _, s := range append([]*gfx.Sprite{c.lineSp, c.bubbleSp}, c.dots...) {
			s.FadeTo(1, 120*time.Millisecond, gfx.EaseOut, nil)
		}
	}
	x = math.Round(x) + 0.5
	c.lineSp.SetPos(gfx.Point{X: x})
	for i, d := range c.dots {
		v := c.vals[i][k]
		if c.spec.stacked {
			v = c.curves[i].ys[k]
		}
		d.SetPos(gfx.Point{X: x, Y: c.yOf(v)})
	}
	if c.ts[k] != c.hoverKey {
		c.hoverKey = c.ts[k]
		c.bubbleSp.SetPicture(c.bubble(k))
	}
	bw := c.bubbleW
	bx := x + 12
	if bx+bw > c.w-4 {
		bx = x - 12 - bw
	}
	c.bubbleSp.SetPos(gfx.Point{X: bx, Y: 6})
}

// bubble is the tooltip for sample k: the time and one row per series.
func (c *chart) bubble(k int) *gfx.Picture {
	p := c.pal
	type row struct {
		name, val string
		col       gfx.Color
		dot       bool
	}
	var rows []row
	if c.spec.stacked && c.spec.total != "" {
		rows = append(rows, row{c.spec.total, c.spec.value(c.curves[len(c.curves)-1].ys[k]), p.text, false})
	}
	for i := len(c.spec.names) - 1; i >= 0; i-- {
		rows = append(rows, row{c.spec.names[i], c.spec.value(c.vals[i][k]), c.spec.color(p, i), true})
	}
	m := gfx.DefaultMeasurer
	title := c.t0.Add(time.Duration(c.ts[k] * float64(time.Second))).Format("15:04:05")
	lineH, padX, padY, dotW := p.regular.Size+5, 10.0, 8.0, 14.0
	nameW, valW := 0.0, 0.0
	for _, r := range rows {
		nameW = max(nameW, m.MeasureText(r.name, p.regular).Width)
		valW = max(valW, m.MeasureText(r.val, p.strong).Width)
	}
	w := max(padX*2+dotW+nameW+16+valW, padX*2+m.MeasureText(title, p.small).Width)
	h := padY*2 + lineH*float64(len(rows)+1) - 2
	c.bubbleW = w
	rec := gfx.NewRecorder()
	rec.FillShadowRRect(0, 0, w, h, 9, p.bubble, gfx.Shadow{DY: 3, Blur: 10, Color: gfx.Color{A: 0.28}})
	rec.StrokeRRect(gfx.R(0.5, 0.5, w-1, h-1), 9, gfx.StrokeStyle{Width: 1}, p.bubbleEdge)
	y := padY + p.small.Size
	rec.Text(title, padX, y, p.small, p.label)
	for _, r := range rows {
		y += lineH
		if r.dot {
			rec.FillEllipse(gfx.R(padX+0.5, y-p.regular.Size*0.36-3.5, 7, 7), r.col)
		}
		rec.Text(r.name, padX+dotW, y, p.regular, p.text)
		rec.Text(r.val, w-padX-m.MeasureText(r.val, p.strong).Width, y, p.strong, p.text)
	}
	return rec.Finish()
}
