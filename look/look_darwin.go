package look

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func setBackdrop(s *swt.Shell, b Backdrop) {
	if had := cocoa.InstallBackdrop(s.View, int(b), true); b != BackdropNone || had {
		clearBackground(&s.Composite, b != BackdropNone)
	}
}

func classic() { cocoa.RequireCompatibleLook() }

func setFullSizeContent(s *swt.Shell, on bool) {
	cocoa.SetFullSizeContent(s.View, on)
	s.Layout()
}

// clearBackground makes the composite paint nothing of its own and lets its children inherit that, the SWT
// way (alpha-0 background plus INHERIT_FORCE), so a backdrop behind it stays visible.
func clearBackground(c *swt.Composite, on bool) {
	if !on {
		c.SetBackgroundWithColor(nil)
		c.SetBackgroundMode(swt.NONE)
		return
	}
	col := swt.NewColorDeviceRedGreenBlueAlpha(c.GetDisplay(), 0, 0, 0, 0)
	c.SetBackgroundWithColor(col)
	col.Dispose()
	c.SetBackgroundMode(swt.INHERIT_FORCE)
}

func setGlass(c *swt.Composite, on bool) {
	k := cocoa.BackdropNone
	if on {
		k = cocoa.BackdropGlass
	}
	had := cocoa.InstallBackdrop(c.View, k, false)
	if on {
		cocoa.SetGlassCornerRadius(c.View, 12)
		cocoa.SyncBackdrop(c.View)
	}
	if on && !had {
		c.AddControlListener(swt.ControlListenerControlMovedAdapter(func(*swt.ControlEvent) { cocoa.SyncBackdrop(c.View) }))
		c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { cocoa.SyncBackdrop(c.View) }))
	}
	if on || had {
		clearBackground(c, on)
	}
}

func glassButton(c *swt.Control) {
	if c.GetStyle()&(swt.CHECK|swt.RADIO|swt.TOGGLE|swt.ARROW) == 0 {
		cocoa.SetGlassButton(c.View)
	}
}

func setRoundedCorners(*swt.Shell, bool) {}

func setDarkContent(*swt.Display, bool) {}

func setToolbar(s *swt.Shell, items []ToolbarItem) bool {
	ci := make([]cocoa.ToolbarItem, len(items))
	for i, it := range items {
		ci[i] = cocoa.ToolbarItem{Label: it.Title, Symbol: it.Symbol, Space: it.Space, Click: it.OnClick}
	}
	ok := cocoa.InstallToolbar(s.View, ci)
	if ok && !toolbarShells[s] {
		toolbarShells[s] = true
		s.AddDisposeListener(&toolbarDisposer{s})
	}
	s.Layout()
	return ok
}

func setGlassGroup(c *swt.Composite, spacing float64) { cocoa.InstallGlassGroup(c.View, spacing) }

// toolbarShells holds the shells that already have the dispose hook; the hook frees the Go click handlers.
var toolbarShells = map[*swt.Shell]bool{}

type toolbarDisposer struct{ s *swt.Shell }

func (d *toolbarDisposer) WidgetDisposed(*swt.DisposeEvent) {
	cocoa.ReleaseToolbar(d.s.View)
	delete(toolbarShells, d.s)
}
