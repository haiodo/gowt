package look

import (
	"fmt"
	"math"
	"os"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/qaview"
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
	c.SetBackgroundWithColor(clearColor(c.GetDisplay()))
	c.SetBackgroundMode(swt.INHERIT_FORCE)
}

// clearPanel clears a composite on glass with INHERIT_DEFAULT, not FORCE: under FORCE a Text field
// inherited the alpha-0 background and lost its field; text, combo and spinner fields keep theirs.
func clearPanel(c *swt.Composite) {
	c.SetBackgroundWithColor(clearColor(c.GetDisplay()))
	c.SetBackgroundMode(swt.INHERIT_DEFAULT)
}

// clearColor is never disposed: StyledText and SashForm keep the Color itself and fail on a disposed one
// at the next paint or new sash. A cocoa Color holds no native resource.
func clearColor(d *swt.Display) *swt.Color {
	return swt.NewColorDeviceRedGreenBlueAlpha(d, 0, 0, 0, 0)
}

func setGlass(c *swt.Composite, on bool) {
	k := cocoa.BackdropNone
	if on {
		k = cocoa.BackdropGlass
	}
	had := cocoa.InstallPanelBackdrop(c.View, k)
	if on {
		applyGlassRadius(c)
		cocoa.SyncBackdrop(c.View)
	}
	if on && !had {
		c.AddControlListener(swt.ControlListenerControlMovedAdapter(func(*swt.ControlEvent) { cocoa.SyncBackdrop(c.View) }))
		c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) {
			applyGlassRadius(c)
			cocoa.SyncBackdrop(c.View)
		}))
	}
	switch {
	case on:
		clearTree(c)
		clearAdded(c)
	case had:
		clearBackground(c, false)
		delete(clearRoots, c)
	}
}

// setMaterial is the SetGlass panel glass with radius 0; masksToBounds then clips the glass shadow at
// the panel edges, as ClipBackdrop does for the inspector.
func setMaterial(c *swt.Composite, on bool) {
	if on {
		c.SetDataKeyValue(glassRadiusKey, 0.0)
	} else {
		c.SetDataKeyValue(glassRadiusKey, nil)
	}
	setGlass(c, on)
}

const glassRadiusKey = "gowt.look.glassRadius"

func setGlassRadius(c *swt.Composite, r float64) {
	c.SetDataKeyValue(glassRadiusKey, r)
	applyGlassRadius(c)
}

// applyGlassRadius sets the radius asked by SetGlassRadius, 12 without one, on the glass of c; a
// capsule is half the shorter side of c now. It does nothing while c has no glass.
func applyGlassRadius(c *swt.Composite) {
	r, ok := c.GetDataKey(glassRadiusKey).(float64)
	if !ok {
		r = 12
	}
	if r < 0 {
		f := c.View.Frame()
		r = math.Min(f.Width, f.Height) / 2
	}
	cocoa.SetGlassCornerRadius(c.View, r)
}

func glassButton(c *swt.Control) {
	if c.GetStyle()&(swt.CHECK|swt.RADIO|swt.TOGGLE|swt.ARROW) == 0 {
		cocoa.SetGlassButton(c.View)
	}
}

const glassBorderKey = "gowt.look.glassBorder"

// setGlassBorder sets the stroke on the glass of c and, once per control, follows what changes it: the
// system appearance (a layer keeps the CGColor it was given) and the size of a capsule button.
func setGlassBorder(c *swt.Control, on bool) {
	cocoa.SetGlassBorder(c.View, on)
	if c.GetDataKey(glassBorderKey) != nil {
		return
	}
	t := &borderTheme{c}
	c.SetDataKeyValue(glassBorderKey, t)
	c.GetDisplay().AddListener(swt.Settings, t)
	c.AddListener(swt.Dispose, t)
	c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { cocoa.RefreshGlassBorder(c.View) }))
}

type borderTheme struct{ c *swt.Control }

func (t *borderTheme) HandleEvent(e *swt.Event) {
	if e.Type == swt.Dispose {
		t.c.GetDisplay().RemoveListener(swt.Settings, t)
		return
	}
	cocoa.RefreshGlassBorder(t.c.View)
}

const glassFocusKey = "gowt.look.glassFocus"

