// Package look sets the platform look of a gowt app: the window material, macOS 26 Liquid Glass,
// Windows 11 corners and dark content. Every call does nothing on a system without the feature.
package look

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// Backdrop is the material behind a window's content.
type Backdrop int

const (
	BackdropNone        Backdrop = iota // opaque, the platform default
	BackdropTranslucent                 // blurred desktop behind the window: NSVisualEffectView, Acrylic
	BackdropGlass                       // Liquid Glass on macOS 26, Mica on Windows 11
)

// Container is a gowt container that has a composite: Panel, Window, Group, Split and CoolBar.
type Container interface{ AsComposite() *swt.Composite }

// SetBackdrop sets the window material. A value the OS cannot show falls back to the nearest
// one it can; Linux ignores it.
func SetBackdrop(w *gowt.Window, b Backdrop) { setBackdrop(w.Unwrap(), b) }

// SetRoundedCorners forces rounded (true) or square (false) window corners on Windows 11; other
// systems and older Windows builds ignore it. Windows 11 rounds top-level windows by default.
func SetRoundedCorners(w *gowt.Window, on bool) { setRoundedCorners(w.Unwrap(), on) }

// Classic asks macOS 26 and newer to keep the look of earlier releases instead of Liquid
// Glass (the UIDesignRequiresCompatibility key). Call it before gowt.Run. Other systems ignore it.
// Without an Info.plist the key can only be set at run time and macOS may not read it there;
// building with -ldflags=-macsdk=15.0 records an older SDK in the binary, which is reliable.
func Classic() { classic() }

// SetFullSizeContent lets the window content extend under the title bar, which becomes
// transparent, so backgrounds and glass run under it. The window layout (Grid, Row, Form) gets the title bar and
// toolbar height as extra top margin, kept in step on resize, SetToolbar and full screen; other
// layouts need their own margin. macOS only; elsewhere it does nothing.
func SetFullSizeContent(w *gowt.Window, on bool) {
	setFullSizeContent(w.Unwrap(), on)
	if !insetShells[w] {
		insetShells[w] = true
		w.Unwrap().AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { refreshInset(w) }))
	}
	refreshInset(w)
}

var insetShells = map[*gowt.Window]bool{}

// refreshInset keeps the top margin of the window layout equal to the title bar and toolbar height
// that full-size content runs under; it is zero without full-size content and off macOS.
func refreshInset(w *gowt.Window) { w.SetTopInset(topInset(w.Unwrap())) }

// SetGlass puts a Liquid Glass surface behind the container's content, with rounded corners; the
// container and the controls under it, also those created later, stop painting their own background
// as in a SetSidebar sidebar, and lists, tables and trees get the source-list style. It needs
// macOS 26; older macOS shows a blurred material, other systems nothing.
func SetGlass(c Container, on bool) { setGlass(c.AsComposite(), on) }

// SetMaterial puts the material of a SetSidebar sidebar and a SetInspector inspector behind the
// container's content, flush with its edges and with square corners, for a pane of a Split such as
// a code panel under a canvas; SetGlass gives a rounded card instead. macOS 26: the edge-to-edge
// Liquid Glass of the inspector, clipped to the container. Older macOS: the blurred material the
// sidebar and inspector fall back to. As with SetGlass, the container and the controls under it,
// also those created later, stop painting their own background. Windows: the Fluent window
// background (Layer) of the sidebar and inspector panels; Linux: the GTK window background. There
// the controls inside keep their own background: give a StyledText gowt.NoBackground. In a Split, the
// sash beside the container draws a hairline in the separator colour, the thin divider of an
// NSSplitView, on every system.
func SetMaterial(c Container, on bool) {
	comp := c.AsComposite()
	setMaterial(comp, on)
	comp.SetDataKeyValue(materialOnKey, on)
	if split := comp.GetParent(); on && split != nil {
		if _, ok := split.Impl().(*swt.SashForm); ok {
			hookDividers(split)
		}
	}
}

const (
	materialOnKey = "gowt.look.materialOn"
	dividerKey    = "gowt.look.divider"
)

// hookDividers gives each sash of split the divider painter, once; SashForm makes its sashes in its
// layout, so the sashes are looked for again after each resize.
func hookDividers(split *swt.Composite) {
	if split.GetDataKey(dividerKey) != nil {
		return
	}
	split.SetDataKeyValue(dividerKey, true)
	hook := func() {
		if split.IsDisposed() {
			return
		}
		for _, ch := range split.GetChildren() {
			if _, ok := ch.Impl().(*swt.Sash); !ok || ch.GetDataKey(dividerKey) != nil {
				continue
			}
			sash := ch
			sash.SetDataKeyValue(dividerKey, true)
			sash.AddPaintListener(painter(func(e *swt.PaintEvent) { paintDivider(e, sash, split) }))
			sash.Redraw()
		}
	}
	hook()
	later := func() { split.GetDisplay().AsyncExec(jrt.NewRunnable(hook)) }
	later()
	split.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { later() }))
}

