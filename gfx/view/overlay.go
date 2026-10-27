//go:build darwin || linux || windows

package view

import (
	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// Overlay is a gowt control placed over the view at a scene rectangle. The view moves and sizes it
// on every frame with the camera, so it follows pan and zoom, and scales its font with the zoom.
// Wheel, pinch and smart zoom over it go to the control, not to the view.
type Overlay struct {
	v    *View
	w    g.Widget
	rect gfx.Rect
	font g.Font  // size at zoom 1 in Px; Px 0 leaves the control's own font
	px   float64 // font size applied
	b    [4]int  // bounds applied: x, y, w, h in view points
	gone bool
}

// Overlay creates the widget that opt describes (g.Text(...), g.Button(...)) as a child of the view
// at the scene rectangle rect. font is the font at zoom 1: Px, or Size in points, scaled by the
// zoom; Name, Weight, Bold and Italic are kept. A zero size leaves the control's font alone. It
// returns nil if opt is not a widget. Call it on the UI goroutine.
func (v *View) Overlay(opt g.Option, rect gfx.Rect, font g.Font) *Overlay {
	w := v.cv.AddChild(opt)
	if w == nil {
		return nil
	}
	if font.Px == 0 {
		font.Px = float64(font.Size)
	}
	o := &Overlay{v: v, w: w, rect: rect, font: font, b: [4]int{-1, -1, -1, -1}}
	v.overlays = append(v.overlays, o)
	w.OnDispose(o.drop)
	o.sync()
	return o
}

// Widget is the control, for its own methods: type-assert to *g.TextView and so on.
func (o *Overlay) Widget() g.Widget { return o.w }

// SetRect moves the control to another scene rectangle.
func (o *Overlay) SetRect(r gfx.Rect) {
	o.rect = r
	o.sync()
}

// Remove disposes the control. It is safe after the control or the view is gone.
func (o *Overlay) Remove() {
	if !o.gone {
		o.w.Dispose()
	}
}

func (o *Overlay) drop() {
	o.gone = true
	s := o.v.overlays
	for i, x := range s {
		if x == o {
			o.v.overlays = append(s[:i], s[i+1:]...)
			return
		}
	}
}

// sync puts the control where the camera says; it does nothing when nothing changed.
func (o *Overlay) sync() {
	if o.gone || o.v.gone.Load() {
		return
	}
	cam := o.v.comp.Camera()
	if o.font.Px > 0 {
		if px := zoomedPx(o.font.Px, cam.Zoom); px != o.px {
			f := o.font
			f.Size, f.Px = 0, px
			o.w.SetFont(f)
			o.px = px
		}
	}
	x, y, w, h := childRect(cam, o.rect)
	if b := [4]int{x, y, w, h}; b != o.b {
		o.v.cv.SetChildBounds(o.w, x, y, w, h)
		o.b = b
	}
}

func (v *View) syncOverlays() {
	for _, o := range v.overlays {
		o.sync()
	}
}

// covered reports whether the view point p is over an overlay.
func (v *View) covered(p gfx.Point) bool {
	for _, o := range v.overlays {
		if p.X >= float64(o.b[0]) && p.X < float64(o.b[0]+o.b[2]) && p.Y >= float64(o.b[1]) && p.Y < float64(o.b[1]+o.b[3]) {
			return true
		}
	}
	return false
}

// EditOptions describe Edit.
type EditOptions struct {
	Rect      gfx.Rect // scene rectangle of the field; make it at least one text line high
	Text      string
	Font      g.Font // font at zoom 1, as for Overlay
	Multiline bool
	OnCommit  func(text string)
	OnCancel  func()
	Opts      []g.Option // more options of the g.Text
}

// Editor is an inline text field over the view; see Edit.
type Editor struct {
	o *Overlay
	e *editor
}

// Edit puts a g.Text over the view for in-place editing, selects its text and gives it the focus.
// Esc cancels. Enter commits; in a Multiline field Cmd+Enter or Ctrl+Enter commits and Enter types
// a line break. The focus leaving the field commits: a click on the view or on another control, or
// the window losing the focus. OnCommit or OnCancel runs once, after the field is gone. Starting
// another Edit commits the one that is open.
func (v *View) Edit(opts EditOptions) *Editor {
	v.endEdit()
	all := append([]g.Option{}, opts.Opts...)
	if opts.Multiline {
		all = append(all, g.Multiline())
	}
	o := v.Overlay(g.Text(all...), opts.Rect, opts.Font)
	if o == nil {
		return nil
	}
	tv := o.w.(*g.TextView)
	t := tv.Unwrap()
	display := v.cv.Unwrap().GetDisplay()
	ed := &editor{text: tv.Text, onCommit: opts.OnCommit, onCancel: opts.OnCancel, multiline: opts.Multiline}
	ed.finish = func(refocus bool) {
		if v.editor != nil && v.editor.e == ed {
			v.editor = nil
		}
		o.drop() // out of the list now; the control is disposed after the event that ended the edit
		t.SetVisible(false)
		display.AsyncExec(jrt.NewRunnable(func() {
			if !v.gone.Load() && refocus {
				v.cv.Unwrap().SetFocus()
			}
			tv.Dispose()
		}))
	}
	t.AddKeyListener(swt.KeyListenerKeyPressedAdapter(func(e *swt.KeyEvent) {
		if ed.key(e.KeyCode, modsOfMask(e.StateMask)) {
			e.Doit = false
		}
	}))
	t.AddFocusListener(swt.FocusListenerFocusLostAdapter(func(*swt.FocusEvent) { ed.end(true, false) }))
	tv.SetText(opts.Text)
	focus := func() bool {
		ok := tv.Focus()
		t.SelectAll()
		return ok
	}
	// Focus after the event when not now: on macOS the opening key also reached a field focused in the
	// view's key handler, and SWT refuses the focus during a FocusOut (OnCommit of a focus-loss commit).
	if v.inKey || !focus() {
		display.AsyncExec(jrt.NewRunnable(func() {
			if !ed.done && !t.IsDisposed() {
				focus()
			}
		}))
	}
	h := &Editor{o: o, e: ed}
	v.editor = h
	return h
}

// SetRect moves the field to another scene rectangle.
func (h *Editor) SetRect(r gfx.Rect) { h.o.SetRect(r) }

// Widget is the g.TextView.
func (h *Editor) Widget() *g.TextView { return h.o.w.(*g.TextView) }

// Commit ends the edit as Enter does. It does nothing once the edit has ended.
func (h *Editor) Commit() { h.e.end(true, false) }

// Cancel ends the edit as Esc does. It does nothing once the edit has ended.
func (h *Editor) Cancel() { h.e.end(false, false) }

// endEdit commits the open edit: a press on the view, where GTK keeps the focus in the field.
func (v *View) endEdit() {
	if v.editor != nil {
		v.editor.Commit()
	}
}
