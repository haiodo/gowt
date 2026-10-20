// Package look sets the platform look of a gowt app: the window material, macOS 26 Liquid Glass,
// Windows 11 corners and dark content. Every call does nothing on a system without the feature.
package look

import (
	"github.com/haiodo/gowt"
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
// container and its children stop painting their own background. It needs macOS 26; older macOS
// shows a blurred material, other systems nothing.
func SetGlass(c Container, on bool) { setGlass(c.AsComposite(), on) }

// GlassButton gives a push button (not Check, Radio, Toggle or Arrow) the Liquid Glass bezel on macOS 26 and newer; elsewhere the
// button is unchanged.
func GlassButton() gowt.Option { return gowt.Custom(glassButton) }

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
	// OnSearch makes the item a search field (NSSearchToolbarItem) with Title as the placeholder. It
	// gets the text after a typing pause, on Return and on clearing.
	OnSearch func(text string)
	// Enabled, when set, enables and disables a push item or a Group segment as it changes, for as
	// long as the toolbar stays; nil keeps the item enabled.
	Enabled rx.Reader[bool]
}

// SetToolbar puts a native unified toolbar (NSToolbar) on the window, replacing an earlier one; with
// no items it removes the toolbar. On macOS 26 the items are Liquid Glass. It reports whether the
// native toolbar exists. Where it does not (Windows, Linux), a window laid out by SetSidebar, before
// or after this call, gets the items as a bar of gowt widgets at the top of its content column: flat
// icon buttons, the window title, a search field, in the content's colour. A window without
// SetSidebar gets nothing there; build your own gowt.ToolBar in that case. Use it together with
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

// SetGlassGroup makes the glass surfaces (SetGlass) of the direct children of c share one
// NSGlassEffectContainerView, so surfaces closer than spacing points merge into one shape. Call it
// before the children get their glass. macOS 26 only; elsewhere it does nothing.
func SetGlassGroup(c Container, spacing float64) { setGlassGroup(c.AsComposite(), spacing) }

// SetSidebar lays w out the way Finder and System Settings are: sidebar on the left, width points wide,
// content filling the rest. It replaces the window layout, so w must hold just these two. On macOS 26
// the sidebar floats: Liquid Glass with rounded corners, inset from the window edges and running under
// the traffic lights; the window gets full-size content, sidebar children and content start below the
// title bar and toolbar, and the items and title of SetToolbar start past the sidebar. Older macOS
// shows a flush sidebar material; Windows and Linux put two plain panels side by side, with the
// SetToolbar items as a bar above the content. A background on w itself paints over the macOS
// sidebar glass; colour the content instead.
//
// The user resizes the sidebar by dragging its edge; dragged narrower than 120 points, or than its
// Sidebar toolbar items need, it hides. Read and change that with SidebarVisible, SetSidebarVisible,
// SidebarWidth and SetSidebarWidth, and keep it between runs with OnSidebarChange.
func SetSidebar(w *gowt.Window, sidebar, content Container, width int) {
	SetFullSizeContent(w, true)
	sb := sidebar.AsComposite()
	s := w.Unwrap()
	if old := sidebarOf(w); old != nil {
		old.sash.Dispose()
	}
	l := &sidebarLayout{Layout: swt.NewLayout(), w: w, sidebar: sb, content: &content.AsComposite().Control, width: int32(width)}
	l.SetImpl_(l)
	l.sash = swt.NewSash(s, swt.VERTICAL|swt.SMOOTH)
	l.sash.MoveAbove(nil)
	for _, t := range []int32{swt.Selection, swt.MouseDown, swt.MouseUp} {
		l.sash.AddListener(t, l)
	}
	s.SetLayout(l)
	setSidebarChrome(s, sb, &l.sash.Control)
	b := barOf(w)
	b.content, b.sidebar = l.content, sidebar
	b.build()
	// Without gowt.Focused the window opens with the focus on its first list, as Finder does, not on a
	// button of the sidebar.
	s.SetDataKeyValue(focusDefaultKey, func() *swt.Control {
		if c := firstList(content.AsComposite()); c != nil {
			return c
		}
		return firstList(sb)
	})
}

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

