package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func classicLook() { cocoa.RequireCompatibleLook() }

func (w *Window) setFullSizeContent(on bool) { cocoa.SetFullSizeContent(w.shell.View, on) }

func (p *panel) setGlass(on bool) {
	k := cocoa.BackdropNone
	if on {
		k = cocoa.BackdropGlass
	}
	cocoa.InstallBackdrop(p.c.View, k, false)
	if on {
		cocoa.SetGlassCornerRadius(p.c.View, 12)
	}
}

func glassButton() Option {
	return Option{apply: func(c *swt.Control) { cocoa.SetGlassButton(c.View) }}
}
