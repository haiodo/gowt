//go:build darwin

// Hand-written: NSPopover around a view that SWT built.
package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	popoverOnce     sync.Once
	popoverShowFn   func(self, sel uintptr, r NSRect, view, edge uintptr)
	popoverSizeFn   func(self, sel uintptr, s NSSize)
	popovers        = map[uintptr]*popoverState{}
	popoverDelegate uintptr
)

// Visual sides of the anchor for ShowPopover.
const (
	PopoverBelow = iota
	PopoverAbove
	PopoverRight
	PopoverLeft
)

const (
	nsMinXEdge = 0
	nsMinYEdge = 1
	nsMaxXEdge = 2
	nsMaxYEdge = 3

	popoverBehaviorTransient = 1 // NSPopoverBehaviorTransient: closes on a click outside and on Esc
)

// popoverState remembers where the content view lives while the popover is closed.
type popoverState struct {
	content *NSView
	window  uintptr
	onClose func()
	focus   *NSView // given the first responder when the popover is on screen
}

func (st *popoverState) applyFocus() {
	if st.focus == nil {
		return
	}
	if w := st.focus.Window(); w != nil {
		msg(uintptr(w.Id), "makeFirstResponder:", uintptr(st.focus.Id))
	}
}

// restore puts the content back as the contentView of its window, so SWT keeps a valid hierarchy
// while the popover is not on screen. It does nothing when the view is already there.
func (st *popoverState) restore() {
	if st.window == 0 || msg(st.window, "contentView") == uintptr(st.content.Id) {
		return
	}
	msg(st.window, "setContentView:", uintptr(st.content.Id))
}

// NewPopover creates an NSPopover that shows content, the content view of a hidden SWT window. The
// view is in the popover only while it is shown. onClose runs after the popover closed, whoever closed it.
func NewPopover(content *NSView, onClose func()) uintptr {
	popoverOnce.Do(func() {
		objcInit()
		purego.RegisterFunc(&popoverShowFn, objcMsgSend)
		purego.RegisterFunc(&popoverSizeFn, objcMsgSend)
		cls := newClass("GowtPopoverDelegate", []string{"NSPopoverDelegate"}, map[string]objcMethod{
			"popoverDidShow:": {purego.NewCallback(func(self, _, note uintptr) {
				if st := popovers[msg(note, "object")]; st != nil {
					st.applyFocus()
				}
			}), "v@:@"},
			"popoverDidClose:": {purego.NewCallback(func(self, _, note uintptr) {
				pop := msg(note, "object")
				st := popovers[pop]
				// A late notification of an earlier close must not take the view from a shown popover.
				if st == nil || msg(pop, "isShown")&0xff != 0 {
					return
				}
				st.restore()
				st.onClose()
			}), "v@:@"},
		})
		popoverDelegate = msg(msg(cls, "alloc"), "init")
	})
	pop := msg(msg(class("NSPopover"), "alloc"), "init")
	vc := msg(msg(class("NSViewController"), "alloc"), "init")
	msg(vc, "setView:", uintptr(content.Id))
	msg(pop, "setContentViewController:", vc)
	msg(vc, "release")
	msg(pop, "setBehavior:", popoverBehaviorTransient)
	msg(pop, "setAnimates:", 1)
	msg(pop, "setDelegate:", popoverDelegate)
	var window uintptr
	if w := content.Window(); w != nil {
		window = uintptr(w.Id)
	}
	popovers[pop] = &popoverState{content: content, window: window, onClose: onClose}
	return pop
}

// ShowPopover opens pop on the given visual side of r, a rectangle in the coordinates SWT uses for
// anchor (origin at the top-left), with a content of size.
func ShowPopover(pop uintptr, anchor *NSView, r NSRect, edge int, size NSSize) {
	v := uintptr(anchor.Id)
	flipped := msg(v, "isFlipped")&0xff != 0
	if !flipped {
		r.Y = anchor.Bounds().Height - r.Y - r.Height
	}
	// AppKit names the sides by the axis of the view: in a flipped view the bottom is the max Y.
	sides := [...][2]uintptr{
		PopoverBelow: {nsMinYEdge, nsMaxYEdge},
		PopoverAbove: {nsMaxYEdge, nsMinYEdge},
		PopoverRight: {nsMaxXEdge, nsMaxXEdge},
		PopoverLeft:  {nsMinXEdge, nsMinXEdge},
	}[edge]
	side := sides[0]
	if flipped {
		side = sides[1]
	}
	popoverSizeFn(pop, sel("setContentSize:"), size)
	popoverShowFn(pop, sel("showRelativeToRect:ofView:preferredEdge:"), r, v, side)
	if w := popovers[pop].content.Window(); w != nil {
		msg(uintptr(w.Id), "makeKeyWindow")
	}
}

// PopoverContentRect returns the content of a shown pop in screen points with the origin at the
// top-left of the primary screen of height primaryHeight.
func PopoverContentRect(pop uintptr, primaryHeight float64) (x, y, w, h float64) {
	v := popovers[pop].content
	win := v.Window()
	if win == nil || msg(pop, "isShown")&0xff == 0 {
		return
	}
	b := v.Bounds()
	r := v.ConvertRect_toView_(b, nil)
	top := win.ConvertBaseToScreen(NSPoint{X: r.X, Y: r.Y + r.Height})
	return top.X, primaryHeight - top.Y, b.Width, b.Height
}

// PopoverShown reports whether pop is on screen.
func PopoverShown(pop uintptr) bool { return msg(pop, "isShown")&0xff != 0 }

// ClosePopover closes pop at once and gives the view back to its window. The caller reports this
// close itself; the delegate reports the ones the user made.
func ClosePopover(pop uintptr) {
	msg(pop, "close")
	if st := popovers[pop]; st != nil {
		st.restore()
	}
}

// ReleasePopover closes pop and forgets it.
func ReleasePopover(pop uintptr) {
	msg(pop, "close")
	if st := popovers[pop]; st != nil {
		st.restore()
	}
	delete(popovers, pop)
	msg(pop, "release")
}

// FocusPopover makes v the first responder of the popover window, now and again when the popover
// has appeared; it needs no active application, unlike the key window.
func FocusPopover(pop uintptr, v *NSView) {
	st := popovers[pop]
	st.focus = v
	if msg(pop, "isShown")&0xff != 0 {
		st.applyFocus()
	}
}

// PerformClosePopover closes pop the way a click outside does: with the animation and the delegate.
func PerformClosePopover(pop uintptr) { msg(pop, "performClose:", 0) }

// FirstResponderIn reports whether the first responder of the window of v is v or inside v, as the
// field editor of a text field is.
func FirstResponderIn(v *NSView) bool {
	w := v.Window()
	if w == nil {
		return false
	}
	for r := msg(uintptr(w.Id), "firstResponder"); r != 0; {
		if r == uintptr(v.Id) {
			return true
		}
		if msg(r, "respondsToSelector:", sel("superview"))&0xff == 0 {
			return false
		}
		r = msg(r, "superview")
	}
	return false
}
