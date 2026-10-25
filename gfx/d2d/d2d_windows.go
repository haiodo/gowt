//go:build windows

package d2d

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

// Direct2D vtable slots (IUnknown takes 0..2).
const (
	d2FactoryCreatePathGeometry = 10
	d2FactoryCreateStrokeStyle  = 11
	d2FactoryCreateDevice       = 17 // ID2D1Factory1

	d2DeviceCreateDeviceContext = 4

	// ID2D1RenderTarget
	d2DCCreateSolidBrush    = 8
	d2DCCreateGradStops     = 9
	d2DCCreateLinearBrush   = 10
	d2DCCreateRadialBrush   = 11
	d2DCDrawGeometry        = 22
	d2DCFillGeometry        = 23
	d2DCFillRect            = 17
	d2DCDrawBitmap          = 26
	d2DCDrawTextLayout      = 28
	d2DCSetTransform        = 30
	d2DCSetAntialias        = 32
	d2DCSetTextAntialias    = 34
	d2DCPopLayer            = 41
	d2DCFlush               = 42
	d2DCPushAxisAlignedClip = 45
	d2DCPopAxisAlignedClip  = 46
	d2DCClear               = 47
	d2DCBeginDraw           = 48
	d2DCEndDraw             = 49
	d2DCSetDpi              = 51
	// ID2D1DeviceContext
	d2DCCreateBitmap1 = 57
	d2DCSetTarget     = 74
	d2DCPushLayer1    = 86

	d2BrushSetOpacity   = 4
	d2SolidBrushSetColor = 8

	d2BitmapCopyFrom = 8
	d2BitmapMap      = 14
	d2BitmapUnmap    = 15

	d2GeomOpen        = 17
	d2SinkSetFillMode = 3
	d2SinkBeginFigure = 5
	d2SinkEndFigure   = 8
	d2SinkClose       = 9
	d2SinkAddLine     = 10
	d2SinkAddBezier   = 11
	d2SinkAddQuad     = 12
)

type colorF struct{ r, g, b, a float32 }
type rectF struct{ l, t, r, b float32 }
type point2F struct{ x, y float32 }

type bitmapProps1 struct {
	format, alpha uint32
	dpiX, dpiY    float32
	options       uint32
	_             uint32
	colorContext  uintptr
}

type mappedRect struct {
	pitch uint32
	_     uint32
	bits  uintptr
}

const (
	dxgiB8G8R8A8   = 87
	alphaPremul    = 1
	bmpTarget      = 1
	bmpCPURead     = 2
	bmpCannotDraw  = 4
	d3dDriverHW    = 1
	d3dDriverWarp  = 5
	d3dBGRASupport = 0x20
)

// Options select the variants the benchmarks compare.
type Options struct {
	TextCache    bool // keep text layouts per (font, string)
	ShadowSprite bool // draw box shadows from a cached 9-slice sprite
	LOD          bool // skip text, shadow and dashes that are unreadable at the current scale
	// FontDir is registered with RegisterFonts before the first text is laid out; the environment
	// variable GFX_FONT_DIR stands in when it is empty.
	FontDir string
	// Hardware asks for a GPU device and falls back to WARP; the default is WARP, whose output does
	// not depend on the GPU and driver.
	Hardware bool
}

// Renderer is a Direct2D 1.1 device with DirectWrite; it is safe for concurrent use and its Targets
// are not shared.
type Renderer struct {
	opts    Options
	factory uintptr // ID2D1Factory1, multithreaded
	d3d     uintptr
	d3dCtx  uintptr
	dxgi    uintptr
	dev     uintptr // ID2D1Device
	warp    bool
	text    *textEngine

	sprites sync.Map // spriteKey -> *shadowSprite
	wholes  sync.Map // wholeKey -> *Image
	images  sync.Map // *gfx.Raster -> *Image
	strokes sync.Map // strokeKey -> uintptr
}

var _ gfx.Renderer = (*Renderer)(nil)