// SetSidebarVisible shows or hides the sidebar of a SetSidebar window; hidden, the content takes the
// whole width and the Sidebar toolbar items move to the start of the toolbar.
func SetSidebarVisible(w *gowt.Window, on bool) {
	if l := sidebarOf(w); l != nil && l.hidden == on {
		l.hidden = !on
		l.apply()
		l.notify()
	}
}

// SidebarVisible reports whether the sidebar of a SetSidebar window is shown; false without one.
func SidebarVisible(w *gowt.Window) bool {
	l := sidebarOf(w)
	return l != nil && !l.hidden
}

// SetSidebarWidth sets the sidebar width in points; a hidden sidebar comes back at it. The width is at
// least 120 and at least the room of the Sidebar toolbar items, as far as the window knows it yet.
func SetSidebarWidth(w *gowt.Window, width int) {
	if l := sidebarOf(w); l != nil && l.width != max(int32(width), l.minWidth()) {
		l.width = max(int32(width), l.minWidth())
		l.apply()
		l.notify()
	}
}

// SidebarWidth is the sidebar width in points, also while it is hidden; 0 without SetSidebar.
func SidebarWidth(w *gowt.Window) int {
	if l := sidebarOf(w); l != nil {
		return int(l.width)
	}
	return 0
}

// OnSidebarChange runs f after the sidebar of a SetSidebar window was shown, hidden or resized: by
// SetSidebarVisible, SetSidebarWidth and arrow keys on its edge, and once at the end of a drag.
func OnSidebarChange(w *gowt.Window, f func(visible bool, width int)) {
	if l := sidebarOf(w); l != nil {
		l.onChange = append(l.onChange, f)
	}
}

// minSidebarWidth is the narrowest sidebar a drag leaves without Sidebar toolbar items; a drag below
// it hides the sidebar.
// minContentWidth is the content a drag of the sidebar edge leaves at least.
const (
	minSidebarWidth = 120
	minContentWidth = 200
)

func sidebarOf(w *gowt.Window) *sidebarLayout {
	if lo := w.Unwrap().GetLayout(); lo != nil {
		l, _ := lo.Impl().(*sidebarLayout)
		return l
	}
	return nil
}

type sidebarLayout struct {
	*swt.Layout
	w        *gowt.Window
	sidebar  *swt.Composite
	content  *swt.Control
	sash     *swt.Sash // the draggable edge between sidebar and content
	width    int32
	itemsMin int32 // the width the Sidebar items of the macOS toolbar need, from its last layout
	hidden   bool
	dragging bool
	// from is the visible state and width when the drag began
	from     [2]int32
	onChange []func(visible bool, width int)
}

// apply shows or hides the sidebar and lays the window out again. A focus inside a hidden sidebar
// moves to the content, as Finder does.
func (l *sidebarLayout) apply() {
	s := l.w.Unwrap()
	focusIn := false
	for c := s.GetDisplay().GetFocusControl(); c != nil && l.hidden; c = parentControl(c) {
		focusIn = focusIn || c == &l.sidebar.Control
	}
	l.sidebar.SetVisible(!l.hidden)
	if b := bars[s]; b != nil {
		b.sidebarShown(!l.hidden)
	}
	s.Layout()
	if focusIn {
		if c := firstList(l.content.Impl().(interface{ AsComposite() *swt.Composite }).AsComposite()); c != nil {
			c.SetFocus()
		} else {
			l.content.SetFocus()
		}
	}
}

func parentControl(c *swt.Control) *swt.Control {
	if p := c.GetParent(); p != nil {
		return &p.Control
	}
	return nil
}

func (l *sidebarLayout) notify() {
	for _, f := range l.onChange {
		f(!l.hidden, int(l.width))
	}
}

