package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// The shell is never shown. Its view lives in the NSPopover while it is on screen, and a placeholder
// stands in for it as the contentView of the shell's window.
const (
	popoverStyle     = swt.NO_TRIM | swt.ON_TOP
	popoverBodyStyle = swt.NONE
)

// popoverNative is an NSPopover.
type popoverNative struct{ pop uintptr }

func (n *popoverNative) init(p *Popover) {
	popLog = cocoa.PopLog
	p.shell.AddShellListener(swt.ShellListenerShellActivatedAdapter(func(*swt.ShellEvent) { popLog("shell activated") }))
	p.shell.AddListener(swt.Show, popLogListener("shell Show event"))
	p.shell.AddListener(swt.Hide, popLogListener("shell Hide event"))
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
	p.shell.SetHosted(true)
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

// outsideClick closes the NSPopover without the Popover knowing, so that OnClose has to come from
// popoverDidClose:. The click outside itself is AppKit's transient behaviour: an inactive test process
// cannot drive it, and it is checked by hand.
func (n *popoverNative) outsideClick(p *Popover) { cocoa.CloseBehindBack(n.pop) }

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

const outsideClickName = "an AppKit close (popoverDidClose:)"

// The shell is not the active shell while its body is in the NSPopover, and the transient popover
// closes itself, so a deactivation of the shell must not close it.
func (n *popoverNative) closeOnDeactivate() bool { return false }

func (n *popoverNative) dismissed(p *Popover) { p.shell.SetHosted(false) }

type popLogListener string

func (l popLogListener) HandleEvent(*swt.Event) { popLog(string(l)) }
