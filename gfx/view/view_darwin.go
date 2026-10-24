//go:build darwin

// Package view puts a gfx scene into a gowt window as a widget.
//
// Experimental, like gfx: the API may change without notice.
package view

import (
	"sync/atomic"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// View shows a gfx.Scene on a layer-backed canvas. Its methods are for the UI goroutine, except
// Scene().Update, which any goroutine may call.
type View struct {
	cv     *g.CanvasView
	scene  *gfx.Scene
	comp   *gfx.Compositor
	host   *cg.Host
	link   *cg.DisplayLink
	nsview uintptr
	posted atomic.Bool
	gone   atomic.Bool
	// polling is set while pollRelease has a timer running; UI thread only
	polling bool
	dpr     float64
	resize  atomic.Bool
	r       *router
	onKey   func(*Key)
	stops   []func()
	cursor  map[Cursor]*swt.Cursor
	report  func(FrameReport)
	prevFi  time.Duration
}

// FrameReport describes one frame the view ran, for tuning: the numbers of the scroll path.
type FrameReport struct {
	Time     time.Duration // frame clock
	Interval time.Duration // since the previous frame the view ran, 0 for the first
	CPU      time.Duration // time the compositor took in Frame
	Events   int           // scroll events taken in since the previous frame
	DX, DY   float64       // pan applied by this frame, view points
	Origin   gfx.Point     // camera origin after the frame
	Zoom     float64
	Tiles    int // tile images put on layers by this frame
}

// OnFrameReport sets f to run after every frame the view runs, on the UI goroutine.
func (v *View) OnFrameReport(f func(FrameReport)) { v.report = f }

// New returns an Option that creates the scene canvas in the widget tree, like g.Canvas does. init
// runs on the UI goroutine once the view exists; it gets the View to set the camera and add overlay
// sprites. A frame clock failure (no display) goes to App.OnError.
func New(scene *gfx.Scene, init func(*View), opts ...g.Option) g.Option {
	return g.Canvas(nil, append(opts, g.Init(func(cv *g.CanvasView) {
		v, err := attach(cv, scene)
		if err != nil {
			panic(err)
		}
		if init != nil {
			init(v)
		}
	}))...)
}

func attach(cv *g.CanvasView, scene *gfx.Scene) (*View, error) {
	v := &View{cv: cv, scene: scene, nsview: uintptr(cv.Unwrap().View.Id)}
	v.dpr = cg.BackingScale(v.nsview)
	v.host = cg.NewHost(cg.ViewLayer(v.nsview), v.dpr)
	v.comp = gfx.NewCompositor(scene, cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: true}), v.host, gfx.Options{})
	v.comp.SetCamera(gfx.Camera{Zoom: 1})
	w, h := cv.Size()
	v.comp.SetViewport(float64(w), float64(h), v.dpr)
	v.wireInput()
	cv.Unwrap().AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { v.resize.Store(true) }))
	display := cv.Unwrap().GetDisplay()
	link, err := cg.NewDisplayLink(func(fi gfx.FrameInfo) {
		// runs on a Core Video thread: only decide, then hand over to the UI thread
		if v.gone.Load() || !(v.comp.NeedsFrame() || v.resize.Load() || v.r.acc.wake.Load() || v.r.autoWake.Load()) || !v.posted.CompareAndSwap(false, true) {
			return
		}
		defer func() { _ = recover() }() // the display may be gone as the window closes
		display.AsyncExec(jrt.NewRunnable(func() {
			v.posted.Store(false)
			if !v.gone.Load() {
				v.frame(fi)
			}
		}))
	})
	if err != nil {
		return nil, err
	}
	v.link = link
	cv.OnDispose(func() {
		v.gone.Store(true)
		for _, stop := range v.stops {
			stop()
		}
		v.link.Close()
		v.comp.Close()
		v.host.Close()
	})
	return v, link.Start()
}

func (v *View) frame(fi gfx.FrameInfo) {
	if v.resize.Swap(false) {
		w, h := v.cv.Size()
		if d := cg.BackingScale(v.nsview); d != v.dpr {
			v.dpr = d
			v.host.SetScale(d)
		}
		v.comp.SetViewport(float64(w), float64(h), v.dpr)
	}
	dx, dy, events := v.r.tick(fi.Time)
	v.comp.Frame(fi)
	if v.report != nil {
		st, cam := v.comp.Stats(), v.comp.Camera()
		r := FrameReport{Time: fi.Time, CPU: st.FrameTime, Events: events, DX: dx, DY: dy, Origin: cam.Origin, Zoom: cam.Zoom, Tiles: st.TilesApplied}
		if v.prevFi != 0 {
			r.Interval = fi.Time - v.prevFi
		}
		v.prevFi = fi.Time
		v.report(r)
	}
}

// Canvas is the underlying gowt canvas: mouse listeners, focus, size.
func (v *View) Canvas() *g.CanvasView { return v.cv }

func (v *View) Scene() *gfx.Scene { return v.scene }

// Compositor gives the camera, the lifting of nodes for drags, sprites and statistics.
func (v *View) Compositor() *gfx.Compositor { return v.comp }

// HitTest returns the topmost node under the view point (x, y), with a 4 point tolerance.
func (v *View) HitTest(x, y int) *gfx.Node {
	p := v.comp.SceneAt(gfx.Point{X: float64(x), Y: float64(y)})
	return v.scene.HitTest(p, 4/v.comp.Camera().Zoom)
}