// paintDivider draws the hairline down the middle of sash when a SetMaterial pane of split touches it.
func paintDivider(e *swt.PaintEvent, sash *swt.Control, split *swt.Composite) {
	s := sash.GetBounds()
	beside := false
	for _, ch := range split.GetChildren() {
		if on, _ := ch.GetDataKey(materialOnKey).(bool); !on {
			continue
		}
		b := ch.GetBounds()
		touch := func(a, b int32) bool { return a-b <= 1 && b-a <= 1 }
		beside = beside || touch(s.Y+s.Height, b.Y) || touch(b.Y+b.Height, s.Y) ||
			touch(s.X+s.Width, b.X) || touch(b.X+b.Width, s.X)
	}
	if !beside {
		return
	}
	bg := sash.GetBackground().GetRGB()
	fg := barForeground(*sash.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND).GetRGB(), *bg)
	c := swt.NewColorRgb(blend(*bg, fg, 12))
	e.Gc.SetForeground(c)
	if s.Width > s.Height {
		e.Gc.DrawLine(0, s.Height/2, s.Width, s.Height/2)
	} else {
		e.Gc.DrawLine(s.Width/2, 0, s.Width/2, s.Height)
	}
	c.Dispose()
}

// Capsule is the radius for SetGlassRadius that rounds the corners fully: half the shorter side,
// followed on every resize.
const Capsule = -1

// SetGlassRadius sets the corner radius in points of the glass SetGlass puts behind c, 12 by default;
// Capsule makes the shorter side a half circle. It can come before SetGlass and stays when the glass
// is turned off and on. For a field that grows over several lines use a fixed radius of half the
// one-line height, as the iPad composer does: Capsule would turn a tall field into an oval. Other
// systems have no glass: nothing happens.
func SetGlassRadius(c Container, r float64) { setGlassRadius(c.AsComposite(), r) }

// GlassButton gives a push button (not Check, Radio, Toggle or Arrow) the Liquid Glass bezel on macOS 26 and newer; elsewhere the
// button is unchanged.
func GlassButton() gowt.Option { return gowt.Custom(glassButton) }

// SetGlassBorder draws a hairline around the glass SetGlass puts behind c: one device pixel in the
// system separator colour, following the corner radius. On a white content panel a glass capsule is
// otherwise almost invisible. The colour follows light and dark mode and the stroke follows
// resizes. It can come before SetGlass. A panel in a SetGlassGroup gets none. Other systems and
// macOS before 26: nothing happens.
func SetGlassBorder(c Container, on bool) { setGlassBorder(&c.AsComposite().Control, on) }

// SetGlassFocusRing turns the glass stroke of c into a ring in the keyboard focus colour
// (NSColor.keyboardFocusIndicatorColor, the accent colour) while c or any control inside it has
// keyboard focus, as a text field has one; on blur the SetGlassBorder stroke is back. It follows the
// corner radius, resizes (AutoGrow too), the accent colour and light and dark mode. Two points wide,
// inside the glass: the glass is clipped to its corners. macOS 26 only; elsewhere nothing happens.
// macOS 26 Messages shows no such ring on its composer, so this is opt-in.
func SetGlassFocusRing(c Container, on bool) { setGlassFocusRing(&c.AsComposite().Control, on) }

// GlassBorder is SetGlassBorder(true) for a glass button; it works with GlassButton and
// gowt.Circle in either order.
func GlassBorder() gowt.Option {
	return gowt.Custom(func(c *swt.Control) { setGlassBorder(c, true) })
}

// SetDarkContent on Windows turns on the dark theme of the window content (scroll bars, tables, trees,
// buttons) through undocumented uxtheme exports. Call before creating windows; it does not switch live.
// GTK and AppKit theme the content themselves, so it does nothing there.
func SetDarkContent(a *gowt.App, on bool) { setDarkContent(a.Unwrap(), on) }

// ToolbarItem is one entry of the window toolbar.
type ToolbarItem struct {
	Title   string // label and tooltip
	Symbol  string // SF Symbol name such as "plus" or "square.and.arrow.up"; "" shows the title only
	OnClick func()
	Space   bool // flexible space; the other fields are ignored
	// Group makes the item one capsule of segments, such as the back and forward pair; each segment
	// has its own Title, Symbol and OnClick, and the item's own Symbol and OnClick are not used.
	Group []ToolbarItem
	// Nav places the item at the leading edge before the window title, where back and forward go.
	Nav bool
	// Sidebar places the item over the sidebar of a SetSidebar window, ending at the sidebar edge: on
	// macOS in the title bar, level with the traffic lights (Finder, Mail), on Windows and Linux in a row
	// at the top of the sidebar. Without SetSidebar it is an ordinary item.
	Sidebar bool
	// Inspector places the item over the inspector of a SetInspector window, starting at its left edge
	// and following it; with the inspector hidden it sits at the right end. On macOS it comes after an
	// NSTrackingSeparatorToolbarItem, on Windows and Linux in a row at the top of the inspector. Without
	// SetInspector it is an ordinary item.
	Inspector bool
	// Text, when set, makes a push item a bordered button showing this text instead of a symbol and
	// following its changes, such as a zoom level "100%"; Title stays the tooltip.
	Text rx.Reader[string]
	// Small gives a push item, a text item, a Group or a search field the small control size
	// (NSControlSizeSmall, an 11-point font on macOS); elsewhere smaller icons and text.
	Small bool
	// OnSearch makes the item a search field (NSSearchToolbarItem) with Title as the placeholder. It
	// gets the text after a typing pause, on Return and on clearing.
	OnSearch func(text string)
	// ID names the item for the QA driver (qa.ID), as gowt.ID names a widget.
	ID string
	// Enabled, when set, enables and disables a push item or a Group segment as it changes, for as
	// long as the toolbar stays; nil keeps the item enabled.
	Enabled rx.Reader[bool]
}

