//go:build darwin

// Hand-written: NSPopover around a view that SWT built.
package cocoa

import (
	"fmt"
	"os"
	"sync"
	"time"

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
	window  uintptr // the Shell's window
	holder  uintptr // stands in for content as the contentView of window while content is in the popover
	onClose func()
	keyWin  uintptr // the popover window observed for becoming key
	focus   *NSView // given the first responder when the popover is on screen
}

func (st *popoverState) applyFocus() {
	if st.focus == nil || st.focus.Window() == nil {
		PopLog("applyFocus: nothing to do (focus view %v, window %v)", st.focus != nil, st.focus != nil && st.focus.Window() != nil)
		return
	}
	if w := st.focus.Window(); w != nil {
		ok := msg(uintptr(w.Id), "makeFirstResponder:", uintptr(st.focus.Id))&0xff != 0
		PopLog("applyFocus: makeFirstResponder=%v responder=%s key=%v", ok, className(msg(uintptr(w.Id), "firstResponder")),
			msg(uintptr(w.Id), "isKeyWindow")&0xff != 0)
	}
}

func className(o uintptr) string {
	if o == 0 {
		return "nil"
	}
	return goString(msg(o, "className"))
}

// observeKey asks for gowtPopoverKey: when the popover window becomes key: in an application that
// activates after the popover appeared, the first responder is set again then.
func (st *popoverState) observeKey(on bool) {
	center := msg(class("NSNotificationCenter"), "defaultCenter")
	name := nsString("NSWindowDidBecomeKeyNotification")
	if on {
		w := st.content.Window()
		if w == nil || st.keyWin == uintptr(w.Id) {
			return
		}
		st.keyWin = uintptr(w.Id)
		msg(center, "addObserver:selector:name:object:", popoverDelegate, sel("gowtPopoverKey:"), name, st.keyWin)
	} else if st.keyWin != 0 {
		msg(center, "removeObserver:name:object:", popoverDelegate, name, st.keyWin)
		st.keyWin = 0
	}
}

// hold puts a placeholder of the size of content in the Shell's window and leaves content to the
// popover. SWT reads the size of the Shell from the contentView of its window and the children of the
// Shell from content, so both keep working while the popover is shown.
func (st *popoverState) hold() {
	if st.window == 0 || msg(st.window, "contentView") != uintptr(st.content.Id) {
		return
	}
	if st.holder == 0 {
		st.holder = msg(msg(class("NSView"), "alloc"), "init")
	}
	msgRectOnly(st.holder, sel("setFrame:"), st.content.Frame())
	msg(st.window, "setContentView:", st.holder)
}

// restore gives content back to the window as its contentView. It does nothing when it is there.
func (st *popoverState) restore() {
	st.observeKey(false)
	PopLog("restore")
	if st.window == 0 || msg(st.window, "contentView") == uintptr(st.content.Id) {
		return
	}
	msg(st.window, "setContentView:", uintptr(st.content.Id))
}

// NewPopover creates an NSPopover that shows content, the contentView of a hidden SWT Shell. The
// view is in the popover only while it is shown. onClose runs after the popover closed, whoever closed it.
func NewPopover(content *NSView, onClose func()) uintptr {
	popoverOnce.Do(func() {
		objcInit()
		purego.RegisterFunc(&popoverShowFn, objcMsgSend)
		purego.RegisterFunc(&popoverSizeFn, objcMsgSend)
		cls := newClass("GowtPopoverDelegate", []string{"NSPopoverDelegate"}, map[string]objcMethod{
			"popoverDidShow:": {purego.NewCallback(func(self, _, note uintptr) {
				PopLog("delegate didShow")
				// The notification is late when the app is inactive: the popover may be closed by now.
				if pop := msg(note, "object"); popovers[pop] != nil && msg(pop, "isShown")&0xff != 0 {
					popovers[pop].applyFocus()
				}
			}), "v@:@"},
			"gowtPopoverKey:": {purego.NewCallback(func(self, _, note uintptr) {
				w := msg(note, "object")
				for pop, st := range popovers {
					if win := st.content.Window(); win != nil && uintptr(win.Id) == w && msg(pop, "isShown")&0xff != 0 {
						PopLog("popover window became key")
						st.applyFocus()
					}
				}
			}), "v@:@"},
			"popoverDidClose:": {purego.NewCallback(func(self, _, note uintptr) {
				pop := msg(note, "object")
				st := popovers[pop]
				PopLog("delegate didClose isShown=%v known=%v", msg(pop, "isShown")&0xff != 0, st != nil)
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
	st := popovers[pop]
	PopLog("ShowPopover: hold, contentView is content: %v", msg(st.window, "contentView") == uintptr(st.content.Id))
	st.hold()
	// A new controller per show: a popover whose controller keeps the same view does not attach the
	// view again after it was taken out, and shows nothing until the animation ends.
	vc := msg(msg(class("NSViewController"), "alloc"), "init")
	msg(vc, "setView:", uintptr(st.content.Id))
	msg(pop, "setContentViewController:", vc)
	msg(vc, "release")
	popoverShowFn(pop, sel("showRelativeToRect:ofView:preferredEdge:"), r, v, side)
	PopLog("ShowPopover: shown=%v", msg(pop, "isShown")&0xff != 0)
	st.observeKey(true)
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
	PopLog("ClosePopover")
	// An animated close keeps the popover "shown" for a while, and a show in that time is dropped.
	msg(pop, "setAnimates:", 0)
	msg(pop, "close")
	msg(pop, "setAnimates:", 1)
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
	if st := popovers[pop]; st != nil && st.holder != 0 {
		msg(st.holder, "release")
	}
	delete(popovers, pop)
	msg(pop, "release")
}

// FocusPopover makes v the first responder of the popover window, now and again when the popover
// has appeared; it needs no active application, unlike the key window.
// It reports whether the view was in the window and got the focus.
func FocusPopover(pop uintptr, v *NSView) bool {
	st := popovers[pop]
	st.focus = v
	if msg(pop, "isShown")&0xff == 0 || v.Window() == nil {
		return false
	}
	st.applyFocus()
	return true
}

// CloseBehindBack closes pop as AppKit does for a click outside: nothing here restores the view or
// reports the close, only the delegate does. performClose: does nothing in an inactive application.
func CloseBehindBack(pop uintptr) {
	PopLog("CloseBehindBack")
	msg(pop, "close")
}

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

var popoverT0 = time.Now()

// PopLog writes a line with the time to stderr (GOWT_POPOVER_DEBUG=1) or appends it to the file
// named by GOWT_POPOVER_DEBUG.
func PopLog(f string, a ...any) {
	v := os.Getenv("GOWT_POPOVER_DEBUG")
	if v == "" {
		return
	}
	line := fmt.Sprintf("%7dms popover: %s\n", time.Since(popoverT0).Milliseconds(), fmt.Sprintf(f, a...))
	if v == "1" {
		os.Stderr.WriteString(line)
	} else if w, err := os.OpenFile(v, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		w.WriteString(line)
		w.Close()
	}
}
