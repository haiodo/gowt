//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

const (
	popoverStyle     = swt.NO_TRIM | swt.ON_TOP
	popoverBodyStyle = swt.BORDER
)

// popoverNative is the borderless window of Windows and Linux.
type popoverNative struct{}

func (n *popoverNative) init(p *Popover) { fluentPopover(p.shell, p.c) }

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
	origin := anchor.ToDisplay(int32(local.X), int32(local.Y))
	at := popRect{int(origin.X), int(origin.Y), local.W, local.H}
	area := anchor.GetMonitor().GetClientArea()
	screen := popRect{int(area.X), int(area.Y), int(area.Width), int(area.Height)}
	x, y := popoverPlace(at, screen, w, h, edge)
	p.shell.SetBounds(int32(x), int32(y), int32(w), int32(h))
	p.shell.Layout()
	p.shell.Open()
}

func (n *popoverNative) hide(p *Popover) { p.shell.SetVisible(false) }

func (n *popoverNative) dispose() {}

func (n *popoverNative) focus(c *swt.Control) { c.SetFocus() }

func (n *popoverNative) focused(p *Popover, c *swt.Control) bool { return p.shell.GetDisplay().GetFocusControl() == c }

func (n *popoverNative) onScreen(p *Popover) bool { return p.shell.GetVisible() }

func (n *popoverNative) bounds(p *Popover) *swt.Rectangle { return p.shell.GetBounds() }