// SetToolbar puts a native unified toolbar (NSToolbar) on the window, replacing an earlier one; with
// no items it removes the toolbar. On macOS 26 the items are Liquid Glass. It reports whether the
// native toolbar exists. Where it does not (Windows, Linux), a window laid out by SetSidebar or
// SetInspector, before or after this call, gets the items as a bar of gowt widgets at the top of its
// content column: flat icon buttons, the window title, a search field, in the content's colour. A
// window without either gets nothing there; build your own gowt.ToolBar in that case. Use it together with
// SetFullSizeContent only if the content leaves room for the bar.
func SetToolbar(w *gowt.Window, items ...ToolbarItem) bool {
	ok := setToolbar(w.Unwrap(), items)
	if b := bars[w.Unwrap()]; ok && b != nil {
		b.items = nil
		b.build()
	}
	if !ok {
		b := barOf(w)
		b.items = items
		b.build()
	}
	refreshInset(w)
	return ok
}

// FocusSearch focuses the search field of the toolbar SetToolbar made for w (the item with OnSearch),
// for an app that binds Cmd+F; it selects the text there and reports whether there is such a field. On
// macOS the field is the NSSearchToolbarItem: a collapsed one expands first and takes the focus on the
// toolbar's next layout. Elsewhere it is the search Text of the bar SetToolbar builds. UI thread only.
func FocusSearch(w *gowt.Window) bool {
	s := w.Unwrap()
	if b := bars[s]; b != nil && b.search != nil && !b.search.Unwrap().IsDisposed() {
		b.search.Focus()
		b.search.Unwrap().SelectAll()
		return true
	}
	if f, ok := qaview.ToolbarSearchOf(s); ok {
		return f.Focus()
	}
	return false
}

// SetGlassGroup makes the glass surfaces (SetGlass) of the direct children of c share one
// NSGlassEffectContainerView, so surfaces closer than spacing points merge into one shape. Call it
// before the children get their glass. macOS 26 only; elsewhere it does nothing.
func SetGlassGroup(c Container, spacing float64) { setGlassGroup(c.AsComposite(), spacing) }

// SetPanesOverlay opts into floating panes over content on macOS 26; other systems keep their layout.
// Call before or after SetSidebar and SetInspector. Bottom panels stay below content.
func SetPanesOverlay(w *gowt.Window, on bool) {
	on = on && sidebarInset() > 0
	w.Unwrap().SetDataKeyValue("look.panesOverlay", on)
	if l := layoutOf(w); l != nil {
		l.overlay = on
		w.Unwrap().Layout()
	}
}

// SetSidebar lays w out the way Finder and System Settings are: sidebar on the left, width points wide,
// content filling the rest. It replaces the window layout, so w must hold just these two, or these and
// the inspector of SetInspector, called before or after with the same content. On macOS 26
// the sidebar floats: Liquid Glass with rounded corners, inset from the window edges and running under
// the traffic lights; the window gets full-size content, sidebar children and content start below the
// title bar and toolbar, and the items and title of SetToolbar start past the sidebar. Older macOS
// shows a flush sidebar material; Windows and Linux put two plain panels side by side, with the
// SetToolbar items as a bar above the content. A background on w itself paints over the macOS
// sidebar glass; colour the content instead.
//
// The user resizes the sidebar by dragging its edge, which stops at 120 points or at the room its
// Sidebar toolbar items need. Read and change that with SidebarVisible, SetSidebarVisible,
// SidebarWidth and SetSidebarWidth, and keep it between runs with OnSidebarChange.
func SetSidebar(w *gowt.Window, sidebar, content Container, width int) {
	l := panesOf(w, content)
	sb := sidebar.AsComposite()
	if l.side != nil {
		l.side.sash.Dispose()
	}
	l.side = l.newPane(sb, width, sidebarPane)
	setSidebarChrome(w.Unwrap(), sb, &l.side.sash.Control)
	padSidebar(sb)
	b := barOf(w)
	b.sidebar = sidebar
	b.build()
}

// padSidebar gives the children of a sidebar the side margin the macOS glass gives them, where the
// sidebar is a plain panel. It applies to a VBox or HBox sidebar and adds to the margin it has.
func padSidebar(sb *swt.Composite) {
	m := sidebarPadding()
	if m == 0 || sb.GetLayout() == nil {
		return
	}
	if g, ok := sb.GetLayout().Impl().(interface{ Grid() *swt.GridLayout }); ok {
		g.Grid().MarginLeft += m
		g.Grid().MarginRight += m
	}
}

// SetInspector puts an inspector on the right of w, width points wide, the trailing counterpart of
// SetSidebar: content fills the rest, or the room between a SetSidebar sidebar and the inspector. It
// replaces the window layout, so w must hold just the content, the inspector and a SetSidebar
// sidebar. On macOS 26 the inspector is edge-to-edge Liquid Glass alongside the content, as AppKit
// draws an NSSplitViewController inspector: flush with the top, right and bottom window edges and
// running under the toolbar, its children starting below it. Older macOS shows a flush material;
// Windows and Linux a plain panel. Without a sidebar there, the window also gets full-size content,
// and on Windows and Linux the SetToolbar items as a bar above the content.
//
// The user resizes the inspector by dragging its left edge, which stops at 160 points and leaves the
// content at least 200. Read and change that with InspectorVisible, SetInspectorVisible,
// InspectorWidth and SetInspectorWidth, and keep it between runs with OnInspectorChange.
func SetInspector(w *gowt.Window, content, inspector Container, width int) {
	l := panesOf(w, content)
	ib := inspector.AsComposite()
	if l.insp != nil {
		l.insp.sash.Dispose()
	}
	l.insp = l.newPane(ib, width, inspectorPane)
	setInspectorChrome(w.Unwrap(), ib, &l.insp.sash.Control)
	barOf(w).build()
}

