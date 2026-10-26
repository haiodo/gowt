package gowt

import (
	"github.com/haiodo/gowt/swt"
)

// PopoverEdge is the side of the anchor on which a Popover prefers to open.
type PopoverEdge int

// The zero value is PopoverBelow. A popover that does not fit on its edge opens on the opposite one.
const (
	PopoverBelow PopoverEdge = iota
	PopoverAbove
	PopoverRight
	PopoverLeft
)

// popoverGap is the space between the anchor and a popover without an arrow (Windows, Linux).
const popoverGap = 6

// Popover is a floating panel next to a control or a point: a quick panel, an operation box, a
// "go to" search. Its children are ordinary widgets, added as in a Window. It closes by a click
// outside, Esc and loss of the application focus. A closed popover keeps its content and can be shown again.
//
// macOS shows an NSPopover with the arrow and the system material. Windows and Linux show a
// borderless window with a frame, placed so that it stays on the screen of the anchor.
type Popover struct {
	*panel
	shell    *swt.Shell
	w, h     int // SetSize; 0 means the preferred size
	onClose  []func()
	onEscape func() bool
	shown    bool
	closing  bool
	active   bool // a deactivation before the first activation is left over from an earlier hide
	filters  *popoverFilter
	native   popoverNative
	owner    *swt.Shell
}

// qaPopover tells the QA driver about a popover; set in a gowtqa build.
var qaPopover func(*Popover)

// Popover creates a hidden popover for w; Show it with ShowAt or ShowAtPoint once its content is built.
// Children of a callback that needs the popover go in p.Add after this call.
// Options: Gap, Padding, children; the placement options do not apply. Padding defaults to Spacing.SM.
func (w *Window) Popover(opts ...Option) *Popover {
	s := swt.NewShellParentStyle(w.shell, popoverStyle)
	s.SetLayout(swt.NewFillLayout())
	c := swt.NewCompositeParentStyle(&s.Composite, popoverBodyStyle)
	gap, pad := spacingOf(opts, Spacing.SM)
	c.SetLayout(newBox(false, gap, pad))
	p := &Popover{panel: newPanel(c), shell: s, owner: w.shell}
	p.native.init(p)
	finish(p, 0, opts)
	s.AddShellListener(swt.ShellListenerShellActivatedAdapter(func(*swt.ShellEvent) { p.active = p.shown }))
	s.AddShellListener(swt.ShellListenerShellDeactivatedAdapter(func(*swt.ShellEvent) { p.deactivated() }))
	s.AddDisposeListener(disposeFunc(func() { p.native.dispose(); p.dismissed() }))
	if qaPopover != nil {
		qaPopover(p)
	}
	return p
}

// SetSize fixes the content size in points; without it the popover takes its preferred size at Show.
func (p *Popover) SetSize(width, height int) { p.w, p.h = width, height }

// ShowAt opens the popover next to the control anchor on the given edge. A gfx view is an anchor
// through its Canvas.
func (p *Popover) ShowAt(anchor Widget, edge PopoverEdge) {
	b := anchor.control().GetBounds()
	p.show(anchor.control(), popRect{0, 0, int(b.Width), int(b.Height)}, edge)
}

// ShowAtPoint opens the popover at the point (x, y) of the control anchor, in the control's points.
// Over a gfx view the caller converts a scene point to the canvas point first.
func (p *Popover) ShowAtPoint(anchor Widget, x, y int, edge PopoverEdge) {
	p.show(anchor.control(), popRect{x, y, 0, 0}, edge)
}

// Close hides the popover as the user's click outside would, and runs OnClose. It does nothing when
// the popover is not shown.
func (p *Popover) Close() {
	if !p.shown || p.closing {
		return
	}
	p.closing = true // hiding the window reports its deactivation
	p.native.hide(p)
	p.closing = false
	p.dismissed()
}

// Shown reports whether the popover is on screen.
func (p *Popover) Shown() bool { return p.shown }

// OnClose registers f, called each time the popover closes, however it closed.
func (p *Popover) OnClose(f func()) { p.onClose = append(p.onClose, f) }

// OnEscape registers f, called on Esc in the popover before it closes; returning true keeps the
// popover open (the search clears its text first, say). Enter is not handled: use the OnActivate of
// the text field or the list.
func (p *Popover) OnEscape(f func() bool) { p.onEscape = f }

