//go:build windows

package d2d

import (
	"errors"
	"syscall"
	"time"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

// The host draws the layers with Direct2D into a flip-model swap chain on the canvas window, one
// Present per frame. DirectComposition would save the tile copy but needs a surface and a visual
// per tile, which buys nothing on WARP.

const (
	dxgiGetParent         = 6
	dxgiDeviceGetAdapter  = 7
	dxgiFactoryCreateHwnd = 15
	scPresent             = 8
	scGetBuffer           = 9
	scResizeBuffers       = 13

	d2DCCreateBitmapFromDxgi = 62
)

var iidDXGIFactory2 = mkGUID(0x50c83a1c, 0xe072, 0x4c48, 0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0)
var iidDXGISurface = mkGUID(0xcafcb56c, 0x6ac3, 0x4889, 0xbf, 0x47, 0x9e, 0x23, 0xbb, 0xd2, 0x60, 0xec)

type swapChainDesc1 struct {
	width, height uint32
	format        uint32
	stereo        int32
	sampleCount   uint32
	sampleQuality uint32
	usage         uint32
	bufferCount   uint32
	scaling       uint32
	swapEffect    uint32
	alphaMode     uint32
	flags         uint32
}

// Host is a gfx.Host that composites its layers into a window.
type Host struct {
	r      *Renderer
	hwnd   uintptr
	dc     uintptr
	sc     uintptr
	target uintptr // bitmap over the back buffer
	w, h   int     // back buffer size in device pixels
	dpr    float64
	bg     gfx.Color
	groups [4][]*hostLayer
	dirty  bool
	// Stats of the last Commit.
	LastDraw    time.Duration
	LastPresent time.Duration
}

var _ gfx.Host = (*Host)(nil)

// NewHost makes a swap chain of wpx x hpx device pixels on the window hwnd. dpr is device pixels
// per view point.
func NewHost(r *Renderer, hwnd uintptr, wpx, hpx int, dpr float64) (*Host, error) {
	if wpx < 1 || hpx < 1 {
		wpx, hpx = 1, 1
	}
	h := &Host{r: r, hwnd: hwnd, w: wpx, h: hpx, dpr: dpr, bg: gfx.Color{R: 1, G: 1, B: 1, A: 1}, dirty: true}
	var err error
	if h.dc, err = r.NewDeviceContext(); err != nil {
		return nil, err
	}
	vcall(h.dc, d2DCSetDpi, fl(96), fl(96))
	var adapter, fac uintptr
	if hr := vcall(r.dxgi, dxgiDeviceGetAdapter, ref(&adapter)); failed(hr) {
		h.Close()
		return nil, hrErr("GetAdapter", hr)
	}
	hr := vcall(adapter, dxgiGetParent, ref(iidDXGIFactory2), ref(&fac))
	release(adapter)
	if failed(hr) {
		h.Close()
		return nil, hrErr("GetParent(IDXGIFactory2)", hr)
	}
	defer release(fac)
	desc := &swapChainDesc1{width: uint32(wpx), height: uint32(hpx), format: dxgiB8G8R8A8, sampleCount: 1,
		usage: 0x20, bufferCount: 2, scaling: 0, swapEffect: 4} // RENDER_TARGET_OUTPUT, STRETCH, FLIP_DISCARD
	if hr := vcall(fac, dxgiFactoryCreateHwnd, ui(r.d3d), ui(hwnd), ref(desc), ui(0), ui(0), ref(&h.sc)); failed(hr) {
		h.Close()
		return nil, hrErr("CreateSwapChainForHwnd", hr)
	}
	if err := h.bindTarget(); err != nil {
		h.Close()
		return nil, err
	}
	return h, nil
}

func (h *Host) bindTarget() error {
	var surf uintptr
	if hr := vcall(h.sc, scGetBuffer, ui(0), ref(iidDXGISurface), ref(&surf)); failed(hr) {
		return hrErr("GetBuffer", hr)
	}
	defer release(surf)
	// no properties: the bitmap takes format, alpha and the target option from the surface, which
	// D2D rejects as invalid when they are spelled out
	if hr := vcall(h.dc, d2DCCreateBitmapFromDxgi, ui(surf), ui(0), ref(&h.target)); failed(hr) {
		return hrErr("CreateBitmapFromDxgiSurface", hr)
	}
	vcall(h.dc, d2DCSetTarget, ui(h.target))
	return nil
}

// Resize changes the swap chain to wpx x hpx device pixels.
func (h *Host) Resize(wpx, hpx int) error {
	wpx, hpx = max(wpx, 1), max(hpx, 1)
	if wpx == h.w && hpx == h.h {
		return nil
	}
	vcall(h.dc, d2DCSetTarget, ui(0))
	release(h.target)
	h.target = 0
	if hr := vcall(h.sc, scResizeBuffers, ui(0), ui(uintptr(wpx)), ui(uintptr(hpx)), ui(0), ui(0)); failed(hr) {
		return hrErr("ResizeBuffers", hr)
	}
	h.w, h.h = wpx, hpx
	h.dirty = true
	return h.bindTarget()
}

// SetScale sets the device pixels per view point.
func (h *Host) SetScale(dpr float64) {
	if dpr != h.dpr {
		h.dpr = dpr
		h.dirty = true
	}
}

// SetBackground sets the color shown where no layer covers the view.
func (h *Host) SetBackground(c gfx.Color) { h.bg, h.dirty = c, true }

// Invalidate makes the next Commit draw and present again, e.g. after the window was exposed.
func (h *Host) Invalidate() { h.dirty = true }

// Begin and Commit bracket one frame; Commit draws and presents when a layer changed.
func (h *Host) Begin() {}

// Commit draws the layers again and presents, when one changed.
func (h *Host) Commit() {
	if !h.dirty || h.target == 0 {
		return
	}
	h.dirty = false
	t0 := time.Now()
	dc := h.dc
	vcall(dc, d2DCBeginDraw)
	vcall(dc, d2DCSetTransform, ref(&matrix{1, 0, 0, 1, 0, 0}))
	vcall(dc, d2DCClear, ref(&colorF{float32(h.bg.R), float32(h.bg.G), float32(h.bg.B), float32(h.bg.A)}))
	view := rectF{0, 0, float32(float64(h.w) / h.dpr), float32(float64(h.h) / h.dpr)}
	for g := range h.groups {
		for _, l := range h.groups[g] {
			l.draw(h, &view)
		}
	}
	if hr := vcall(dc, d2DCEndDraw, ui(0), ui(0)); failed(hr) {
		panic(hrErr("EndDraw (host)", hr))
	}
	t1 := time.Now()
	hr := vcall(h.sc, scPresent, ui(1), ui(0))
	h.LastDraw, h.LastPresent = t1.Sub(t0), time.Since(t1)
	if failed(hr) && uint32(hr) != 0x087A0001 { // DXGI_STATUS_OCCLUDED is fine
		panic(hrErr("Present", hr))
	}
}

// Close releases the swap chain; layers still on the host release their images.
func (h *Host) Close() {
	for g := range h.groups {
		for _, l := range h.groups[g] {
			l.img.Release()
		}
		h.groups[g] = nil
	}
	if h.dc != 0 {
		vcall(h.dc, d2DCSetTarget, ui(0))
	}
	release(h.target)
	release(h.sc)
	release(h.dc)
	h.target, h.sc, h.dc = 0, 0, 0
}

type hostLayer struct {
	h       *Host
	g       gfx.Group
	img     *Image
	frame   gfx.Rect
	anchor  gfx.Point
	m       gfx.Affine
	opacity float64
	hidden  bool
}

// NewLayer adds a layer in group g.
func (h *Host) NewLayer(g gfx.Group) gfx.Layer {
	l := &hostLayer{h: h, g: g, m: gfx.Identity(), opacity: 1}
	h.groups[g] = append(h.groups[g], l)
	return l
}

func (l *hostLayer) SetImage(img gfx.Image) {
	src, ok := img.(*Image)
	if !ok || src.bmp == 0 {
		return
	}
	vcall(src.bmp, 1) // AddRef: the caller releases img after this call
	l.img.Release()
	l.img = &Image{src.bmp, src.w, src.h}
	l.h.dirty = true
}

func (l *hostLayer) Set(frame gfx.Rect, anchor gfx.Point, m gfx.Affine) {
	if frame != l.frame || anchor != l.anchor || m != l.m {
		l.frame, l.anchor, l.m = frame, anchor, m
		l.h.dirty = true
	}
}

func (l *hostLayer) SetOpacity(a float64) {
	if a != l.opacity {
		l.opacity = a
		l.h.dirty = true
	}
}

func (l *hostLayer) SetHidden(hide bool) {
	if hide != l.hidden {
		l.hidden = hide
		l.h.dirty = true
	}
}

func (l *hostLayer) Remove() {
	gs := l.h.groups[l.g]
	for i, o := range gs {
		if o == l {
			l.h.groups[l.g] = append(gs[:i:i], gs[i+1:]...)
			break
		}
	}
	l.img.Release()
	l.img = nil
	l.h.dirty = true
}

func (l *hostLayer) draw(h *Host, view *rectF) {
	if l.hidden || l.img == nil || l.img.bmp == 0 || l.opacity <= 0 {
		return
	}
	f := l.frame
	ap := f.Min.Add(l.anchor)
	m := gfx.ScaleBy(h.dpr, h.dpr).Mul(gfx.TranslateBy(ap.X, ap.Y)).Mul(l.m).Mul(gfx.TranslateBy(-ap.X, -ap.Y))
	if bb := m.Bounds(gfx.Rect{Min: f.Min, Max: f.Max}); bb.Max.X < 0 || bb.Max.Y < 0 || bb.Min.X > float64(h.w) || bb.Min.Y > float64(h.h) {
		return
	}
	mm := toMatrix(m)
	vcall(h.dc, d2DCSetTransform, ref(&mm))
	vcall(h.dc, d2DCDrawBitmap, ui(l.img.bmp), ref(&rectF{float32(f.Min.X), float32(f.Min.Y), float32(f.Max.X), float32(f.Max.Y)}),
		fl(l.opacity), ui(1), ui(0))
}

// FrameClock calls tick once per display refresh on its own goroutine. DwmFlush, not a waitable swap
// chain: that one is signaled whenever nothing is queued, so an idle app would spin.
type FrameClock struct {
	tick func(gfx.FrameInfo)
	stop chan struct{}
	done chan struct{}
}

var (
	dwmapi       = syscall.NewLazyDLL("dwmapi.dll")
	procDwmFlush = dwmapi.NewProc("DwmFlush")
)

// NewFrameClock returns a stopped clock.
func NewFrameClock(tick func(gfx.FrameInfo)) (*FrameClock, error) {
	if err := dwmapi.Load(); err != nil {
		return nil, err
	}
	return &FrameClock{tick: tick, stop: make(chan struct{}), done: make(chan struct{})}, nil
}

// Start begins the frame loop.
func (c *FrameClock) Start() error {
	if hr, _, _ := procDwmFlush.Call(); failed(hr) {
		return hrErr("DwmFlush", hr)
	}
	go c.run()
	return nil
}

func (c *FrameClock) run() {
	defer close(c.done)
	t0 := time.Now()
	last := t0
	interval := 16667 * time.Microsecond
	var seq uint64
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		hr, _, _ := procDwmFlush.Call()
		now := time.Now()
		if failed(hr) { // composition is off: fall back to a timer
			time.Sleep(interval)
			now = time.Now()
		}
		if d := now.Sub(last); d > 2*time.Millisecond && d < 100*time.Millisecond {
			interval += (d - interval) / 8
		}
		last = now
		seq++
		c.tick(gfx.FrameInfo{Time: now.Sub(t0) + interval, Interval: interval, Seq: seq})
	}
}

// Close stops the clock and waits for the goroutine to end.
func (c *FrameClock) Close() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	<-c.done
}

var errNoWindow = errors.New("gfx/d2d: no window")

// ClientSize returns the client area of hwnd in device pixels.
func ClientSize(hwnd uintptr) (w, h int, err error) {
	var rc [4]int32
	r, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	if r == 0 {
		return 0, 0, errNoWindow
	}
	return int(rc[2] - rc[0]), int(rc[3] - rc[1]), nil
}

var procGetClientRect = syscall.NewLazyDLL("user32.dll").NewProc("GetClientRect")