// SetBottomPanel puts a panel under the content of w, height points high, for a console or a code
// pane below a canvas. It shares the layout of SetSidebar and SetInspector: content is the same
// content, and the panel spans its width, between a sidebar and an inspector, down to the bottom
// window edge, with a hairline in the system separator colour along its top. On macOS 26 the panel
// is the inspector's edge-to-edge glass, and under a floating sidebar the glass runs on to the
// window's left edge, as AppKit extends a split item beneath the sidebar; the panel's children stay
// right of the sidebar. The panel gets the SetMaterial material, so a SetMaterial on it is turned
// off. Older macOS shows the flush material; Windows and Linux the SetMaterial background, flush with
// the content, with the hairline in the sash gap above it.
//
// The user resizes the panel by dragging its top edge, which stops at 60 points and leaves the
// content at least 100. Read and change that with BottomPanelVisible, SetBottomPanelVisible,
// BottomPanelHeight and SetBottomPanelHeight, and keep it between runs with OnBottomPanelChange.
func SetBottomPanel(w *gowt.Window, content, panel Container, height int) {
	l := panesOf(w, content)
	pb := panel.AsComposite()
	if l.bottom != nil {
		l.bottom.sash.Dispose()
	}
	if on, _ := pb.GetDataKey(materialOnKey).(bool); on {
		SetMaterial(panel, false)
	}
	l.bottom = l.newPane(pb, height, bottomPane)
	setBottomChrome(w.Unwrap(), pb, &l.bottom.sash.Control)
	w.Unwrap().Layout()
}

// panesOf returns the layout of a SetSidebar or SetInspector window, making it the layout of w with
// content as its content if w has none yet.
func panesOf(w *gowt.Window, content Container) *paneLayout {
	s := w.Unwrap()
	ct := &content.AsComposite().Control
	defer func() { barOf(w).content = ct }()
	if l := layoutOf(w); l != nil {
		l.content = ct
		return l
	}
	SetFullSizeContent(w, true)
	overlay, _ := s.GetDataKey("look.panesOverlay").(bool)
	l := &paneLayout{Layout: swt.NewLayout(), w: w, content: ct, overlay: overlay}
	l.SetImpl_(l)
	// The shell minimum is read once: on macOS 26 it read back 20 points taller than it was set, so a
	// value read after each set grew.
	m := s.GetMinimumSize()
	l.appMin = [2]int32{m.X, m.Y}
	s.SetDataKeyValue(minSizeHookKey, func(width, height int) {
		l.appMin = [2]int32{int32(width), int32(height)}
		l.fitMinSize()
	})
	s.SetLayout(l)
	s.AddPaintListener(painter(l.paintUnder))
	// Without gowt.Focused the window opens with the focus on its first list, as Finder does, not on a
	// button of the sidebar.
	s.SetDataKeyValue(focusDefaultKey, func() *swt.Control {
		if c := firstList(l.content.Impl().(interface{ AsComposite() *swt.Composite }).AsComposite()); c != nil {
			return c
		}
		if l.side != nil {
			if c := firstList(l.side.c); c != nil {
				return c
			}
		}
		startUnfocused(s)
		return nil
	})
	return l
}

// minSizeHookKey holds the func that gowt.Window.SetMinSize calls instead of setting the shell
// minimum; gowt has the same key.
const minSizeHookKey = "gowt.look.minSize"

// focusDefaultKey holds the func that gowt.Window.Show asks for the control to focus when no widget is
// Focused; gowt has the same key.
const focusDefaultKey = "gowt.look.focus"

// firstList returns the first visible List, Table or Tree under c, depth first, or nil.
func firstList(c *swt.Composite) *swt.Control {
	for _, ch := range c.GetChildren() {
		if !ch.GetVisible() {
			continue
		}
		switch v := ch.Impl().(type) {
		case *swt.List, *swt.Table, *swt.Tree:
			return ch
		case interface{ AsComposite() *swt.Composite }:
			if l := firstList(v.AsComposite()); l != nil {
				return l
			}
		}
	}
	return nil
}

// eachList calls f with the List, Table and Tree widgets under c, depth first.
func eachList(c *swt.Composite, f func(any)) {
	for _, ch := range c.GetChildren() {
		switch v := ch.Impl().(type) {
		case *swt.List, *swt.Table, *swt.Tree:
			f(v)
		case interface{ AsComposite() *swt.Composite }:
			eachList(v.AsComposite(), f)
		}
	}
}

// SetSidebarVisible shows or hides the sidebar of a SetSidebar window; hidden, the content takes the
// whole width and the Sidebar toolbar items move to the start of the toolbar.
func SetSidebarVisible(w *gowt.Window, on bool) { sidebarOf(w).setVisible(on) }

// SidebarVisible reports whether the sidebar of a SetSidebar window is shown; false without one.
func SidebarVisible(w *gowt.Window) bool { return sidebarOf(w).visible() }

// SetSidebarWidth sets the sidebar width in points; a hidden sidebar comes back at it. The width is at
// least 120 and at least the room of the Sidebar toolbar items, as far as the window knows it yet.
func SetSidebarWidth(w *gowt.Window, width int) { sidebarOf(w).setWidth(width) }

// SidebarWidth is the sidebar width in points, also while it is hidden; 0 without SetSidebar.
func SidebarWidth(w *gowt.Window) int { return sidebarOf(w).widthOrZero() }

