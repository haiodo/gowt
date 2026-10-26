//go:build darwin || linux || windows

// Package view puts a gfx scene into a gowt window as a widget.
//
// Experimental, like gfx: the API may change without notice. macOS composites tiles as CALayers;
// Linux composites them on the CPU with cairo into the widget (GTK 3); Windows draws them
// with Direct2D into a DXGI swap chain.
package view

import (
	"os"
	"sync/atomic"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

// View shows a gfx.Scene on a canvas. Its methods are for the UI goroutine, except
// Scene().Update, which any goroutine may call.
type View struct {
	cv       *g.CanvasView
	scene    *gfx.Scene
	comp     *gfx.Compositor
	gone     atomic.Bool
	dpr      float64
	resize   atomic.Bool
	frames   atomic.Uint64 // frames started
	force    atomic.Bool   // a QA capture asked for a frame
	r        *router
	onKey    func(*Key)
	stops    []func()
	report   func(FrameReport)
	prevFi   time.Duration
	paint    time.Duration // the last widget paint, 0 where the OS composites
	overlays []*Overlay
	editor   *Editor
	platform
}

// Tracing is set by GFXDEMO_TRACE: the demos print what the view and the router see.
var Tracing = os.Getenv("GFXDEMO_TRACE") != ""

// Tracef prints a line to stderr when Tracing is set.
func Tracef(format string, a ...any) {
	if Tracing {
		traceOut(format, a...)
	}
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
	// Paint is how long the last widget paint took: the CPU compositing of the tiles on Linux, 0 on macOS.
	Paint time.Duration
}

// PaintTime is how long the last widget paint took: the CPU compositing of the tiles on Linux, 0 on macOS.
func (v *View) PaintTime() time.Duration { return v.paint }

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
		v.r.hud = func(p gfx.Point) *gfx.Node { return v.comp.HUDHit(p, 1) }
		v.registerQA()
		if init != nil {
			init(v)
		}
	}))...)
}

// wantsFrame is what the frame clock asks on every tick, from its own thread.
func (v *View) wantsFrame() bool {
	return v.comp.NeedsFrame() || v.resize.Load() || v.r.acc.wake.Load() || v.r.autoWake.Load() || v.force.Load()
}

func (v *View) frame(fi gfx.FrameInfo) {
	v.frames.Add(1)
	v.force.Store(false)
	if v.resize.Swap(false) {
		w, h := v.cv.Size()
		v.syncScale()
		v.comp.SetViewport(float64(w), float64(h), v.dpr)
	}
	dx, dy, events := v.r.tick(fi.Time)
	v.comp.Frame(fi)
	v.syncOverlays()
	if v.report != nil {
		st, cam := v.comp.Stats(), v.comp.Camera()
		r := FrameReport{Time: fi.Time, CPU: st.FrameTime, Events: events, DX: dx, DY: dy, Origin: cam.Origin, Zoom: cam.Zoom, Tiles: st.TilesApplied, Paint: v.paint}
		if v.prevFi != 0 {
			r.Interval = fi.Time - v.prevFi
		}
		v.prevFi = fi.Time
		v.report(r)
	}
}

// OnResize sets f to run with the view size in points when layout gave it a new one, and once for
// the first size known; see g.CanvasView.OnResize. The camera is yours to set there, for example to
// fit the content to the window the first time.
func (v *View) OnResize(f func(w, h int)) { v.cv.OnResize(f) }

// Canvas is the underlying gowt canvas: mouse listeners, focus, size.
func (v *View) Canvas() *g.CanvasView { return v.cv }

// Scene returns the scene the view shows.
func (v *View) Scene() *gfx.Scene { return v.scene }

// Compositor gives the camera, the lifting of nodes for drags, sprites and statistics.
func (v *View) Compositor() *gfx.Compositor { return v.comp }

// HitTest returns the topmost node under the view point (x, y), with a 4 point tolerance.
func (v *View) HitTest(x, y int) *gfx.Node {
	p := v.comp.SceneAt(gfx.Point{X: float64(x), Y: float64(y)})
	return v.scene.HitTest(p, 4/v.comp.Camera().Zoom)
}

// HUDHitTest returns the topmost node of the heads-up scene under the view point (x, y), or nil;
// it is what to ask before HitTest.
func (v *View) HUDHitTest(x, y int) *gfx.Node {
	return v.comp.HUDHit(gfx.Point{X: float64(x), Y: float64(y)}, 1)
}

func modsOfMask(m int32) g.Mods {
	return g.Mods{Ctrl: m&swt.CTRL != 0, Shift: m&swt.SHIFT != 0, Alt: m&swt.ALT != 0, Cmd: m&swt.COMMAND != 0}
}

// Drag moves nodes with the pointer until the button is released. Call it from the PointerDown
// that starts the drag, with the nodes to move (for example p.Node). The nodes are lifted into the
// active layer and landed at the end, so the tiles under them are not redrawn each move. The
// app's OnPointer callback still gets every event, after the nodes moved: update what depends on
// them there. Releasing outside the view drops the nodes at the nearest visible point; Escape or
// a lost focus puts them back (Aborted). Positions are read from the last committed state, so call
// it from the UI goroutine.
func (v *View) Drag(p *Pointer, nodes ...*gfx.Node) { v.r.beginDrag(v.comp, v.scene, p, nodes) }

// OnPointer sets f to run for every press, move, release and cancel, with the scene point and the
// node under the pointer. While a button is held the moves and the release keep coming outside
// the view. A Ctrl-click is Button 1 with Mods.Ctrl.
func (v *View) OnPointer(f func(*Pointer)) { v.r.onPointer = f }

// AutoScroll pans the camera each frame while the pointer is in the edge band or outside the view;
// call it from the PointerDown that starts a drag. After each step the view repeats the last
// position as a PointerMove, so the app re-places the dragged content. It ends on release, on
// cancel or when stop is called. The drop rule on a release with Outside set is the app's.
func (v *View) AutoScroll(o AutoScrollOptions) (stop func()) { return v.r.startAuto(o) }

// OnScroll sets f to run for scroll gestures and wheel events. f returns true when it used the
// event; otherwise the view pans (zooms with Cmd or Ctrl held).
func (v *View) OnScroll(f func(*Scroll) bool) { v.r.onScroll = f }

// OnMagnify sets f to run for pinch steps. f returns true when it used the event; otherwise the
// view zooms about the pointer.
func (v *View) OnMagnify(f func(*Magnify) bool) { v.r.onMagnify = f }

// OnSmartZoom sets f to run for the two finger double tap. f returns true when it used the event;
// otherwise the view toggles between zoom 1 and the zoom before.
func (v *View) OnSmartZoom(f func(*SmartZoom) bool) { v.r.onSmart = f }

// OnKey sets f to run for key presses and releases while the view has the keyboard focus. A press
// on the view gives it the focus.
func (v *View) OnKey(f func(*Key)) { v.onKey = f }

// Pan moves the content by (dx, dy) view points.
func (v *View) Pan(dx, dy float64) { v.r.pan(dx, dy) }

// SetZoomLimits bounds the zoom the built-in pinch, wheel zoom and smart zoom reach. The default
// is 0.02 to 64.
func (v *View) SetZoomLimits(lo, hi float64) { v.r.minZ, v.r.maxZ = lo, hi }
