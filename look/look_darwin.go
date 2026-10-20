package look

import (
	"fmt"
	"math"
	"os"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/rx"
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
			Group: toolbarItems(it.Group), Nav: it.Nav, Sidebar: it.Sidebar, Search: it.OnSearch, Disabled: it.Enabled != nil && !it.Enabled.Peek()}
	}
	return ci
}

func setToolbar(s *swt.Shell, items []ToolbarItem) bool {
	ok := cocoa.InstallToolbar(s.View, toolbarItems(items))
	if stop := toolbarBindings[s]; stop != nil {
		stop()
		delete(toolbarBindings, s)
	}
	if ok {
		toolbarBindings[s] = rx.Scope(func() { bindToolbar(s, items) })
	}
	if ok && !toolbarShells[s] {
		toolbarShells[s] = true
		s.AddDisposeListener(&toolbarDisposer{s})
	}
	s.Layout()
	return ok
}

// toolbarBindings ends the Enabled effects of a shell's toolbar when it is replaced or disposed.
var toolbarBindings = map[*swt.Shell]func(){}

func bindToolbar(s *swt.Shell, items []ToolbarItem) {
	bind := func(item, segment int, r rx.Reader[bool]) {
		if r == nil {
			return
		}
		rx.Effect(func() {
			on := r.Get()
			rx.Untrack(func() { cocoa.SetToolbarItemEnabled(s.View, item, segment, on) })
		})
	}
	for i, it := range items {
		bind(i, -1, it.Enabled)
		for j, g := range it.Group {
			bind(i, j, g.Enabled)
		}
	}
}

func setGlassGroup(c *swt.Composite, spacing float64) { cocoa.InstallGlassGroup(c.View, spacing) }

// toolbarShells holds the shells that already have the dispose hook; the hook frees the Go click handlers.
var toolbarShells = map[*swt.Shell]bool{}

type toolbarDisposer struct{ s *swt.Shell }

func (d *toolbarDisposer) WidgetDisposed(*swt.DisposeEvent) {
	cocoa.ReleaseToolbar(d.s.View)
	delete(toolbarShells, d.s)
	if stop := toolbarBindings[d.s]; stop != nil {
		stop()
		delete(toolbarBindings, d.s)
	}
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
func setSidebarChrome(s *swt.Shell, sb *swt.Composite, sash *swt.Control) {
	cocoa.InstallBackdrop(sb.View, cocoa.BackdropGlass, false)
	cocoa.SetGlassCornerRadius(sb.View, sidebarRadius)
	clearTree(sb)
	clearControl(sash)
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
		if os.Getenv("GOWT_SIDEBAR_DEBUG") == "1" {
			debugSidebar(s)
		}
	}))
}

var sidebarShells = map[*swt.Shell]bool{}

type sidebarHooks struct{ s *swt.Shell }

func (d *sidebarHooks) HandleEvent(*swt.Event) { refreshSidebar(d.s) }

func (d *sidebarHooks) WidgetDisposed(*swt.DisposeEvent) {
	cocoa.ReleaseSidebar(d.s.View)
	delete(sidebarShells, d.s)
}

// clearTree makes c and the controls under it paint no background of their own, so the sidebar glass
// shows, and gives lists, tables and trees the source-list style. Each control gets its own alpha-0
// background: SWT fills a control without one in the default window colour, and an alpha-0 background of
// a parent does not count as one there. Controls added later are not covered.
func clearTree(c *swt.Composite) {
	clearBackground(c, true)
	for _, ch := range c.GetChildren() {
		switch v := ch.Impl().(type) {
		case *swt.List, *swt.Table, *swt.Tree:
			cocoa.SetSourceList(ch.View)
			clearControl(ch)
		case *swt.Label, *swt.Link:
			clearControl(ch)
		case *swt.Button:
			if v.GetStyle()&(swt.CHECK|swt.RADIO) != 0 {
				clearControl(ch)
			}
		case *swt.Group:
			clearTree(&v.Composite)
		case *swt.Composite:
			clearTree(v)
		}
	}
}

func clearControl(c *swt.Control) {
	col := swt.NewColorDeviceRedGreenBlueAlpha(c.GetDisplay(), 0, 0, 0, 0)
	c.SetBackgroundWithColor(col)
	col.Dispose()
}

// debugSidebar prints, for every list, table and tree in the sidebar of s, the SWT background state
// of it and its parents and the AppKit view chain up to the content view (GOWT_SIDEBAR_DEBUG=1).
func debugSidebar(s *swt.Shell) {
	var walk func(c *swt.Composite)
	walk = func(c *swt.Composite) {
		for _, ch := range c.GetChildren() {
			switch v := ch.Impl().(type) {
			case *swt.List, *swt.Table, *swt.Tree:
				fmt.Fprintf(os.Stderr, "sidebar debug: %T\n", v)
				for w := ch; w != nil; w = parentControl(w) {
					b := w.GetBackground()
					mode := int32(-1)
					if p, ok := w.Impl().(interface{ GetBackgroundMode() int32 }); ok {
						mode = p.GetBackgroundMode()
					}
					fmt.Fprintf(os.Stderr, "  swt %T background %d,%d,%d,%d mode %d\n", w.Impl(), b.GetRed(), b.GetGreen(), b.GetBlue(), b.GetAlpha(), mode)
				}
				fmt.Fprint(os.Stderr, cocoa.DescribeViewChain(ch.View))
			case *swt.Composite:
				walk(v)
			case *swt.Group:
				walk(&v.Composite)
			}
		}
	}
	walk(&s.Composite)
}

// sidebarDivider is 0: the sash lies over the sidebar edge, transparent (setSidebarChrome).
func sidebarDivider() int32 { return 0 }

// itemsMinEdge is the sidebar edge x the Sidebar toolbar items need, from the toolbar's last layout.
func itemsMinEdge(s *swt.Shell) (int32, bool) {
	e, ok := cocoa.SidebarItemsMinEdge(s.View)
	return int32(math.Ceil(e)), ok
}

// syncSidebar gets bg with width 0 for a hidden sidebar, which leaves no glass.
func syncSidebar(s *swt.Shell, sb *swt.Composite, bg [4]int32, hidden bool) {
	cocoa.SetBackdropFrame(sb.View, float64(bg[0]), float64(bg[1]), float64(bg[2]), float64(bg[3]))
	cocoa.SetSidebarHidden(s.View, hidden)
	cocoa.SetSidebarEdge(s.View, float64(bg[0]+bg[2]))
}
