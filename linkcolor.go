package gowt

import "github.com/haiodo/gowt/swt"

// linkStyle is the app-wide link colour and the Links it applies to.
type linkStyle struct {
	color    *Adaptive
	links    map[*swt.Link]struct{}
	watching bool
}

// SetLinkColor sets the colour of the anchors of every Link, light and dark, instead of the system
// link colour (on GNOME the theme accent). nil gives the system colour back, which is the default.
// It applies to the Links that exist and to those created later, and follows a theme switch.
func (a *App) SetLinkColor(c *Adaptive) {
	a.link.color = c
	if c != nil && !a.link.watching {
		a.link.watching = true
		a.OnThemeChange(func(bool) { a.applyLinkColor() })
	}
	a.applyLinkColor()
}

func (a *App) applyLinkColor() {
	for l := range a.link.links {
		a.styleLink(l)
	}
}

func (a *App) styleLink(l *swt.Link) {
	if a.link.color == nil {
		l.SetLinkForeground(nil)
		return
	}
	c := a.link.color.in(a.Dark())
	l.SetLinkForeground(c.color())
}

// trackLink gives a new Link the app's link colour.
func (a *App) trackLink(l *swt.Link) {
	if a.link.links == nil {
		a.link.links = map[*swt.Link]struct{}{}
	}
	a.link.links[l] = struct{}{}
	l.AddDisposeListener(disposeFunc(func() { delete(a.link.links, l) }))
	if a.link.color != nil {
		a.styleLink(l)
	}
}