// OnSidebarChange runs f after the sidebar of a SetSidebar window was shown, hidden or resized: by
// SetSidebarVisible, SetSidebarWidth and arrow keys on its edge, and once at the end of a drag.
func OnSidebarChange(w *gowt.Window, f func(visible bool, width int)) { sidebarOf(w).addChange(f) }

// SetInspectorVisible shows or hides the inspector of a SetInspector window; hidden, the content takes
// its width.
func SetInspectorVisible(w *gowt.Window, on bool) { inspectorOf(w).setVisible(on) }

// InspectorVisible reports whether the inspector of a SetInspector window is shown; false without one.
func InspectorVisible(w *gowt.Window) bool { return inspectorOf(w).visible() }

// SetInspectorWidth sets the inspector width in points, at least 160, and on Windows and Linux at
// least the room of its Inspector toolbar items; a hidden inspector comes back at it.
func SetInspectorWidth(w *gowt.Window, width int) { inspectorOf(w).setWidth(width) }

// InspectorWidth is the inspector width in points, also while it is hidden; 0 without SetInspector.
func InspectorWidth(w *gowt.Window) int { return inspectorOf(w).widthOrZero() }

// OnInspectorChange runs f after the inspector of a SetInspector window was shown, hidden or
// resized: by SetInspectorVisible, SetInspectorWidth and arrow keys on its edge, and once at the end of a drag.
func OnInspectorChange(w *gowt.Window, f func(visible bool, width int)) { inspectorOf(w).addChange(f) }

// SetBottomPanelVisible shows or hides the panel of a SetBottomPanel window; hidden, the content takes
// its height.
func SetBottomPanelVisible(w *gowt.Window, on bool) { bottomOf(w).setVisible(on) }

// BottomPanelVisible reports whether the panel of a SetBottomPanel window is shown; false without one.
func BottomPanelVisible(w *gowt.Window) bool { return bottomOf(w).visible() }

// SetBottomPanelHeight sets the panel height in points, at least 60; a hidden panel comes back at it.
func SetBottomPanelHeight(w *gowt.Window, height int) { bottomOf(w).setWidth(height) }

// BottomPanelHeight is the panel height in points, also while it is hidden; 0 without SetBottomPanel.
func BottomPanelHeight(w *gowt.Window) int { return bottomOf(w).widthOrZero() }

// OnBottomPanelChange runs f after the panel of a SetBottomPanel window was shown, hidden or resized:
// by SetBottomPanelVisible, SetBottomPanelHeight and arrow keys on its edge, and once at the end of a
// drag.
func OnBottomPanelChange(w *gowt.Window, f func(visible bool, height int)) { bottomOf(w).addChange(f) }

// minSidebarWidth is where a drag of the sidebar edge stops without Sidebar toolbar items,
// minInspectorWidth where a drag of the inspector edge stops; minContentWidth is the content such a
// drag leaves at least. They are the points of the doc comments scaled by the density, see gowt.Scaled.
func minSidebarWidth() int32   { return int32(gowt.Scaled(120)) }
func minInspectorWidth() int32 { return int32(gowt.Scaled(160)) }
func minContentWidth() int32   { return int32(gowt.Scaled(200)) }

// minBottomHeight is where a drag of the bottom panel edge stops, minContentHeight the content such a
// drag leaves at least.
func minBottomHeight() int32  { return int32(gowt.Scaled(60)) }
func minContentHeight() int32 { return int32(gowt.Scaled(100)) }

func layoutOf(w *gowt.Window) *paneLayout {
	if lo := w.Unwrap().GetLayout(); lo != nil {
		l, _ := lo.Impl().(*paneLayout)
		return l
	}
	return nil
}

func sidebarOf(w *gowt.Window) *pane {
	if l := layoutOf(w); l != nil {
		return l.side
	}
	return nil
}

func inspectorOf(w *gowt.Window) *pane {
	if l := layoutOf(w); l != nil {
		return l.insp
	}
	return nil
}

func bottomOf(w *gowt.Window) *pane {
	if l := layoutOf(w); l != nil {
		return l.bottom
	}
	return nil
}

// paneLayout lays out a SetSidebar, SetInspector or SetBottomPanel window: sidebar, content over the
// bottom panel, inspector.
type paneLayout struct {
	*swt.Layout
	w                  *gowt.Window
	side, insp, bottom *pane // nil without SetSidebar, SetInspector, SetBottomPanel
	content            *swt.Control
	overlay            bool
	// column is the frame of the content and the bottom panel at the last layout, under the area behind
	// a floating sidebar beside the content.
	column, under [4]int32
	itemsMin      int32 // the width the Sidebar items of the macOS toolbar need, from its last layout
	// appMin is the minimum size the app set, lastMin the one fitMinSize set last, once minSet
	appMin, lastMin [2]int32
	minSet          bool
}

type paneKind int

const (
	sidebarPane paneKind = iota
	inspectorPane
	bottomPane
)

// pane is the sidebar, the inspector or the bottom panel: its composite, the sash on its inner edge and
// its state. The methods take a nil pane, which a window without that pane has.
type pane struct {
	l        *paneLayout
	c        *swt.Composite
	sash     *swt.Sash // the draggable edge between the pane and the content
	kind     paneKind
	width    int32 // the height of the bottom panel
	hidden   bool
	dragging bool
	from     int32 // the width when the drag began
	onChange []func(visible bool, width int)
}

