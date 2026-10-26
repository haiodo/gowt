//go:build linux

package view

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
	"github.com/haiodo/gowt/swt"
)

// platform is what the Linux View adds: the CPU host, the frame clock tick and the GTK signal ids.
type platform struct {
	host    *cairo.Host
	handle  uintptr // the GTK widget that gets events and the frame clock
	tick    uint32
	signals []signal
	t0      int64 // frame clock origin, µs
	seq     uint64
	cursor  map[Cursor]*swt.Cursor
	scrolls scrollState
	queued  atomic.Bool // a draw is queued and the widget has not painted since
}

func traceOut(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...) }

type painter func(*swt.PaintEvent)

func (f painter) PaintControl(e *swt.PaintEvent) { f(e) }

// attach composites tiles on the CPU with cairo into the canvas and runs frames from the GTK frame
// clock. The tick callback runs on the UI thread, so no hand-over is needed.
func attach(cv *g.CanvasView, scene *gfx.Scene) (*View, error) {
	bindGTK()
	c := cv.Unwrap()
	v := &View{cv: cv, scene: scene}
	v.handle = uintptr(c.EventHandle())
	v.dpr = float64(max(scaleFactor(v.handle), 1))
	r, err := offscreenRenderer()
	if err != nil {
		return nil, err
	}
	// Control.Redraw invalidates the whole GdkWindow the canvas shares with its siblings, children
	// excluded, which blanks the siblings (labels in the same card); queue_draw invalidates the canvas only.
	host, err := cairo.NewHost(func() { v.queued.Store(true); queueDraw(v.handle) })
	if err != nil {
		return nil, err
	}
	v.host = host
	v.comp = gfx.NewCompositor(scene, r, host, gfx.Options{})
	v.comp.SetCamera(gfx.Camera{Zoom: 1})
	w, h := cv.Size()
	v.comp.SetViewport(float64(w), float64(h), v.dpr)
	v.wireInput()
	c.AddPaintListener(painter(func(e *swt.PaintEvent) {
		if cr := e.Gc.GetGCData().Cairo; cr != 0 && !v.gone.Load() {
			v.queued.Store(false)
			t := time.Now()
			host.Paint(uintptr(cr))
			v.paint = time.Since(t)
		}
	}))
	c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { v.resize.Store(true) }))

	tick := func(widget, clock, data uintptr) uintptr {
		if v.gone.Load() {
			return 0
		}
		if !v.wantsFrame() {
			return 1
		}
		now := clockTime(clock)
		if v.t0 == 0 {
			v.t0 = now
		}
		var interval, presentation int64
		clockRefresh(clock, 0, &interval, &presentation)
		v.seq++
		v.frame(gfx.FrameInfo{Time: time.Duration(now-v.t0) * time.Microsecond, Interval: time.Duration(interval) * time.Microsecond, Seq: v.seq})
		return 1
	}
	v.tick = addTick(v.handle, purego.NewCallback(tick), 0, 0)
	cv.OnDispose(func() {
		v.gone.Store(true)
		for _, stop := range v.stops {
			stop()
		}
		for _, s := range v.signals {
			signalDisconnect(s.inst, s.id)
		}
		removeTick(v.handle, v.tick)
		v.comp.Close()
		host.Close()
	})
	return v, nil
}

// syncScale follows the window to a monitor with another scale factor. The CPU host draws
// images by their size in points, so only the compositor needs the new value.
func (v *View) syncScale() { v.dpr = float64(max(scaleFactor(v.handle), 1)) }

// unpainted is true while a frame's output waits for the widget to paint.
func (v *View) unpainted() bool { return v.queued.Load() }