func (l *sidebarLayout) state() [2]int32 {
	if l.hidden {
		return [2]int32{0, l.width}
	}
	return [2]int32{1, l.width}
}

// minWidth is the narrowest sidebar: minSidebarWidth, or the room its Sidebar toolbar items need where
// that is more (the macOS toolbar group up to its inset from the edge, the Windows and Linux row).
func (l *sidebarLayout) minWidth() int32 {
	m := max(int32(minSidebarWidth), l.itemsMin)
	if b := bars[l.w.Unwrap()]; b != nil && b.side != nil {
		m = max(m, b.side.Unwrap().ComputeSize(swt.DEFAULT, swt.DEFAULT).X)
	}
	return m
}

// HandleEvent follows a drag of the sash: it resizes the sidebar while the mouse moves, hides it
// below minWidth, and tells OnSidebarChange once the button is released.
func (l *sidebarLayout) HandleEvent(e *swt.Event) {
	switch e.Type {
	case swt.MouseDown:
		l.dragging, l.from = true, l.state()
	case swt.MouseUp:
		l.dragging = false
		l.w.Unwrap().Layout()
		if l.state() != l.from {
			l.notify()
		}
	case swt.Selection:
		r := l.w.Unwrap().GetClientArea()
		in, div := sidebarInset(), sidebarDivider()
		width, ok := dragWidth(sidebarWidthAt(e.X-r.X, in, div), l.minWidth(), r.Width-in-div-minContentWidth)
		if ok != !l.hidden || (ok && width != l.width) {
			l.hidden = !ok
			if ok {
				l.width = width
			}
			l.apply()
			if !l.dragging { // arrow keys on the focused sash
				l.notify()
			}
		}
		// The sash moves itself to e.X, e.Y; the layout has already put it where the sidebar ends.
		b := l.sash.GetBounds()
		e.X, e.Y = b.X, b.Y
	}
}

func (l *sidebarLayout) ComputeSize_(c *swt.Composite, wHint, hHint int32, _ bool) *swt.Point {
	in, top := sidebarInset(), int32(topInset(c.GetShell()))
	sb := l.sidebar.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	ct := l.content.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	side := in + l.width + sidebarDivider()
	if l.hidden {
		side, sb.Y = 0, 0
	}
	p := swt.NewPoint(side+ct.X, top+max(sb.Y+in, ct.Y+barHeight(c.GetShell())))
	if wHint != swt.DEFAULT {
		p.X = wHint
	}
	if hHint != swt.DEFAULT {
		p.Y = hHint
	}
	return p
}

func (l *sidebarLayout) LayoutFn_(c *swt.Composite, _ bool) {
	r := c.GetClientArea()
	width := l.width
	if l.hidden {
		width = 0
	}
	bg, sb, ct, sh := sidebarFrames(r.X, r.Y, r.Width, r.Height, width, sidebarInset(), int32(topInset(c.GetShell())), sidebarDivider())
	l.sidebar.SetBounds(sb[0], sb[1], sb[2], sb[3])
	l.sash.SetBounds(sh[0], sh[1], sh[2], sh[3])
	// A hidden sidebar has no edge to drag, except during the drag that hid it.
	l.sash.SetVisible(!l.hidden || l.dragging)
	ct = barTop(c.GetShell(), ct)
	l.content.SetBounds(ct[0], ct[1], ct[2], ct[3])
	syncSidebar(c.GetShell(), l.sidebar, bg, l.hidden)
	if e, ok := itemsMinEdge(c.GetShell()); ok && !l.hidden {
		l.itemsMin = e - bg[0]
	}
}

// sashGrip is the width of the sash where the sidebar leaves no gap for it (div 0): it lies over the
// sidebar edge, half on each side.
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

// dragWidth is the width a drag to want gives, kept from least to most (or least, if most is smaller);
// false hides the sidebar, for a want below least.
func dragWidth(want, least, most int32) (int32, bool) {
	if want < least {
		return 0, false
	}
	return max(min(want, most), least), true
}
