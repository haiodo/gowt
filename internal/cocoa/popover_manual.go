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
	home    *NSView // the superview of content at NewPopover, in the Shell's window
	host    uintptr // the view of the popover's view controller
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

// restore puts the content back into its home view in the Shell's window, so SWT keeps a valid
// hierarchy while the popover is not on screen. The Shell's own contentView never moves, which is why
// SWT calls on the Shell work while the popover is shown. It does nothing when the view is home.
func (st *popoverState) restore() {
	if st.home == nil || msg(uintptr(st.content.Id), "superview") == uintptr(st.home.Id) {
		return
	}
	msg(uintptr(st.home.Id), "addSubview:", uintptr(st.content.Id))
}

// NewPopover creates an NSPopover that shows content, the top view of the one child of a hidden SWT
// Shell. The view is in the popover only while it is shown. onClose runs after the popover closed, whoever closed it.
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
	host := msg(msg(class("NSView"), "alloc"), "init")
	msg(vc, "setView:", host)
	msg(host, "release")
	msg(pop, "setContentViewController:", vc)
	msg(vc, "release")
	msg(pop, "setBehavior:", popoverBehaviorTransient)
	msg(pop, "setAnimates:", 1)
	msg(pop, "setDelegate:", popoverDelegate)
	popovers[pop] = &popoverState{content: content, home: content.Superview(), host: host, onClose: onClose}
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
	st := popovers[pop]
	msg(st.host, "addSubview:", uintptr(st.content.Id))
	popoverShowFn(pop, sel("showRelativeToRect:ofView:preferredEdge:"), r, v, side)
	if w := st.content.Window(); w != nil {
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

// CloseBehindBack closes pop as AppKit does for a click outside: nothing here restores the view or
// reports the close, only the delegate does. performClose: does nothing in an inactive application.
func CloseBehindBack(pop uintptr) { msg(pop, "close") }

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
