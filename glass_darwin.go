package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func classicLook() { cocoa.RequireCompatibleLook() }

func (w *Window) setFullSizeContent(on bool) {
	cocoa.SetFullSizeContent(w.shell.View, on)
	w.shell.Layout()
}

// clear makes the panel paint nothing of its own and lets its children inherit that, the SWT
// way (alpha-0 background plus INHERIT_FORCE), so a backdrop behind the panel stays visible.
func (p *panel) clear(on bool) {
	if !on {
		p.c.SetBackgroundWithColor(nil)
		p.c.SetBackgroundMode(swt.NONE)
		return
	}
	c := swt.NewColorDeviceRedGreenBlueAlpha(p.c.GetDisplay(), 0, 0, 0, 0)
	p.c.SetBackgroundWithColor(c)
	c.Dispose()
	p.c.SetBackgroundMode(swt.INHERIT_FORCE)
}

func (p *panel) setGlass(on bool) {
	k := cocoa.BackdropNone
	if on {
		k = cocoa.BackdropGlass
	}
	had := cocoa.InstallBackdrop(p.c.View, k, false)
	if on {
		cocoa.SetGlassCornerRadius(p.c.View, 12)
		cocoa.SyncBackdrop(p.c.View)
	}
	if on && !had {
		p.c.AddControlListener(swt.ControlListenerControlMovedAdapter(func(*swt.ControlEvent) { cocoa.SyncBackdrop(p.c.View) }))
		p.c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { cocoa.SyncBackdrop(p.c.View) }))
	}
	if on || had {
		p.clear(on)
	}
}

func glassButton() Option {
	return Option{apply: func(c *swt.Control) {
		if c.GetStyle()&(swt.CHECK|swt.RADIO|swt.TOGGLE|swt.ARROW) == 0 {
			cocoa.SetGlassButton(c.View)
		}
	}}
}
