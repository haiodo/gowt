//go:build darwin

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

// platform is what the macOS View adds: the layer host, the shared frame clock subscription and
// the window visibility.
type platform struct {
	host   *cg.Host
	nsview uintptr
	posted atomic.Bool
	hidden atomic.Bool // the window is miniaturized or fully occluded
	// occlAt is the last visibility re-check, on the frame clock; the CV thread only
	occlAt time.Duration
	// polling is set while pollRelease has a timer running; UI thread only
	polling bool
	cursor  map[Cursor]*swt.Cursor
}

func traceOut(format string, a ...any) { cg.Tracef(format, a...) }

func attach(cv *g.CanvasView, scene *gfx.Scene, o gfx.Options) (*View, error) {
	v := &View{cv: cv, scene: scene}
	v.nsview = uintptr(cv.Unwrap().View.Id)
	v.dpr = cg.BackingScale(v.nsview)
	v.host = cg.NewHost(cg.ViewLayer(v.nsview), v.dpr)
	v.comp = gfx.NewCompositor(scene, cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: true}), v.host, o)
	v.comp.SetCamera(gfx.Camera{Zoom: 1})
	w, h := cv.Size()
	v.comp.SetViewport(float64(w), float64(h), v.dpr)
	v.wireInput()
	cv.Unwrap().AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { v.resize.Store(true) }))
	display := cv.Unwrap().GetDisplay()
	// one frame clock serves every window: the callback runs on a Core Video thread, decides, and
	// hands the frame over to the UI thread
	visChanged := func(visible bool) {
		if v.hidden.Swap(!visible) && visible {
			// one frame to repaint what the hidden window missed
			v.resize.Store(true)
		}
	}
	v.stops = append(v.stops, cg.SubscribeFrames(func(fi gfx.FrameInfo) {
		// the occlusion notification is not always delivered (observed with several windows shown in
		// one burst) and occlusionState lags: re-check lazily, once a second per window
		if d := fi.Time - v.occlAt; d >= time.Second || d < 0 {
			v.occlAt = fi.Time
			cg.RecheckVisibility()
		}
		if v.gone.Load() || !v.wantsFrame() || !v.posted.CompareAndSwap(false, true) {
			return
		}
		defer func() { _ = recover() }() // the display may be gone as the window closes
		display.AsyncExec(jrt.NewRunnable(func() {
			v.posted.Store(false)
			if !v.gone.Load() {
				v.frame(fi)
			}
		}))
	}))
	v.stops = append(v.stops, cg.OnWindowVisibility(v.nsview, visChanged))
	cv.OnDispose(func() {
		v.gone.Store(true)
		for _, stop := range v.stops {
			stop()
		}
		v.comp.Close()
		v.host.Close()
	})
	return v, nil
}

// syncScale follows the window to a screen with another backing scale.
func (v *View) syncScale() {
	if d := cg.BackingScale(v.nsview); d != v.dpr {
		v.dpr = d
		v.host.SetScale(d)
	}
}

// unpainted is false: Frame puts the output on screen itself.
func (v *View) unpainted() bool { return false }
