package view

import (
	"math"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// Phase is where a trackpad gesture or a scroll stands.
type Phase uint8

const (
	PhaseNone     Phase = iota // a mouse wheel, or no momentum
	PhaseMayBegin              // fingers rest on the trackpad: coasting content should stop
	PhaseBegan
	PhaseChanged
	PhaseEnded
	PhaseCancelled
)

// phaseOf maps an NSEventPhase mask: 1 began, 2 stationary, 4 changed, 8 ended, 16 cancelled,
// 32 may begin.
func phaseOf(mask uint64) Phase {
	switch {
	case mask&1 != 0:
		return PhaseBegan
	case mask&(2|4) != 0:
		return PhaseChanged
	case mask&8 != 0:
		return PhaseEnded
	case mask&16 != 0:
		return PhaseCancelled
	case mask&32 != 0:
		return PhaseMayBegin
	}
	return PhaseNone
}

// PointerKind tells what a Pointer event is.
type PointerKind uint8

const (
	PointerDown PointerKind = iota
	PointerMove
	PointerUp
	// PointerCancel ends a drag that lost its release: the view lost focus with a button held.
	PointerCancel
)

// Pointer is a mouse event with the scene point and the node under it.
type Pointer struct {
	Kind PointerKind
	// Pos is in view points, y down. During a drag it may lie outside the view: the view keeps
	// the pointer captured until the last button is released.
	Pos   gfx.Point
	Scene gfx.Point
	// Node is the topmost node under the pointer, nil on empty space.
	Node *gfx.Node
	// Button is SWT's numbering: 1 left, 2 middle, 3 right; on a move the button held, 0 for a hover.
	Button int
	// Count is the click count of a Down (2 for a double click).
	Count int
	Mods  g.Mods
	// Drag is set on moves, releases and cancels that belong to a press.
	Drag bool
	// Target and Origin are the node and the scene point under the press that began the drag.
	Target *gfx.Node
	Origin gfx.Point
}

// Scroll is a scroll gesture or wheel event.
type Scroll struct {
	Pos   gfx.Point
	Scene gfx.Point
	// DX, DY are how far the content should move, as the user's natural scrolling setting says:
	// view points when Precise (trackpad), lines for a wheel.
	DX, DY  float64
	Precise bool
	// Phase is the finger phase of a trackpad scroll; Momentum the phase of the coasting after the
	// fingers left. Both are PhaseNone for a wheel.
	Phase, Momentum Phase
	Mods            g.Mods
}

// Magnify is one step of a pinch.
type Magnify struct {
	Pos   gfx.Point
	Scene gfx.Point
	// Factor multiplies the zoom: 1 for no change.
	Factor float64
	Phase  Phase
	Mods   g.Mods
}

// SmartZoom is the two finger double tap.
type SmartZoom struct {
	Pos   gfx.Point
	Scene gfx.Point
	Node  *gfx.Node
	Mods  g.Mods
}

// Key is a key press or release on the focused view. Code is SWT's key code and Char the character
// typed, 0 for none.
type Key struct {
	Down bool
	Code int
	Char rune
	Mods g.Mods
}

// Cursor is a mouse cursor shape.
type Cursor uint8

const (
	CursorArrow Cursor = iota
	CursorHand
	CursorGrab
	CursorGrabbing
	CursorCrosshair
	CursorText
	CursorResizeH
	CursorResizeV
	CursorNotAllowed
)

// camera is the part of gfx.Compositor the input routing uses.
type camera interface {
	Camera() gfx.Camera
	SetCamera(gfx.Camera)
	SceneAt(gfx.Point) gfx.Point
	ZoomAt(f float64, p gfx.Point)
}

const (
	lineStep      = 16.0  // points a wheel line moves the content
	wheelZoomRate = 0.005 // zoom exponent per point of wheel with Cmd or Ctrl held
)

// router turns native events into the app's callbacks and, where a callback does not take the
// event, into camera navigation.
type router struct {
	cam  camera
	hit  func(gfx.Point) *gfx.Node
	pre  func(*Pointer) // before the callbacks; the view takes focus here
	minZ float64
	maxZ float64

	onPointer func(*Pointer)
	onScroll  func(*Scroll) bool
	onMagnify func(*Magnify) bool
	onSmart   func(*SmartZoom) bool

	last     gfx.Point // view point of the last pointer event
	held     uint8     // bit b set while button b is down
	target   *gfx.Node
	origin   gfx.Point
	coastOff bool    // momentum is dropped until the next finger scroll begins
	prevZoom float64 // zoom before the last smart zoom
}

func newRouter(cam camera, hit func(gfx.Point) *gfx.Node) *router {
	return &router{cam: cam, hit: hit, minZ: 0.02, maxZ: 64}
}

func (r *router) pointer(k PointerKind, pos gfx.Point, button, count int, m g.Mods) {
	p := &Pointer{Kind: k, Pos: pos, Scene: r.cam.SceneAt(pos), Button: button, Count: count, Mods: m}
	p.Node = r.hit(p.Scene)
	r.last = pos
	switch k {
	case PointerDown:
		if r.held == 0 {
			r.target, r.origin = p.Node, p.Scene
		}
		r.held |= 1 << button
		r.coastOff = true
		if r.pre != nil {
			r.pre(p)
		}
	case PointerMove:
		p.Button, p.Drag = lowestButton(r.held), r.held != 0
	case PointerUp:
		p.Drag = r.held != 0
		r.held &^= 1 << button
	case PointerCancel:
		p.Drag = r.held != 0
		r.held = 0
	}
	if p.Drag || k == PointerDown {
		p.Target, p.Origin = r.target, r.origin
	}
	if r.held == 0 {
		defer func() { r.target = nil }()
	}
	if r.onPointer != nil {
		r.onPointer(p)
	}
}

func lowestButton(held uint8) int {
	for b := 1; b < 8; b++ {
		if held&(1<<b) != 0 {
			return b
		}
	}
	return 0
}

func (r *router) scroll(pos gfx.Point, dx, dy float64, precise bool, phase, momentum Phase, m g.Mods) {
	switch {
	case phase == PhaseMayBegin:
		r.coastOff = true
	case phase == PhaseBegan:
		r.coastOff = false
	}
	// a press or a touch stops the coasting, as it does in a native scroll view
	if momentum != PhaseNone && r.coastOff {
		return
	}
	s := &Scroll{Pos: pos, Scene: r.cam.SceneAt(pos), DX: dx, DY: dy, Precise: precise, Phase: phase, Momentum: momentum, Mods: m}
	if r.onScroll != nil && r.onScroll(s) {
		return
	}
	if !precise {
		dx, dy = dx*lineStep, dy*lineStep
	}
	if m.Cmd || m.Ctrl {
		r.zoom(math.Exp(dy*wheelZoomRate), pos)
		return
	}
	r.pan(dx, dy)
}

func (r *router) magnify(pos gfx.Point, delta float64, phase Phase, m g.Mods) {
	f := 1 + delta
	e := &Magnify{Pos: pos, Scene: r.cam.SceneAt(pos), Factor: f, Phase: phase, Mods: m}
	if r.onMagnify != nil && r.onMagnify(e) {
		return
	}
	r.zoom(f, pos)
}

func (r *router) smartZoom(pos gfx.Point, m g.Mods) {
	sp := r.cam.SceneAt(pos)
	e := &SmartZoom{Pos: pos, Scene: sp, Node: r.hit(sp), Mods: m}
	if r.onSmart != nil && r.onSmart(e) {
		return
	}
	z := r.cam.Camera().Zoom
	switch {
	case z != 1:
		r.prevZoom = z
		r.zoom(1/z, pos)
	case r.prevZoom > 0:
		r.zoom(r.prevZoom, pos)
	}
}

// zoom multiplies the zoom by f about the view point p, within the zoom limits.
func (r *router) zoom(f float64, p gfx.Point) {
	z := r.cam.Camera().Zoom
	f = math.Min(math.Max(f, r.minZ/z), r.maxZ/z)
	if f != 1 {
		r.cam.ZoomAt(f, p)
	}
}

// pan moves the content by (dx, dy) view points.
func (r *router) pan(dx, dy float64) {
	c := r.cam.Camera()
	c.Origin.X -= dx / c.Zoom
	c.Origin.Y -= dy / c.Zoom
	r.cam.SetCamera(c)
}
