package look

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

func setBackdrop(s *swt.Shell, b Backdrop) {
	if had := cocoa.InstallBackdrop(s.View, int(b), true); b != BackdropNone || had {
		clearBackground(&s.Composite, b != BackdropNone)
	}
}

func topInset(s *swt.Shell) int { return cocoa.ContentTopInset(s.View) }

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

func toolbarItems(items []ToolbarItem) []cocoa.ToolbarItem {
	ci := make([]cocoa.ToolbarItem, len(items))
	for i, it := range items {
		ci[i] = cocoa.ToolbarItem{Label: it.Title, Symbol: it.Symbol, Space: it.Space, Click: it.OnClick,
			Group: toolbarItems(it.Group), Nav: it.Nav, Search: it.OnSearch}
	}
	return ci
}

func setToolbar(s *swt.Shell, items []ToolbarItem) bool {
	ok := cocoa.InstallToolbar(s.View, toolbarItems(items))
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

// sidebarRadius and the 7-point inset match the Finder sidebar of macOS 26, measured on a screenshot.
const sidebarRadius = 18

func sidebarInset() int32 {
	if cocoa.GlassAvailable() {
		return 7
	}
	return 0
}

// setSidebarChrome puts the glass straight behind the sidebar rather than through setGlass, whose
// listeners would pin it to the sidebar frame: the glass also runs under the title bar strip.
func setSidebarChrome(s *swt.Shell, sb *swt.Composite) {
	cocoa.InstallBackdrop(sb.View, cocoa.BackdropGlass, false)
	cocoa.SetGlassCornerRadius(sb.View, sidebarRadius)
	clearBackground(sb, true)
	sourceLists(sb)
	if !sidebarShells[s] {
		sidebarShells[s] = true
		s.AddDisposeListener(&sidebarHooks{s})
		s.AddListener(swt.Show, &sidebarHooks{s})
	}
	if s.IsVisible() {
		refreshSidebar(s)
	}
}

// refreshSidebar lays the window out again and marks its whole view tree for display once the window is
// on screen: the first frame after opening showed the sidebar glass but none of the SWT views until a resize.
func refreshSidebar(s *swt.Shell) {
	s.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
		if s.IsDisposed() {
			return
		}
		s.LayoutOverload2(true, true)
		cocoa.NeedsDisplayTree(s.View.Window().ContentView())
	}))
}

var sidebarShells = map[*swt.Shell]bool{}

type sidebarHooks struct{ s *swt.Shell }

func (d *sidebarHooks) HandleEvent(*swt.Event) { refreshSidebar(d.s) }

func (d *sidebarHooks) WidgetDisposed(*swt.DisposeEvent) {
	cocoa.ReleaseSidebar(d.s.View)
	delete(sidebarShells, d.s)
}

// sourceLists gives the lists, tables and trees under c the source-list style.
func sourceLists(c *swt.Composite) {
	for _, ch := range c.GetChildren() {
		switch v := ch.Impl().(type) {
		case *swt.List, *swt.Table, *swt.Tree:
			cocoa.SetSourceList(ch.View)
		case *swt.Group:
			sourceLists(&v.Composite)
		case *swt.Composite:
			sourceLists(v)
		}
	}
}

func syncSidebar(s *swt.Shell, sb *swt.Composite, bg [4]int32) {
	cocoa.SetBackdropFrame(sb.View, float64(bg[0]), float64(bg[1]), float64(bg[2]), float64(bg[3]))
	cocoa.SetSidebarEdge(s.View, float64(bg[0]+bg[2]))
}