func (l *paneLayout) newPane(c *swt.Composite, width int, kind paneKind) *pane {
	p := &pane{l: l, c: c, width: int32(width), kind: kind}
	dir := int32(swt.VERTICAL)
	if kind == bottomPane {
		dir = swt.HORIZONTAL
	}
	p.sash = swt.NewSash(l.w.Unwrap(), dir|swt.SMOOTH)
	p.sash.MoveAbove(nil)
	for _, t := range []int32{swt.Selection, swt.MouseDown, swt.MouseUp} {
		p.sash.AddListener(t, p)
	}
	return p
}

func (p *pane) setVisible(on bool) {
	if p != nil && p.hidden == on {
		p.hidden = !on
		p.apply()
		p.notify()
	}
}

func (p *pane) visible() bool { return p != nil && !p.hidden }

func (p *pane) setWidth(width int) {
	if p != nil && p.width != max(int32(width), p.minWidth()) {
		p.width = max(int32(width), p.minWidth())
		p.apply()
		p.notify()
	}
}

func (p *pane) widthOrZero() int {
	if p == nil {
		return 0
	}
	return int(p.width)
}

func (p *pane) addChange(f func(visible bool, width int)) {
	if p != nil {
		p.onChange = append(p.onChange, f)
	}
}

// inset is the gap between the pane and the window edges.
func (p *pane) inset() int32 {
	switch p.kind {
	case inspectorPane:
		if p.l != nil && p.l.overlay {
			return sidebarInset()
		}
		return inspectorInset()
	case bottomPane:
		return 0
	}
	return sidebarInset()
}

// floats is true for a shown sidebar inset from the window edges, which the content runs under.
func (p *pane) floats() bool { return p.visible() && p.inset() > 0 }

// room is the width the pane takes from the content: 0 for none or a hidden one.
func (p *pane) room() int32 {
	if p == nil || p.hidden {
		return 0
	}
	return p.inset() + p.width + sidebarDivider()
}

// shown is the width the pane is laid out at, 0 while hidden.
func (p *pane) shown() int32 {
	if p.hidden {
		return 0
	}
	return p.width
}

// apply shows or hides the pane and lays the window out again. A focus inside a hidden pane moves to
// the content, as Finder does.
func (p *pane) apply() {
	s := p.l.w.Unwrap()
	focusIn := false
	for c := s.GetDisplay().GetFocusControl(); c != nil && p.hidden; c = parentControl(c) {
		focusIn = focusIn || c == &p.c.Control
	}
	p.c.SetVisible(!p.hidden)
	if b := bars[s]; b != nil && p.kind == sidebarPane {
		b.sidebarShown(!p.hidden)
	}
	s.Layout()
	if focusIn {
		ct := p.l.content
		if c := firstList(ct.Impl().(interface{ AsComposite() *swt.Composite }).AsComposite()); c != nil {
			c.SetFocus()
		} else {
			ct.SetFocus()
		}
	}
}

func parentControl(c *swt.Control) *swt.Control {
	if p := c.GetParent(); p != nil {
		return &p.Control
	}
	return nil
}

func (p *pane) notify() {
	for _, f := range p.onChange {
		f(!p.hidden, int(p.width))
	}
}

// minWidth is the narrowest pane. For the sidebar that is minSidebarWidth, or the room its Sidebar
// toolbar items need where that is more (the macOS toolbar group up to its inset from the edge, the
// Windows and Linux row); for the inspector minInspectorWidth, or the Windows and Linux strip of its
// Inspector items.
func (p *pane) minWidth() int32 {
	if p.kind == bottomPane {
		return minBottomHeight()
	}
	b := bars[p.l.w.Unwrap()]
	if p.kind == inspectorPane {
		m := minInspectorWidth()
		if b != nil && b.strip != nil {
			m = max(m, b.strip.Unwrap().ComputeSize(swt.DEFAULT, swt.DEFAULT).X)
		}
		return m
	}
	m := max(minSidebarWidth(), p.l.itemsMin)
	if b != nil && b.side != nil {
		m = max(m, b.side.Unwrap().ComputeSize(swt.DEFAULT, swt.DEFAULT).X)
	}
	return m
}

// HandleEvent follows a drag of the sash: it resizes the pane while the mouse moves, stopping at
// minWidth and at minContentWidth of content, and tells the change callbacks once the button is
// released. A drag never hides the pane: a divider that jumps away under the pointer confuses more
// than one that stops.
func (p *pane) HandleEvent(e *swt.Event) {
	switch e.Type {
	case swt.MouseDown:
		p.dragging, p.from = true, p.width
	case swt.MouseUp:
		p.dragging = false
		if p.width != p.from {
			p.notify()
		}
	case swt.Selection:
		r := p.l.w.Unwrap().GetClientArea()
		div := sidebarDivider()
		var want, most int32
		switch p.kind {
		case bottomPane:
			col := p.l.column
			want = bottomHeightAt(e.Y, col[1]+col[3], div)
			most = col[3] - div - minContentHeight()
		case inspectorPane:
			want = inspectorWidthAt(e.X, r.X+r.Width, p.inset(), div)
			most = p.l.paneLimit(p, r.Width)
		default:
			want = sidebarWidthAt(e.X-r.X, p.inset(), div)
			most = p.l.paneLimit(p, r.Width)
		}
		width := dragWidth(want, p.minWidth(), most)
		if width != p.width && !p.hidden {
			p.width = width
			p.apply()
			if !p.dragging { // arrow keys on the focused sash
				p.notify()
			}
		}
		// The sash moves itself to e.X, e.Y; the layout has already put it where the pane ends.
		b := p.sash.GetBounds()
		e.X, e.Y = b.X, b.Y
	}
}

