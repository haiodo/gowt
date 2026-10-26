//go:build darwin || linux || windows

package view

import (
	"fmt"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/swt"
)

// qaHook is what the QA driver sees of a View: its frames, which a capture waits for, and an entry
// for scroll, pinch and pointer events that goes through the router like the native events.
type qaHook struct{ v *View }

func (v *View) registerQA() {
	h := qaHook{v}
	v.cv.Unwrap().SetDataKeyValue(qaview.InputKey, qaview.Input(h))
	remove := qaview.Add(h)
	v.cv.OnDispose(remove)
}

// Request asks for a frame. It is done when a frame ran after the request, no tile is queued or
// drawn, nothing waits for another frame and, where the widget paints the tiles, it painted.
func (h qaHook) Request(shell *swt.Shell) func() bool {
	v := h.v
	c := v.cv.Unwrap()
	if v.gone.Load() || c.IsDisposed() || c.GetShell() != shell || !c.IsVisible() {
		return nil
	}
	base := v.frames.Load()
	v.force.Store(true)
	return func() bool {
		return v.gone.Load() || (v.frames.Load() > base && !v.comp.NeedsFrame() && v.comp.Stats().TilesPending == 0 && !v.unpainted())
	}
}

func (h qaHook) Scroll(x, y, dx, dy float64, precise bool, phase, momentum string, m qaview.Mods) error {
	ph, err := parsePhase(phase)
	if err != nil {
		return err
	}
	mo, err := parsePhase(momentum)
	if err != nil {
		return err
	}
	r := h.v.r
	r.scroll(gfx.Point{X: x, Y: y}, dx, dy, precise, ph, mo, modsOfQA(m), 0)
	r.flushScroll()
	return nil
}

func (h qaHook) Magnify(x, y, magnification float64, phase string, m qaview.Mods) error {
	ph, err := parsePhase(phase)
	if err != nil {
		return err
	}
	h.v.r.magnify(gfx.Point{X: x, Y: y}, magnification, ph, modsOfQA(m))
	return nil
}

// Pointer feeds the router the press, move or release a native event would. A press over an overlay
// control is refused: the control, not the view, would get it.
func (h qaHook) Pointer(kind string, x, y float64, button, count int, m qaview.Mods) error {
	var k PointerKind
	switch kind {
	case "down":
		k = PointerDown
	case "move":
		k = PointerMove
	case "up":
		k = PointerUp
	default:
		return fmt.Errorf("pointer kind %q: want down, move or up", kind)
	}
	p := gfx.Point{X: x, Y: y}
	if k == PointerDown && h.v.covered(p) {
		return fmt.Errorf("the point (%g, %g) is over a control placed on the view, which takes the press", x, y)
	}
	h.v.r.pointer(k, p, button, count, modsOfQA(m))
	return nil
}

func modsOfQA(m qaview.Mods) g.Mods {
	return g.Mods{Shift: m.Shift, Ctrl: m.Ctrl, Alt: m.Alt, Cmd: m.Cmd}
}

func parsePhase(s string) (Phase, error) {
	switch s {
	case "":
		return PhaseNone, nil
	case "mayBegin":
		return PhaseMayBegin, nil
	case "began":
		return PhaseBegan, nil
	case "changed":
		return PhaseChanged, nil
	case "ended":
		return PhaseEnded, nil
	case "cancelled":
		return PhaseCancelled, nil
	}
	return 0, fmt.Errorf("phase %q: want mayBegin, began, changed, ended or cancelled", s)
}
