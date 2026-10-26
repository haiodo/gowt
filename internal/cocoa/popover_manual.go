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
	popoverClosed   = map[uintptr]func(){}
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

// NewPopover creates an NSPopover that shows content, a view SWT built, moved out of its hidden
// window. onClose runs after the popover closed, whoever closed it.
func NewPopover(content *NSView, onClose func()) uintptr {
	popoverOnce.Do(func() {
		objcInit()
		purego.RegisterFunc(&popoverShowFn, objcMsgSend)
		purego.RegisterFunc(&popoverSizeFn, objcMsgSend)
		cls := newClass("GowtPopoverDelegate", []string{"NSPopoverDelegate"}, map[string]objcMethod{
			"popoverDidClose:": {purego.NewCallback(func(self, _, note uintptr) {
				if f := popoverClosed[msg(note, "object")]; f != nil {
					f()
				}
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
	popoverClosed[pop] = onClose
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
}

// ClosePopover closes pop; its onClose runs.
func ClosePopover(pop uintptr) { msg(pop, "close") }

// ReleasePopover closes pop and forgets it.
func ReleasePopover(pop uintptr) {
	delete(popoverClosed, pop)
	msg(pop, "close")
	msg(pop, "release")
}