// Dispose destroys the popover and its content; a shown popover closes first.
func (p *Popover) Dispose() {
	if !p.shell.IsDisposed() {
		p.shell.Dispose()
	}
}

// Unwrap returns the underlying shell. On macOS it is never shown and keeps its window; its body view is moved into the NSPopover while the popover is on screen.
func (p *Popover) Unwrap() *swt.Shell { return p.shell }

func (p *Popover) show(anchor *swt.Control, local popRect, edge PopoverEdge) {
	if p.shown || p.shell.IsDisposed() {
		return
	}
	p.shown = true
	p.filters = newPopoverFilter(p)
	p.native.show(p, anchor, local, edge)
	if c := p.initialFocus(); c != nil {
		p.native.focus(c)
	}
}

// initialFocus returns the control given Focused, or the first control that takes the focus.
func (p *Popover) initialFocus() *swt.Control {
	if c, _ := p.shell.GetDataKey(focusedKey).(*swt.Control); c != nil && !c.IsDisposed() {
		return c
	}
	return nil
}

// dismissed runs when the popover has left the screen, whoever hid it.
func (p *Popover) dismissed() {
	if !p.shown {
		return
	}
	p.shown, p.active = false, false
	p.native.dismissed(p)
	if p.filters != nil {
		p.filters.remove()
		p.filters = nil
	}
	for _, f := range p.onClose {
		f()
	}
}

func (p *Popover) deactivated() {
	if p.shown && p.active && p.native.closeOnDeactivate() {
		p.Close()
	}
}

// popRect is a rectangle in points.
type popRect struct{ X, Y, W, H int }

// popoverPlace returns the top-left corner of a w x h panel next to anchor on edge, flipped to the
// opposite edge when it does not fit there and moved to stay inside screen.
func popoverPlace(anchor, screen popRect, w, h int, edge PopoverEdge) (x, y int) {
	at := func(e PopoverEdge) (int, int) {
		switch e {
		case PopoverAbove:
			return anchor.X + (anchor.W-w)/2, anchor.Y - h - popoverGap
		case PopoverRight:
			return anchor.X + anchor.W + popoverGap, anchor.Y + (anchor.H-h)/2
		case PopoverLeft:
			return anchor.X - w - popoverGap, anchor.Y + (anchor.H-h)/2
		}
		return anchor.X + (anchor.W-w)/2, anchor.Y + anchor.H + popoverGap
	}
	fits := func(e PopoverEdge, x, y int) bool {
		switch e {
		case PopoverAbove:
			return y >= screen.Y
		case PopoverRight:
			return x+w <= screen.X+screen.W
		case PopoverLeft:
			return x >= screen.X
		}
		return y+h <= screen.Y+screen.H
	}
	x, y = at(edge)
	if !fits(edge, x, y) {
		opposite := [...]PopoverEdge{PopoverAbove, PopoverBelow, PopoverLeft, PopoverRight}[edge]
		if ox, oy := at(opposite); fits(opposite, ox, oy) {
			x, y = ox, oy
		}
	}
	x = max(min(x, screen.X+screen.W-w), screen.X)
	y = max(min(y, screen.Y+screen.H-h), screen.Y)
	return x, y
}

// popoverFilter closes the popover on Esc and on a mouse press outside it, which no shell
// deactivation reports for a click in a window that is not activated by it.
type popoverFilter struct {
	p *Popover
	d *swt.Display
}

func newPopoverFilter(p *Popover) *popoverFilter {
	f := &popoverFilter{p, p.shell.GetDisplay()}
	f.d.AddFilter(swt.KeyDown, f)
	f.d.AddFilter(swt.MouseDown, f)
	return f
}

func (f *popoverFilter) remove() {
	f.d.RemoveFilter(swt.KeyDown, f)
	f.d.RemoveFilter(swt.MouseDown, f)
}

func (f *popoverFilter) HandleEvent(e *swt.Event) {
	if e.Widget == nil || !f.p.shown {
		return
	}
	c, ok := e.Widget.Impl().(interface{ GetShell() *swt.Shell })
	inside := ok && c.GetShell() == f.p.shell
	switch {
	case e.Type == swt.KeyDown && inside && e.KeyCode == int32(swt.ESC):
		if f.p.onEscape != nil && f.p.onEscape() {
			return
		}
		e.Doit = false
		f.p.Close()
	case e.Type == swt.MouseDown && !inside:
		f.p.Close()
	}
}