// NewRenderer creates the device. Output is the same whatever the machine, within one machine: all
// drawing goes through the WARP software rasterizer unless o.Hardware is set.
func NewRenderer(o Options) (*Renderer, error) {
	if err := d2d1.Load(); err != nil {
		return nil, fmt.Errorf("gfx/d2d: %w", err)
	}
	r := &Renderer{opts: o}
	var opt struct{ typ uint32 } // D2D1_FACTORY_OPTIONS: debug level none
	hr, _, _ := procD2D1F.Call(1, uintptr(unsafe.Pointer(iidD2DFactory1)), uintptr(unsafe.Pointer(&opt)), uintptr(unsafe.Pointer(&r.factory)))
	if failed(hr) {
		return nil, hrErr("D2D1CreateFactory", hr)
	}
	drivers := []uintptr{d3dDriverWarp}
	if o.Hardware {
		drivers = []uintptr{d3dDriverHW, d3dDriverWarp}
	}
	for _, d := range drivers {
		var level uint32
		hr, _, _ = procD3D.Call(0, d, 0, d3dBGRASupport, 0, 0, 7,
			uintptr(unsafe.Pointer(&r.d3d)), uintptr(unsafe.Pointer(&level)), uintptr(unsafe.Pointer(&r.d3dCtx)))
		if !failed(hr) {
			r.warp = d == d3dDriverWarp
			break
		}
		r.d3d, r.d3dCtx = 0, 0
	}
	if r.d3d == 0 {
		r.Close()
		return nil, hrErr("D3D11CreateDevice", hr)
	}
	if mt, err := queryInterface(r.d3dCtx, iidD3D11Multi); err == nil {
		vcall(mt, 5, ui(1)) // SetMultithreadProtected
		release(mt)
	}
	var err error
	if r.dxgi, err = queryInterface(r.d3d, iidDXGIDevice); err != nil {
		r.Close()
		return nil, err
	}
	if hr := vcall(r.factory, d2FactoryCreateDevice, ui(r.dxgi), ref(&r.dev)); failed(hr) {
		r.Close()
		return nil, hrErr("CreateDevice", hr)
	}
	if r.text, err = newTextEngine(); err != nil {
		r.Close()
		return nil, err
	}
	if o.FontDir == "" {
		o.FontDir = os.Getenv("GFX_FONT_DIR")
	}
	if o.FontDir != "" {
		_, _ = r.RegisterFonts(o.FontDir) // a missing directory leaves the system fonts
	}
	if _, ok := gfx.DefaultMeasurer.(gfx.EstimateText); ok {
		gfx.DefaultMeasurer = r // recordings from now on get DirectWrite bounds
	}
	return r, nil
}

// Close frees the device. Targets and Images made by r must be closed first.
func (r *Renderer) Close() {
	if r.text != nil {
		r.text.close()
	}
	r.images.Range(func(_, v any) bool { v.(*Image).Release(); return true })
	r.wholes.Range(func(_, v any) bool { v.(*Image).Release(); return true })
	r.sprites.Range(func(_, v any) bool { v.(*shadowSprite).release(); return true })
	r.strokes.Range(func(_, v any) bool { release(v.(uintptr)); return true })
	release(r.dev)
	release(r.dxgi)
	release(r.d3dCtx)
	release(r.d3d)
	release(r.factory)
	r.text, r.dev, r.dxgi, r.d3dCtx, r.d3d, r.factory = nil, 0, 0, 0, 0, 0
}

// Software reports whether the device is WARP.
func (r *Renderer) Software() bool { return r.warp }

// Factory, Device, D3DDevice and DXGIDevice are the COM objects behind the renderer, for a Host that
// shares them; the renderer keeps the reference.
func (r *Renderer) Factory() uintptr    { return r.factory }
func (r *Renderer) Device() uintptr     { return r.dev }
func (r *Renderer) D3DDevice() uintptr  { return r.d3d }
func (r *Renderer) DXGIDevice() uintptr { return r.dxgi }

// NewDeviceContext creates a device context on the renderer's device; the caller releases it.
func (r *Renderer) NewDeviceContext() (uintptr, error) {
	var dc uintptr
	if hr := vcall(r.dev, d2DeviceCreateDeviceContext, ui(1), ref(&dc)); failed(hr) { // ENABLE_MULTITHREADED_OPTIMIZATIONS
		return 0, hrErr("CreateDeviceContext", hr)
	}
	return dc, nil
}

// Image is a bitmap on the device: a snapshot of a Target or a Raster converted once. Size is in points.
type Image struct {
	bmp  uintptr
	w, h float64
}

func (i *Image) Size() (float64, float64) { return i.w, i.h }

// Bitmap is the ID2D1Bitmap1, for a Host to draw.
func (i *Image) Bitmap() uintptr { return i.bmp }

func (i *Image) Release() {
	if i.bmp != 0 {
		release(i.bmp)
		i.bmp = 0
	}
}

func sizeU(w, h int) arg { return ui(uintptr(uint32(w)) | uintptr(uint32(h))<<32) }

func newBitmap(dc uintptr, w, h int, data []byte, pitch int, opts uint32) (uintptr, error) {
	props := &bitmapProps1{format: dxgiB8G8R8A8, alpha: alphaPremul, dpiX: 96, dpiY: 96, options: opts}
	var bmp uintptr
	hr := vcall(dc, d2DCCreateBitmap1, sizeU(w, h), buf(data), ui(uintptr(pitch)), ref(props), ref(&bmp))
	if failed(hr) {
		return 0, hrErr("CreateBitmap", hr)
	}
	return bmp, nil
}

// Target is an offscreen BGRA bitmap on the device.
type Target struct {
	r       *Renderer
	dc      uintptr
	bmp     uintptr
	staging uintptr
	brush   uintptr
	w, h    int
	scale   float64
	base    gfx.Affine
	pix     []byte
	pixOK   bool
	rp      replayer
}

