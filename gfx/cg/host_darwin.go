//go:build darwin

package cg

import (
	"image"
	"slices"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/cocoa"
)

var (
	msgRect    func(recv, sel uintptr, r cgRect)
	msgPoint   func(recv, sel uintptr, p cgPoint)
	msgF64     func(recv, sel uintptr, f float64)
	msgF32     func(recv, sel uintptr, f float32)
	msgAffine  func(recv, sel uintptr, m cgAffine)
	msgInt     func(recv, sel, a uintptr) uintptr
	msgRectRet func(recv, sel uintptr) cgRect
	msgAffRet  func(recv, sel uintptr) cgAffine
	msgF32Ret  func(recv, sel uintptr) float32
	msgF64Ret  func(recv, sel uintptr) float64

	hostOnce sync.Once
	sels     struct {
		alloc, init, release, addSublayer, removeFromSuperlayer, setFrame, setBounds, setPosition, setAnchorPoint,
		setAffineTransform, setOpacity, setHidden, setContents, setContentsScale, begin, commit, setDisableActions,
		frame, affineTransform, opacity, contents, isHidden, setActions, setMinFilter, minFilter uintptr
	}
	clsLayer, clsTransaction uintptr
	filterTrilinear          uintptr // kCAFilterTrilinear
)

func initHost() {
	hostOnce.Do(func() {
		bind()
		if _, err := purego.Dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
			panic("gfx/cg: dlopen QuartzCore: " + err.Error())
		}
		for _, f := range []any{&msgRect, &msgPoint, &msgF64, &msgF32, &msgAffine, &msgInt, &msgRectRet, &msgAffRet, &msgF32Ret, &msgF64Ret} {
			purego.RegisterLibFunc(f, purego.RTLD_DEFAULT, "objc_msgSend")
		}
		s := func(n string) uintptr { return uintptr(cocoa.OSSel_registerName(n)) }
		sels.alloc, sels.init, sels.release = s("alloc"), s("init"), s("release")
		sels.addSublayer, sels.removeFromSuperlayer = s("addSublayer:"), s("removeFromSuperlayer")
		sels.setFrame, sels.setBounds, sels.setPosition, sels.setAnchorPoint = s("setFrame:"), s("setBounds:"), s("setPosition:"), s("setAnchorPoint:")
		sels.setAffineTransform, sels.setOpacity, sels.setHidden = s("setAffineTransform:"), s("setOpacity:"), s("setHidden:")
		sels.setContents, sels.setContentsScale = s("setContents:"), s("setContentsScale:")
		sels.begin, sels.commit, sels.setDisableActions = s("begin"), s("commit"), s("setDisableActions:")
		sels.frame, sels.affineTransform, sels.opacity, sels.contents, sels.isHidden = s("frame"), s("affineTransform"), s("opacity"), s("contents"), s("isHidden")
		sels.setActions = s("setActions:")
		sels.setMinFilter, sels.minFilter = s("setMinificationFilter:"), s("minificationFilter")
		a, err := purego.Dlsym(purego.RTLD_DEFAULT, "kCAFilterTrilinear")
		if err != nil {
			panic("gfx/cg: dlsym kCAFilterTrilinear: " + err.Error())
		}
		filterTrilinear = *(*uintptr)(*(*unsafe.Pointer)(unsafe.Pointer(&a)))
		clsLayer = uintptr(cocoa.OSObjc_lookUpClass("CALayer"))
		clsTransaction = uintptr(cocoa.OSObjc_lookUpClass("CATransaction"))
	})
}

func send0(recv, sel uintptr) uintptr { return msgInt(recv, sel, 0) }

// Host is a gfx.Host on a CALayer: the four layer groups are sublayers of root, in this order.
// Use it from the main thread. The root layer's coordinates must be y down, as those of a layer
// behind a flipped NSView are.
type Host struct {
	groups [4]uintptr
	scale  float64
	layers [4][]*layer // in the order the layers were added: the z order within a group
}