// setGlassFocusRing keeps the ring of c in step with the focus control through a display filter: a
// control added to c later is covered, and SWT sends FocusOut to the old control before FocusIn to the new.
func setGlassFocusRing(c *swt.Control, on bool) {
	cocoa.SetGlassFocusRing(c.View, on)
	f, _ := c.GetDataKey(glassFocusKey).(*focusRing)
	switch {
	case on && f == nil:
		f = &focusRing{c: c}
		c.SetDataKeyValue(glassFocusKey, f)
		d := c.GetDisplay()
		d.AddFilter(swt.FocusIn, f)
		d.AddFilter(swt.FocusOut, f)
		c.AddListener(swt.Dispose, f)
	case !on && f != nil:
		f.detach()
		c.SetDataKeyValue(glassFocusKey, nil)
	}
	if on {
		f.set(f.holdsFocus(c.GetDisplay().GetFocusControl()))
	}
}

type focusRing struct{ c *swt.Control }

func (f *focusRing) HandleEvent(e *swt.Event) {
	switch e.Type {
	case swt.Dispose:
		f.detach()
	case swt.FocusIn:
		f.set(f.holdsFocus(f.c.GetDisplay().GetFocusControl()))
	case swt.FocusOut:
		f.set(false)
	}
}

func (f *focusRing) detach() {
	d := f.c.GetDisplay()
	d.RemoveFilter(swt.FocusIn, f)
	d.RemoveFilter(swt.FocusOut, f)
	f.c.RemoveListener(swt.Dispose, f)
}

func (f *focusRing) set(focused bool) {
	if !f.c.IsDisposed() {
		cocoa.SetGlassFocused(f.c.View, focused)
	}
}

// holdsFocus is true for the focus control that is c or lies inside it.
func (f *focusRing) holdsFocus(focus *swt.Control) bool {
	for focus != nil {
		if focus == f.c {
			return true
		}
		parent := focus.GetParent()
		if parent == nil {
			return false
		}
		focus = &parent.Control
	}
	return false
}

func setRoundedCorners(*swt.Shell, bool) {}

func setDarkContent(*swt.Display, bool) {}

func toolbarItems(items []ToolbarItem) []cocoa.ToolbarItem {
	ci := make([]cocoa.ToolbarItem, len(items))
	for i, it := range items {
		ci[i] = cocoa.ToolbarItem{Label: it.Title, Symbol: it.Symbol, Space: it.Space, Click: it.OnClick,
			Group: toolbarItems(it.Group), Nav: it.Nav, Sidebar: it.Sidebar, Inspector: it.Inspector, Small: it.Small,
			Search: it.OnSearch, ID: it.ID, Disabled: it.Enabled != nil && !it.Enabled.Peek()}
		if it.Text != nil {
			ci[i].Text = it.Text.Peek()
		}
	}
	return ci
}

// nativeToolbar gives the QA driver the buttons of the NSToolbar.
type nativeToolbar struct{ b *cocoa.ToolbarButtons }

func (t nativeToolbar) Buttons() []qaview.ToolbarButton {
	var out []qaview.ToolbarButton
	for _, b := range t.b.List() {
		out = append(out, qaview.ToolbarButton{ID: b.ID, Title: b.Title, Enabled: b.Enabled, Bounds: b.Bounds})
	}
	return out
}

func (t nativeToolbar) Click(n int) bool { return t.b.Click(n) }

