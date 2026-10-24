//go:build darwin

// Package view puts a gfx scene into a gowt window as a widget.
//
// Experimental, like gfx: the API may change without notice.
package view

import (
	"sync/atomic"

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
	dpr    float64
	resize atomic.Bool
}

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
	cv.Unwrap().AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { v.resize.Store(true) }))
	display := cv.Unwrap().GetDisplay()
	link, err := cg.NewDisplayLink(func(fi gfx.FrameInfo) {
		// runs on a Core Video thread: only decide, then hand over to the UI thread
		if v.gone.Load() || !(v.comp.NeedsFrame() || v.resize.Load()) || !v.posted.CompareAndSwap(false, true) {
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
	v.comp.Frame(fi)
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
