package view

import (
	"math"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

// childRect maps a scene rectangle to the integer rectangle of view points that covers it under cam.
func childRect(cam gfx.Camera, r gfx.Rect) (x, y, w, h int) {
	x0 := math.Floor((r.Min.X - cam.Origin.X) * cam.Zoom)
	y0 := math.Floor((r.Min.Y - cam.Origin.Y) * cam.Zoom)
	x1 := math.Ceil((r.Max.X - cam.Origin.X) * cam.Zoom)
	y1 := math.Ceil((r.Max.Y - cam.Origin.Y) * cam.Zoom)
	return int(x0), int(y0), int(x1 - x0), int(y1 - y0)
}

// fontStep is the grain of the font size that follows the zoom: native fonts are cached per size,
// and a pinch would otherwise make one for every frame.
const fontStep = 0.25

// zoomedPx is the font size px at zoom 1 scaled to zoom, on the fontStep grid and at least 1.
func zoomedPx(px, zoom float64) float64 {
	return math.Max(1, math.Round(px*zoom/fontStep)*fontStep)
}

// editor is the commit and cancel logic of Edit; the SWT control is behind text and finish.
type editor struct {
	text      func() string
	finish    func(refocus bool) // takes the control away
	onCommit  func(string)
	onCancel  func()
	multiline bool
	done      bool
}

// end finishes the session once: later calls, such as the focus loss that disposing the control
// causes, do nothing. refocus asks to give the keyboard back to the canvas, for a key-driven end.
func (e *editor) end(commit, refocus bool) {
	if e.done {
		return
	}
	e.done = true
	s := e.text()
	e.finish(refocus)
	switch {
	case commit && e.onCommit != nil:
		e.onCommit(s)
	case !commit && e.onCancel != nil:
		e.onCancel()
	}
}

// key reports whether it took the key press: Esc cancels, Enter commits a single-line field and
// Cmd or Ctrl with Enter commits a multi-line one (a bare Enter there types a line break).
func (e *editor) key(code int32, m g.Mods) bool {
	switch {
	case code == int32(swt.ESC):
		e.end(false, true)
	case (code == int32(swt.CR) || code == swt.KEYPAD_CR) && (!e.multiline || m.Cmd || m.Ctrl):
		e.end(true, true)
	default:
		return false
	}
	return true
}