func (l *paneLayout) paneLimit(p *pane, width int32) int32 {
	other := l.insp
	if p.kind == inspectorPane {
		other = l.side
	}
	most := width - other.room() - p.inset() - sidebarDivider()
	if !l.overlay {
		most -= minContentWidth()
	}
	return most
}

func (l *paneLayout) requiredWidth(content int32) int32 {
	room := l.side.room() + l.insp.room()
	if l.overlay {
		return max(content, room)
	}
	return content + room
}

func (l *paneLayout) overlayFrames(client [4]int32, top int32) (content [4]int32, paneHeight int32) {
	content = [4]int32{client[0], client[1] + top, client[2], max(client[3]-top, 0)}
	paneHeight = client[3]
	if l.bottom.visible() {
		paneHeight = max(top, paneHeight-l.bottom.shown()-sidebarDivider())
	}
	return
}

func (l *paneLayout) ComputeSize_(c *swt.Composite, wHint, hHint int32, _ bool) *swt.Point {
	top := int32(topInset(c.GetShell()))
	ct := l.content.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	h := ct.Y + barHeight(c.GetShell())
	if p := l.bottom; p.visible() {
		h += p.width + sidebarDivider()
	}
	for _, p := range []*pane{l.side, l.insp} {
		if p != nil && !p.hidden {
			h = max(h, p.c.ComputeSize(swt.DEFAULT, swt.DEFAULT).Y+p.inset())
		}
	}
	// A packed window starts at least as wide as fitMinSize lets it get.
	pt := swt.NewPoint(l.requiredWidth(max(ct.X, minContentWidth())), top+h)
	if wHint != swt.DEFAULT {
		pt.X = wHint
	}
	if hHint != swt.DEFAULT {
		pt.Y = hHint
	}
	return pt
}

func (l *paneLayout) LayoutFn_(c *swt.Composite, _ bool) {
	r := c.GetClientArea()
	s := c.GetShell()
	l.measure()
	l.fitMin()
	top := int32(topInset(s))
	paneHeight := r.Height
	if l.overlay {
		_, paneHeight = l.overlayFrames([4]int32{r.X, r.Y, r.Width, r.Height}, top)
	}
	ct := [4]int32{r.X, r.Y + top, r.Width, max(r.Height-top, 0)}
	if p := l.side; p != nil {
		bg, sb, rest, sh := sidebarFrames(r.X, r.Y, r.Width, paneHeight, p.shown(), p.inset(), top, sidebarDivider())
		p.c.SetBounds(sb[0], sb[1], sb[2], sb[3])
		p.sash.SetBounds(sh[0], sh[1], sh[2], sh[3])
		p.sash.SetVisible(!p.hidden)
		syncSidebar(s, p.c, bg, p.hidden)
		ct = rest
	}
	var ib [4]int32
	if p := l.insp; p != nil {
		var bg, rest, sh [4]int32
		bg, ib, rest, sh = inspectorFrames(ct[0], r.Y, ct[2], paneHeight, p.shown(), p.inset(), top, sidebarDivider())
		p.sash.SetBounds(sh[0], sh[1], sh[2], sh[3])
		p.sash.SetVisible(!p.hidden)
		syncInspector(p.c, bg, l.overlay)
		ct = rest
	}
	if l.overlay {
		ct, _ = l.overlayFrames([4]int32{r.X, r.Y, r.Width, r.Height}, top)
		l.content.MoveBelow(nil)
	}
	ct, ib = barTop(s, ct, ib)
	if p := l.insp; p != nil {
		p.c.SetBounds(ib[0], ib[1], ib[2], ib[3])
	}
	l.column = ct
	if p := l.bottom; p != nil {
		var pb, sh [4]int32
		ct, pb, sh = bottomFrames(ct[0], ct[1], ct[2], ct[3], p.shown(), sidebarDivider())
		p.c.SetBounds(pb[0], pb[1], pb[2], pb[3])
		p.sash.SetBounds(sh[0], sh[1], sh[2], sh[3])
		p.sash.SetVisible(!p.hidden)
		// The panel's material runs on under a floating sidebar, to the window's left edge.
		var sb *swt.Composite
		if l.side.floats() && !l.overlay {
			pb[2] += pb[0] - r.X
			pb[0] = r.X
			sb = l.side.c
		}
		syncBottom(p.c, pb, sb)
	}
	l.content.SetBounds(ct[0], ct[1], ct[2], ct[3])
	under := [4]int32{}
	if l.side.floats() && !l.overlay {
		under = [4]int32{r.X, ct[1], ct[0] - r.X, ct[3]}
	}
	if under != l.under {
		l.under = under
		s.Redraw()
	}
	l.fitMinSize()
}

// paintUnder fills the area behind a floating sidebar beside the content with the content's
// background, as AppKit extends the content split item beneath the sidebar: without it the window
// colour showed in the inset around the sidebar. A theme switch repaints the window; a later
// SetBackground of the content shows there at the next repaint of the window.
func (l *paneLayout) paintUnder(e *swt.PaintEvent) {
	u := l.under
	bg := l.content.GetBackground()
	if u[2] <= 0 || u[3] <= 0 || bg.GetAlpha() != 255 {
		return
	}
	e.Gc.SetBackground(bg)
	e.Gc.FillRectangle(u[0], u[1], u[2], u[3])
}

