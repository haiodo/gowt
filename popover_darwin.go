package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// The shell is never shown: its content view lives in the NSPopover while it is on screen and
// returns to the shell's window when it closes.
const (
	popoverStyle     = swt.NO_TRIM | swt.ON_TOP
	popoverBodyStyle = swt.NONE
)

// popoverNative is an NSPopover.
type popoverNative struct{ pop uintptr }

// A transient NSPopover does not close when the owner window is activated from code.
func (n *popoverNative) init(p *Popover) {
	p.owner.AddShellListener(swt.ShellListenerShellActivatedAdapter(func(*swt.ShellEvent) {
		if p.shown {
			p.Close()
		}
	}))
}

func (n *popoverNative) show(p *Popover, anchor *swt.Control, local popRect, edge PopoverEdge) {
	w, h := p.w, p.h
	if w == 0 || h == 0 {
		pt := p.shell.ComputeSize(swt.DEFAULT, swt.DEFAULT)
		if w == 0 {
			w = int(pt.X)
		}
		if h == 0 {
			h = int(pt.Y)
		}
	}
	p.shell.SetSize(int32(w), int32(h))
	p.shell.Layout()
	if n.pop == 0 {
		n.pop = cocoa.NewPopover(p.shell.View, p.dismissed)
	}
	r := cocoa.NSRect{X: float64(local.X), Y: float64(local.Y), Width: float64(local.W), Height: float64(local.H)}
	cocoa.ShowPopover(n.pop, anchor.TopView(), r, int(edge), cocoa.NSSize{Width: float64(w), Height: float64(h)})
}

// focus gives c the first responder of the popover window; SWT's own SetFocus refuses because its
// shell is not visible. It is repeated when the popover has appeared.
func (n *popoverNative) focus(c *swt.Control) { cocoa.FocusPopover(n.pop, c.FocusView()) }

// focused is first-responder based: the key window is the active application's, which a test
// process may not be.
func (n *popoverNative) focused(p *Popover, c *swt.Control) bool {
	return cocoa.FirstResponderIn(c.FocusView())
}

// outsideClick closes the popover the way a click outside does; an inactive test process cannot
// observe the owner window being activated.
func (n *popoverNative) outsideClick(p *Popover) { cocoa.PerformClosePopover(n.pop) }

func (n *popoverNative) onScreen(p *Popover) bool { return n.pop != 0 && cocoa.PopoverShown(n.pop) }

// bounds is the popover content in screen points; the shell's own bounds belong to the hidden window.
func (n *popoverNative) bounds(p *Popover) *swt.Rectangle {
	x, y, w, h := cocoa.PopoverContentRect(n.pop, p.shell.GetDisplay().GetPrimaryFrame().Height)
	return swt.NewRectangle(int32(x), int32(y), int32(w), int32(h))
}

func (n *popoverNative) hide(p *Popover) { cocoa.ClosePopover(n.pop) }

func (n *popoverNative) dispose() {
	if n.pop != 0 {
		cocoa.ReleasePopover(n.pop)
		n.pop = 0
	}
}