// NewTarget makes an offscreen bitmap of wpx x hpx device pixels.
func (r *Renderer) NewTarget(wpx, hpx int) gfx.Target {
	t, err := r.newTarget(wpx, hpx)
	if err != nil {
		panic(err)
	}
	return t
}

func (r *Renderer) newTarget(wpx, hpx int) (*Target, error) {
	if wpx <= 0 || hpx <= 0 {
		return nil, errors.New("gfx/d2d: empty target")
	}
	dc, err := r.NewDeviceContext()
	if err != nil {
		return nil, err
	}
	t := &Target{r: r, dc: dc, w: wpx, h: hpx, scale: 1, base: gfx.Identity()}
	if t.bmp, err = newBitmap(dc, wpx, hpx, nil, 0, bmpTarget); err != nil {
		t.Close()
		return nil, err
	}
	vcall(dc, d2DCSetTarget, ui(t.bmp))
	vcall(dc, d2DCSetDpi, fl(96), fl(96))
	vcall(dc, d2DCSetTextAntialias, ui(2)) // GRAYSCALE: LCD text cannot be cached in a transparent tile
	col := &colorF{0, 0, 0, 1}
	if hr := vcall(dc, d2DCCreateSolidBrush, ref(col), ui(0), ref(&t.brush)); failed(hr) {
		t.Close()
		return nil, hrErr("CreateSolidColorBrush", hr)
	}
	t.pix = make([]byte, wpx*hpx*4)
	t.rp.init(t)
	return t, nil
}

func (t *Target) Close() {
	release(t.staging)
	release(t.brush)
	release(t.bmp)
	if t.dc != 0 {
		vcall(t.dc, d2DCSetTarget, ui(0))
		release(t.dc)
	}
	t.staging, t.brush, t.bmp, t.dc = 0, 0, 0, 0
}

func (t *Target) Size() (int, int) { return t.w, t.h }

// Begin clears the target to bg and maps the scene point origin to the top left pixel at scale.
func (t *Target) Begin(scale float64, origin gfx.Point, bg gfx.Color) {
	t.scale = scale
	t.base = gfx.Affine{scale, 0, 0, scale, -origin.X * scale, -origin.Y * scale}
	vcall(t.dc, d2DCBeginDraw)
	vcall(t.dc, d2DCSetTransform, ref(&matrix{1, 0, 0, 1, 0, 0}))
	vcall(t.dc, d2DCClear, ref(&colorF{}))
	if bg.A > 0 {
		t.rp.setColor(bg)
		vcall(t.dc, d2DCFillRect, ref(&rectF{0, 0, float32(t.w), float32(t.h)}), ui(t.brush))
	}
	t.endDraw()
	t.pixOK = false
}

func (t *Target) endDraw() {
	if hr := vcall(t.dc, d2DCEndDraw, ui(0), ui(0)); failed(hr) {
		panic(hrErr("EndDraw", hr))
	}
}

// Draw paints the items in order.
func (t *Target) Draw(items []gfx.Item) {
	vcall(t.dc, d2DCBeginDraw)
	t.rp.draw(items)
	t.endDraw()
	t.pixOK = false
}

// Pixels is premultiplied BGRA, rows top to bottom; valid until the next Begin or Draw.
func (t *Target) Pixels() []byte {
	if t.pixOK {
		return t.pix
	}
	if t.staging == 0 {
		var err error
		if t.staging, err = newBitmap(t.dc, t.w, t.h, nil, 0, bmpCPURead|bmpCannotDraw); err != nil {
			panic(err)
		}
	}
	if hr := vcall(t.staging, d2BitmapCopyFrom, ui(0), ui(t.bmp), ui(0)); failed(hr) {
		panic(hrErr("CopyFromBitmap", hr))
	}
	var m mappedRect
	if hr := vcall(t.staging, d2BitmapMap, ui(1), ref(&m)); failed(hr) { // D2D1_MAP_OPTIONS_READ
		panic(hrErr("Map", hr))
	}
	for y := 0; y < t.h; y++ {
		row := unsafe.Slice((*byte)(ptr(m.bits+uintptr(y)*uintptr(m.pitch))), t.w*4)
		copy(t.pix[y*t.w*4:], row)
	}
	vcall(t.staging, d2BitmapUnmap)
	t.pixOK = true
	return t.pix
}

// Snapshot copies the bitmap into an Image of the given size in points, for a Host.
func (t *Target) Snapshot(wpt, hpt float64) gfx.Image {
	bmp, err := newBitmap(t.dc, t.w, t.h, nil, 0, 0)
	if err != nil {
		panic(err)
	}
	if hr := vcall(bmp, d2BitmapCopyFrom, ui(0), ui(t.bmp), ui(0)); failed(hr) {
		release(bmp)
		panic(hrErr("CopyFromBitmap", hr))
	}
	return &Image{bmp, wpt, hpt}
}

// Bitmap is the ID2D1Bitmap1 the target draws into.
func (t *Target) Bitmap() uintptr { return t.bmp }
