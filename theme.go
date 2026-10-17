package gowt

import "github.com/haiodo/gowt/swt"

// Dark reports whether the system is in dark mode (macOS appearance, Windows apps theme, Linux
// xdg-desktop-portal color-scheme). Run keeps the app following it; call from the UI thread.
func (a *App) Dark() bool { return systemDark() }

// OnThemeChange runs f on the UI thread when the system switches between light and dark.
// Cached widget colors and the Adaptive colours of widgets are already refreshed when f runs.
func (a *App) OnThemeChange(f func(dark bool)) {
	last := a.Dark()
	a.display.AddListener(swt.Settings, themeListener(func() {
		if d := a.Dark(); d != last {
			last = d
			f(d)
		}
	}))
}

type themeListener func()

func (l themeListener) HandleEvent(*swt.Event) { l() }

// painted is what a theme switch applies again to one control: the Paint last given for its text
// and for its background, nil for one never set.
type painted struct{ fg, bg Paint }

// followTheme makes a theme switch apply the Adaptive colours again. Run calls it before setup, so
// the colours are switched before the OnThemeChange callbacks run.
func (a *App) followTheme() {
	a.dark = a.Dark()
	a.paints = map[*swt.Control]*painted{}
	a.display.AddListener(swt.Settings, themeListener(func() {
		if d := a.Dark(); d != a.dark {
			a.repaint(d)
		}
	}))
}

func (a *App) repaint(dark bool) {
	a.dark = dark
	for c, p := range a.paints {
		if p.fg != nil {
			c.SetForeground(p.fg.in(dark).color())
		}
		if p.bg != nil {
			c.SetBackgroundWithColor(p.bg.in(dark).color())
		}
	}
}

// setPaint gives c the text colour (fg) or the background p in the current theme; nil restores the
// default. A container's children without their own background show its background (INHERIT_DEFAULT),
// as text fields, lists and buttons keep the native one. The App keeps an Adaptive for a theme switch.
func setPaint(c *swt.Control, p Paint, fg bool) {
	a := appOf(c)
	var col *swt.Color
	if p != nil {
		dark := systemDark()
		if a != nil {
			dark = a.dark
		}
		col = p.in(dark).color()
	}
	if fg {
		c.SetForeground(col)
	} else {
		c.SetBackgroundWithColor(col)
		if comp, ok := c.Impl().(swt.CompositeLike); ok && isContainer(c) {
			mode := int32(swt.INHERIT_DEFAULT)
			if p == nil {
				mode = swt.INHERIT_NONE
			}
			comp.AsComposite().SetBackgroundMode(mode)
		}
	}
	if a == nil {
		return
	}
	e := a.paints[c]
	if e == nil {
		if _, ok := p.(Adaptive); !ok {
			return
		}
		e = &painted{}
		a.paints[c] = e
		c.AddDisposeListener(disposeFunc(func() { delete(a.paints, c) }))
	}
	if fg {
		e.fg = p
	} else {
		e.bg = p
	}
}

// isContainer reports whether c is one of the facade's containers (VBox and the other boxes, Window,
// Group, Pages, Section, Scroll, Split, CoolBar, a Tab page), whose children inherit its background.
func isContainer(c *swt.Control) bool {
	switch c.GetName() {
	case "Composite", "Shell", "Group", "ScrolledComposite", "SashForm", "CoolBar":
		return true
	}
	return false
}
