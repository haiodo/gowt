package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// The shell is never shown: its content view lives in the NSPopover.
const (
	popoverStyle     = swt.NO_TRIM | swt.ON_TOP
	popoverBodyStyle = swt.NONE
)

// popoverNative is an NSPopover.
type popoverNative struct{ pop uintptr }

func (n *popoverNative) init(p *Popover) {}

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

func (n *popoverNative) hide(p *Popover) { cocoa.ClosePopover(n.pop) }

func (n *popoverNative) dispose() {
	if n.pop != 0 {
		cocoa.ReleasePopover(n.pop)
		n.pop = 0
	}
}