func setToolbar(s *swt.Shell, items []ToolbarItem) bool {
	ok := cocoa.InstallToolbar(s.View, toolbarItems(items))
	// The handles look the toolbar up on every call: SetSidebar rebuilds it behind this function.
	if ok {
		s.SetDataKeyValue(qaview.ToolbarKey, nativeToolbar{cocoa.ToolbarButtonsOf(s.View)})
	} else {
		s.SetDataKeyValue(qaview.ToolbarKey, nil)
	}
	hasSearch := false
	for _, it := range items {
		hasSearch = hasSearch || it.OnSearch != nil
	}
	if ok && hasSearch {
		s.SetDataKeyValue(qaview.ToolbarSearchKey, cocoa.ToolbarSearch(s.View))
	} else {
		s.SetDataKeyValue(qaview.ToolbarSearchKey, nil)
	}
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
		if r := it.Text; r != nil {
			rx.Effect(func() {
				t := r.Get()
				rx.Untrack(func() { cocoa.SetToolbarItemText(s.View, i, t) })
			})
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

// sidebarPadding is 0: the glass already insets the children, see sidebarInset.
func sidebarPadding() int32 { return 0 }

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
	clearAdded(sb)
	clearControl(sash)
	hookPanes(s)
}

// setInspectorChrome puts edge-to-edge glass behind the inspector, which AppKit 26 gives an
// NSSplitViewController inspector ("inspectors use an edge-to-edge glass that sits alongside the
// content", WWDC25 session 310); like the sidebar glass it also runs under the title bar strip.
func setInspectorChrome(s *swt.Shell, ib *swt.Composite, sash *swt.Control) {
	cocoa.InstallBackdrop(ib.View, cocoa.BackdropGlass, false)
	cocoa.SetGlassCornerRadius(ib.View, 0)
	cocoa.ClipBackdrop(ib.View)
	cocoa.InstallInspectorTracker(s.View)
	clearTree(ib)
	clearAdded(ib)
	clearControl(sash)
	hookPanes(s)
}

// inspectorInset is 0: the inspector is flush with the window edges, see setInspectorChrome.
func inspectorInset() int32 { return 0 }

// syncInspector also moves the divider the Inspector toolbar items follow to the inspector edge, or
// collapses it for a hidden inspector (bg 0 wide).
func syncInspector(ib *swt.Composite, bg [4]int32) {
	cocoa.SetBackdropFrame(ib.View, float64(bg[0]), float64(bg[1]), float64(bg[2]), float64(bg[3]))
	cocoa.SetEdgeLine(ib.View)
	cocoa.SetInspectorDivider(ib.View, float64(bg[0]), bg[2] == 0)
}

// hookPanes refreshes the window of a sidebar or inspector when it is shown, and now if it already is.
func hookPanes(s *swt.Shell) {
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
	delete(lastMinEdge, d.s)
}

// clearTree makes c and the controls under it paint no background of their own, so the sidebar glass
// shows, and gives lists, tables and trees the source-list style. Each control gets its own alpha-0
// background: SWT fills a control without one in the default window colour, and an alpha-0 background of
// a parent does not count as one there.
func clearTree(c *swt.Composite) {
	clearPanel(c)
	for _, ch := range c.GetChildren() {
		clearChild(ch, true)
	}
}

// clearChild clears one control under a cleared composite, and with deep the controls inside it. A control
// with a background of its own (set by the app, or already cleared) keeps it; the controls inside it do not.
func clearChild(ch *swt.Control, deep bool) {
	own := !ch.GetBackgroundColor().Equals(ch.DefaultBackground())
	var sub *swt.Composite
	switch v := ch.Impl().(type) {
	case *swt.List, *swt.Table, *swt.Tree:
		if !own {
			cocoa.SetSourceList(ch.View)
			clearControl(ch)
		}
	case *swt.Label, *swt.Link, *swt.StyledText:
		if !own {
			clearControl(ch)
		}
	case *swt.Text:
		// A single-line Text keeps its native field, as in the sidebar, but a borderless one is no bezel,
		// only a textBackgroundColor fill; a multi-line one is the text of an input row and shows the glass.
		switch {
		case own:
		case v.GetStyle()&swt.MULTI != 0:
			clearControl(ch)
			cocoa.ClearTextViewBackground(ch.View)
		case v.GetStyle()&(swt.BORDER|swt.SEARCH) == 0:
			glassField(ch)
		}
	case *swt.Button:
		if !own && v.GetStyle()&(swt.CHECK|swt.RADIO) != 0 {
			clearControl(ch)
		}
	case *swt.Group:
		sub = &v.Composite
	case *swt.ScrolledComposite:
		sub = &v.Composite
	case *swt.SashForm:
		sub = &v.Composite
	case *swt.Composite:
		sub = v
	}
	if sub == nil {
		return
	}
	if !own {
		clearPanel(sub)
	}
	if deep {
		for _, c := range sub.GetChildren() {
			clearChild(c, true)
		}
	}
}

// clearRoots are the composites of SetGlass, SetSidebar and SetInspector. SWT sends a Skin event for every new widget on
// the next event-loop turn, before it is drawn; a control created under a root then gets clearChild.
var (
	clearRoots   = map[*swt.Composite]bool{}
	skinDisplays = map[*swt.Display]bool{}
)

func clearAdded(c *swt.Composite) {
	if !clearRoots[c] {
		clearRoots[c] = true
		c.AddDisposeListener(clearRoot{c})
	}
	d := c.GetDisplay()
	if !skinDisplays[d] {
		skinDisplays[d] = true
		d.AddListener(swt.Skin, clearRoot{})
		d.DisposeExec(jrt.NewRunnable(func() { delete(skinDisplays, d) }))
	}
}

type clearRoot struct{ c *swt.Composite }

func (r clearRoot) WidgetDisposed(*swt.DisposeEvent) { delete(clearRoots, r.c) }

func (clearRoot) HandleEvent(e *swt.Event) {
	w, ok := e.Widget.Impl().(interface{ AsControl() *swt.Control })
	if !ok {
		return
	}
	ch := w.AsControl()
	if p := ch.GetParent(); p != nil {
		cocoa.KeepPanelBackdropBelow(p.View)
	}
	for p := ch.GetParent(); p != nil; p = p.GetParent() {
		if clearRoots[p] {
			clearChild(ch, false)
			return
		}
	}
}

const glassFieldKey = "gowt.look.glassField"

// glassField sets the fill of a borderless single-line Text on glass and sets it again at each theme
// switch, unless the app has given the Text a background since.
func glassField(c *swt.Control) {
	cocoa.SetGlassFieldFill(c.View)
	if c.GetDataKey(glassFieldKey) != nil {
		return
	}
	f := &fieldTheme{c}
	c.SetDataKeyValue(glassFieldKey, f)
	c.GetDisplay().AddListener(swt.Settings, f)
	c.AddListener(swt.Dispose, f)
}

type fieldTheme struct{ c *swt.Control }

func (f *fieldTheme) HandleEvent(e *swt.Event) {
	if e.Type == swt.Dispose {
		f.c.GetDisplay().RemoveListener(swt.Settings, f)
		return
	}
	if f.c.GetBackgroundColor().Equals(f.c.DefaultBackground()) {
		cocoa.SetGlassFieldFill(f.c.View)
	}
}

func clearControl(c *swt.Control) {
	c.SetBackgroundWithColor(clearColor(c.GetDisplay()))
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
	if ok && os.Getenv("GOWT_SIDEBAR_DEBUG") == "1" && e != lastMinEdge[s] {
		lastMinEdge[s] = e
		fmt.Fprintf(os.Stderr, "sidebar debug: Sidebar items need the edge at %.1f\n%s", e, cocoa.DescribeSidebarToolbar(s.View))
	}
	return int32(math.Ceil(e)), ok
}

// The toolbar lays its items out after the window layout ran, so a sidebar narrower than its Sidebar
// items is raised to them from here, before the user can drag it.
func init() {
	cocoa.SidebarToolbarLaidOut = func(win uintptr) {
		for s := range sidebarShells {
			if s.IsDisposed() || uintptr(s.View.Window().Id) != win || s.GetLayout() == nil {
				continue
			}
			if l, ok := s.GetLayout().Impl().(*paneLayout); ok && l.measure() && l.fitMin() {
				s.Layout()
			}
		}
	}
}

// lastMinEdge keeps GOWT_SIDEBAR_DEBUG from printing the same measurement on every layout.
var lastMinEdge = map[*swt.Shell]float64{}

// syncSidebar gets bg with width 0 for a hidden sidebar, which leaves no glass.
func syncSidebar(s *swt.Shell, sb *swt.Composite, bg [4]int32, hidden bool) {
	cocoa.SetBackdropFrame(sb.View, float64(bg[0]), float64(bg[1]), float64(bg[2]), float64(bg[3]))
	cocoa.SetSidebarHidden(s.View, hidden)
	cocoa.SetSidebarEdge(s.View, float64(bg[0]+bg[2]))
}

// startUnfocused leaves a sidebar or inspector window without a list with no focused control, the window
// itself first responder, as a macOS window without initialFirstResponder opens. SWT focuses its first
// control on Open and on each activation without a saved focus (Shell.Open and windowDidBecomeKey
// traverse), so the window takes the first responder back after each activation that found no control
// focused when the window was last left. Shell.SetFocus cannot do it: it focuses the first child.
func startUnfocused(s *swt.Shell) {
	h := &unfocused{s: s, on: true}
	s.AddListener(swt.Activate, h)
	s.AddListener(swt.Deactivate, h)
}

type unfocused struct {
	s  *swt.Shell
	on bool
}

func (u *unfocused) HandleEvent(e *swt.Event) {
	if e.Type == swt.Deactivate {
		u.on = cocoa.WindowIsFirstResponder(u.s.View)
		return
	}
	if !u.on {
		return
	}
	// Activate comes before SWT restores the focus; this runs after it.
	u.s.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
		if !u.s.IsDisposed() {
			cocoa.ClearFirstResponder(u.s.View)
		}
	}))
}