// NewHost adds the group layers to root. scale is the backing scale of the screen (contents scale).
func NewHost(root uintptr, scale float64) *Host {
	initHost()
	h := &Host{scale: scale}
	for i := range h.groups {
		l := send0(send0(clsLayer, sels.alloc), sels.init)
		msgPoint(l, sels.setAnchorPoint, cgPoint{})
		msgInt(root, sels.addSublayer, l)
		h.groups[i] = l
	}
	return h
}

// ViewLayer makes the NSView layer-backed and returns its layer, to pass to NewHost.
func ViewLayer(view uintptr) uintptr {
	initHost()
	sel := func(n string) uintptr { return uintptr(cocoa.OSSel_registerName(n)) }
	msgInt(view, sel("setWantsLayer:"), 1)
	return send0(view, sel("layer"))
}

// BackingScale is the device pixels per point of the screen the view's window is on; 1 without a window.
func BackingScale(view uintptr) float64 {
	initHost()
	w := send0(view, uintptr(cocoa.OSSel_registerName("window")))
	if w == 0 {
		return 1
	}
	if s := msgF64Ret(w, uintptr(cocoa.OSSel_registerName("backingScaleFactor"))); s >= 1 {
		return s
	}
	return 1
}

// SetScale changes the contents scale of layers made from now on.
func (h *Host) SetScale(s float64) { h.scale = s }

// Close removes the group layers from the root.
func (h *Host) Close() {
	for _, l := range h.groups {
		send0(l, sels.removeFromSuperlayer)
		send0(l, sels.release)
	}
	h.groups = [4]uintptr{}
}

// Begin starts the layer changes of one frame.
func (h *Host) Begin() {
	send0(clsTransaction, sels.begin)
	msgInt(clsTransaction, sels.setDisableActions, 1)
}

// Commit shows the layer changes of the frame together.
func (h *Host) Commit() { send0(clsTransaction, sels.commit) }

// NewLayer adds a CALayer in group g.
func (h *Host) NewLayer(g gfx.Group) gfx.Layer {
	l := send0(send0(clsLayer, sels.alloc), sels.init)
	msgF64(l, sels.setContentsScale, h.scale)
	if g == gfx.GroupTiles {
		// a tile shown smaller than it was drawn, during a pinch, is mip-mapped: thin lines do not alias away
		msgInt(l, sels.setMinFilter, filterTrilinear)
	}
	msgInt(h.groups[g], sels.addSublayer, l)
	nl := &layer{ref: l, host: h, group: g, op: 1} // a CALayer starts opaque
	h.layers[g] = append(h.layers[g], nl)
	return nl
}

type layer struct {
	host   *Host
	group  gfx.Group
	ref    uintptr
	frame  gfx.Rect
	anchor gfx.Point
	m      gfx.Affine
	op     float64
	hidden bool
	placed bool
	custom bool // anchor or transform set, so the plain frame path is not exact
}

func (l *layer) SetImage(img gfx.Image) {
	if i, ok := img.(*Image); ok {
		msgInt(l.ref, sels.setContents, i.ref)
	}
}

func (l *layer) Set(f gfx.Rect, anchor gfx.Point, m gfx.Affine) {
	if l.placed && f == l.frame && anchor == l.anchor && m == l.m {
		return
	}
	plain := anchor == (gfx.Point{}) && m == gfx.Identity()
	switch {
	case plain && !l.custom:
		msgRect(l.ref, sels.setFrame, cgRect{f.Min.X, f.Min.Y, f.W(), f.H()})
	default:
		w, hh := f.W(), f.H()
		msgRect(l.ref, sels.setBounds, cgRect{0, 0, w, hh})
		if w > 0 && hh > 0 {
			msgPoint(l.ref, sels.setAnchorPoint, cgPoint{anchor.X / w, anchor.Y / hh})
		}
		msgPoint(l.ref, sels.setPosition, cgPoint{f.Min.X + anchor.X, f.Min.Y + anchor.Y})
		msgAffine(l.ref, sels.setAffineTransform, cgAffine{m[0], m[1], m[2], m[3], m[4], m[5]})
		l.custom = !plain
	}
	l.frame, l.anchor, l.m, l.placed = f, anchor, m, true
}