// fitMinSize keeps the window as wide as its panes and minContentWidth, and never below the minimum
// the app gave Window.SetMinSize, height included.
func (l *paneLayout) fitMinSize() {
	s := l.w.Unwrap()
	cw := l.requiredWidth(minContentWidth())
	trim := s.GetSize().X - s.GetClientArea().Width
	want := [2]int32{max(cw+trim, l.appMin[0]), l.appMin[1]}
	if l.minSet && want == l.lastMin {
		return
	}
	l.lastMin, l.minSet = want, true
	s.SetMinimumSize(want[0], want[1])
}

// measure takes the room of the Sidebar items from the macOS toolbar's last layout; it reports
// whether there was a measurement.
func (l *paneLayout) measure() bool {
	s := l.w.Unwrap()
	e, ok := itemsMinEdge(s)
	if ok && l.side != nil && !l.side.hidden {
		l.itemsMin = e - s.GetClientArea().X - sidebarInset()
	}
	return ok
}

// fitMin raises a pane narrower than its minWidth to it, so the divider is drawn where a drag starts;
// the change callbacks hear of it after the layout pass, not inside it.
func (l *paneLayout) fitMin() bool {
	raised := false
	for _, p := range []*pane{l.side, l.insp, l.bottom} {
		if p == nil || p.width >= p.minWidth() {
			continue
		}
		p.width = p.minWidth()
		raised = true
		s := l.w.Unwrap()
		s.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
			if !s.IsDisposed() {
				p.notify()
			}
		}))
	}
	return raised
}

// sashGrip is the width of the sash where the pane leaves no gap for it (div 0): it lies over the
// sidebar edge, half on each side, and just inside the inspector edge, clear of the content's scroll bar.
const sashGrip = 6

// sidebarFrames splits the client area (x, y, w, h): the sidebar background inset by in on three
// sides, the sidebar composite on it below the top strip of top points, the content right of the
// sidebar and a gap of div points below the same strip, and the sash in that gap or over the edge.
// Each frame is x, y, width, height. Width 0 is a hidden sidebar: the content starts at x.
func sidebarFrames(x, y, w, h, width, in, top, div int32) (bg, sb, ct, sh [4]int32) {
	bg = [4]int32{x + in, y + in, width, max(h-2*in, 0)}
	sb = [4]int32{x + in, y + top, width, max(h-top-in, 0)}
	cx := x
	if width > 0 {
		cx = x + in + width + div
	}
	ct = [4]int32{cx, y + top, max(w-(cx-x), 0), max(h-top, 0)}
	sw := div
	if div == 0 {
		sw = sashGrip
	}
	sh = [4]int32{max(cx-div/2-sw/2, x), y + top, sw, max(h-top, 0)}
	return bg, sb, ct, sh
}

// sidebarWidthAt is the sidebar width that puts the sash at x, the inverse of sidebarFrames, with x
// relative to the client area.
func sidebarWidthAt(x, in, div int32) int32 {
	sw := div
	if div == 0 {
		sw = sashGrip
	}
	return x - in - div + div/2 + sw/2
}

// inspectorFrames splits the span x, x+w of the client area rows y, y+h, the mirror of sidebarFrames:
// the inspector background inset by in on three sides at the right end, the inspector composite on it
// below the top strip, the content left of it and a gap of div points, and the sash in that gap or
// just inside the inspector edge. Width 0 is a hidden inspector: the content ends at x+w.
func inspectorFrames(x, y, w, h, width, in, top, div int32) (bg, ib, ct, sh [4]int32) {
	ix := x + w - in - width
	bg = [4]int32{ix, y + in, width, max(h-2*in, 0)}
	ib = [4]int32{ix, y + top, width, max(h-top-in, 0)}
	cw := w
	if width > 0 {
		cw = w - in - width - div
	}
	ct = [4]int32{x, y + top, max(cw, 0), max(h-top, 0)}
	sw := div
	if div == 0 {
		sw = sashGrip
	}
	sh = [4]int32{min(ix-div, x+w-sw), y + top, sw, max(h-top, 0)}
	return bg, ib, ct, sh
}

// inspectorWidthAt is the inspector width that puts the sash at x, the inverse of inspectorFrames,
// with right the right edge of the span (x+w there) in the same coordinates as x.
func inspectorWidthAt(x, right, in, div int32) int32 { return right - in - div - x }

// bottomFrames splits the content column (x, y, w, h) into the content above, the bottom panel height
// points high at its foot and the sash in the gap of div points between them, or just inside the
// panel's top edge for div 0, clear of the content's horizontal scroll bar. Height 0 is a hidden panel:
// the content keeps the column.
func bottomFrames(x, y, w, h, height, div int32) (ct, pb, sh [4]int32) {
	py := y + h - height
	pb = [4]int32{x, py, w, height}
	ct = [4]int32{x, y, w, h}
	if height > 0 {
		ct[3] = max(h-height-div, 0)
	}
	sw := div
	if div == 0 {
		sw = sashGrip
	}
	sh = [4]int32{x, max(py-div, y), w, sw}
	return ct, pb, sh
}

// bottomHeightAt is the panel height that puts the sash at y, the inverse of bottomFrames, with bottom
// the foot of the column in the same coordinates as y.
func bottomHeightAt(y, bottom, div int32) int32 { return bottom - div - y }

// dragWidth is the width a drag to want gives, kept from least to most (or least, if most is smaller).
func dragWidth(want, least, most int32) int32 { return max(min(want, most), least) }
