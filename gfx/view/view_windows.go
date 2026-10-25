//go:build windows

package view

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/d2d"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// platform is what the Windows View adds: the swap chain host and the DwmFlush frame clock.
type platform struct {
	host   *d2d.Host
	clock  *d2d.FrameClock
	hwnd   uintptr
	posted atomic.Bool
	// polling is set while pollRelease has a timer running; UI thread only
	polling bool
	cursor  map[Cursor]*swt.Cursor
	wheel   wheelState
}

func traceOut(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...) }

var (
	rendererOnce sync.Once
	renderer     *d2d.Renderer
	rendererErr  error
)

// viewRenderer is the process-wide Direct2D device the views draw with. It asks for a GPU and falls
// back to WARP; GFX_WARP=1 forces WARP.
func viewRenderer() (*d2d.Renderer, error) {
	rendererOnce.Do(func() {
		renderer, rendererErr = d2d.NewRenderer(d2d.Options{TextCache: true, ShadowSprite: true, LOD: true, Hardware: os.Getenv("GFX_WARP") == ""})
	})
	return renderer, rendererErr
}

// Software reports whether the views draw on the WARP software rasterizer, which is what a machine
// without a GPU gets.
func Software() bool {
	r, err := viewRenderer()
	return err == nil && r.Software()
}

// hostPaint records how long the swap chain host spent in the last Commit as the view's paint time.
type hostPaint struct {
	*d2d.Host
	v *View
}

func (h hostPaint) Commit() {
	h.Host.Commit()
	h.v.paint = h.Host.LastDraw + h.Host.LastPresent
}

func attach(cv *g.CanvasView, scene *gfx.Scene) (*View, error) {
	rd, err := viewRenderer()
	if err != nil {
		return nil, err
	}
	v := &View{cv: cv, scene: scene}
	v.hwnd = uintptr(cv.Unwrap().Handle)
	v.dpr = windowScale(v.hwnd)
	wpx, hpx, _ := d2d.ClientSize(v.hwnd)
	if v.host, err = d2d.NewHost(rd, v.hwnd, wpx, hpx, v.dpr); err != nil {
		return nil, err
	}
	v.comp = gfx.NewCompositor(scene, rd, hostPaint{v.host, v}, gfx.Options{})
	v.comp.SetCamera(gfx.Camera{Zoom: 1})
	v.comp.SetViewport(float64(wpx)/v.dpr, float64(hpx)/v.dpr, v.dpr)
	v.resize.Store(true)
	v.wireInput()
	display := cv.Unwrap().GetDisplay()
	clock, err := d2d.NewFrameClock(func(fi gfx.FrameInfo) {
		// runs on the clock goroutine: only decide, then hand over to the UI thread
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
	v.clock = clock
	cv.OnDispose(func() {
		v.gone.Store(true)
		for _, stop := range v.stops {
			stop()
		}
		v.clock.Close()
		v.comp.Close()
		v.host.Close()
	})
	return v, clock.Start()
}

// syncScale follows the window to a monitor with another scale and the swap chain to the window size.
func (v *View) syncScale() {
	wpx, hpx, _ := d2d.ClientSize(v.hwnd)
	if d := windowScale(v.hwnd); d != v.dpr {
		v.dpr = d
		v.host.SetScale(d)
	}
	if err := v.host.Resize(wpx, hpx); err != nil {
		panic(err)
	}
	v.host.Invalidate()
}