func (l *layer) SetOpacity(a float64) {
	if a != l.op {
		msgF32(l.ref, sels.setOpacity, float32(a))
		l.op = a
	}
}

func (l *layer) SetHidden(h bool) {
	if h != l.hidden {
		var v uintptr
		if h {
			v = 1
		}
		msgInt(l.ref, sels.setHidden, v)
		l.hidden = h
	}
}

func (l *layer) Remove() {
	if l.ref != 0 {
		ls := l.host.layers[l.group]
		if i := slices.Index(ls, l); i >= 0 {
			l.host.layers[l.group] = slices.Delete(ls, i, i+1)
		}
		send0(l.ref, sels.removeFromSuperlayer)
		send0(l.ref, sels.release)
		l.ref = 0
	}
}

// SetHidden hides or shows all three groups at once, without animation: a view capture then has
// the layers or not, whichever way that capture treats them.
func (h *Host) SetHidden(hidden bool) {
	var v uintptr
	if hidden {
		v = 1
	}
	h.Begin()
	for _, g := range h.groups {
		msgInt(g, sels.setHidden, v)
	}
	h.Commit()
}

// Render draws the layers as the window server would show them, from their own images and not from
// the screen, so it works with a sleeping display and needs no screen recording permission. The
// result covers wpt by hpt points of the root layer at scale pixels per point; what lies outside is cut.
func (h *Host) Render(wpt, hpt, scale float64) *image.NRGBA {
	initHost()
	wpx, hpx := int(wpt*scale+0.5), int(hpt*scale+0.5)
	out := image.NewNRGBA(image.Rect(0, 0, wpx, hpx))
	if wpx <= 0 || hpx <= 0 {
		return out
	}
	cs := cgColorSpaceCreateDeviceRGB()
	defer cgColorSpaceRelease(cs)
	ctx := cgBitmapContextCreate(0, uintptr(wpx), uintptr(hpx), 8, uintptr(wpx*4), cs, bitmapInfoBGRA)
	if ctx == 0 {
		return out
	}
	defer cgContextRelease(ctx)
	cgTranslate(ctx, 0, float64(hpx))
	cgScale(ctx, scale, -scale) // y down, in points, as the root layer's
	for _, ls := range h.layers {
		for _, l := range ls {
			l.render(ctx)
		}
	}
	data := cgBitmapContextGetData(ctx)
	px := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&data))), wpx*hpx*4)
	for i := 0; i < wpx*hpx; i++ {
		b, g, r, a := px[i*4], px[i*4+1], px[i*4+2], px[i*4+3]
		if a != 0 && a != 255 {
			b, g, r = byte(int(b)*255/int(a)), byte(int(g)*255/int(a)), byte(int(r)*255/int(a))
		}
		out.Pix[i*4], out.Pix[i*4+1], out.Pix[i*4+2], out.Pix[i*4+3] = r, g, b, a
	}
	return out
}

func (l *layer) render(ctx uintptr) {
	img := msgInt(l.ref, sels.contents, 0)
	op := float64(msgF32Ret(l.ref, sels.opacity))
	if img == 0 || op <= 0 || msgInt(l.ref, sels.isHidden, 0) != 0 || !l.placed {
		return
	}
	w, hh := l.frame.W(), l.frame.H()
	cgSaveGState(ctx)
	cgTranslate(ctx, l.frame.Min.X+l.anchor.X, l.frame.Min.Y+l.anchor.Y)
	cgConcat(ctx, cgAffine{l.m[0], l.m[1], l.m[2], l.m[3], l.m[4], l.m[5]})
	cgTranslate(ctx, -l.anchor.X, -l.anchor.Y)
	cgSetAlpha(ctx, op)
	cgTranslate(ctx, 0, hh) // CGContextDrawImage puts the image upright in a y-up space
	cgScale(ctx, 1, -1)
	cgDrawImage(ctx, cgRect{0, 0, w, hh}, img)
	cgRestoreGSt(ctx)
}
