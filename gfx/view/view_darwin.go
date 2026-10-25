//go:build darwin

package view

import (
	"sync/atomic"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// platform is what the macOS View adds: the layer host and the display link.
type platform struct {
	host   *cg.Host
	link   *cg.DisplayLink
	nsview uintptr
	posted atomic.Bool
	// polling is set while pollRelease has a timer running; UI thread only
	polling bool
	cursor  map[Cursor]*swt.Cursor
}

func traceOut(format string, a ...any) { cg.Tracef(format, a...) }

func attach(cv *g.CanvasView, scene *gfx.Scene) (*View, error) {
	v := &View{cv: cv, scene: scene}
	v.nsview = uintptr(cv.Unwrap().View.Id)
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

// syncScale follows the window to a screen with another backing scale.
func (v *View) syncScale() {
	if d := cg.BackingScale(v.nsview); d != v.dpr {
		v.dpr = d
		v.host.SetScale(d)
	}
}
