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
	// OnSearch makes the item a search field (NSSearchToolbarItem) with Title as the placeholder. It
	// gets the text after a typing pause, on Return and on clearing.
	OnSearch func(text string)
	// Enabled, when set, enables and disables a push item or a Group segment as it changes, for as
	// long as the toolbar stays; nil keeps the item enabled.
	Enabled rx.Reader[bool]
}

// SetToolbar puts a native unified toolbar (NSToolbar) on the window, replacing an earlier one; with
// no items it removes the toolbar. On macOS 26 the items are Liquid Glass. It reports whether the
// toolbar exists: elsewhere nothing is created and the result is false, so build your own gowt.ToolBar
// in that case. Use it together with SetFullSizeContent only if the content leaves room for the bar.
func SetToolbar(w *gowt.Window, items ...ToolbarItem) bool {
	ok := setToolbar(w.Unwrap(), items)
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
// shows a flush sidebar material; Windows and Linux put two plain panels side by side.
func SetSidebar(w *gowt.Window, sidebar, content Container, width int) {
	SetFullSizeContent(w, true)
	sb := sidebar.AsComposite()
	l := &sidebarLayout{Layout: swt.NewLayout(), sidebar: sb, content: &content.AsComposite().Control, width: int32(width)}
	l.SetImpl_(l)
	s := w.Unwrap()
	s.SetLayout(l)
	setSidebarChrome(s, sb)
	s.Layout()
}

type sidebarLayout struct {
	*swt.Layout
	sidebar *swt.Composite
	content *swt.Control
	width   int32
}

func (l *sidebarLayout) ComputeSize_(c *swt.Composite, wHint, hHint int32, _ bool) *swt.Point {
	in, top := sidebarInset(), int32(topInset(c.GetShell()))
	sb := l.sidebar.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	ct := l.content.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	p := swt.NewPoint(in+l.width+ct.X, top+max(sb.Y+in, ct.Y))
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
	bg, sb, ct := sidebarFrames(r.X, r.Y, r.Width, r.Height, l.width, sidebarInset(), int32(topInset(c.GetShell())))
	l.sidebar.SetBounds(sb[0], sb[1], sb[2], sb[3])
	l.content.SetBounds(ct[0], ct[1], ct[2], ct[3])
	syncSidebar(c.GetShell(), l.sidebar, bg)
}

// sidebarFrames splits the client area (x, y, w, h): the sidebar background inset by in on three
// sides, the sidebar composite on it below the top strip of top points, and the content right of the
// sidebar below the same strip. Each frame is x, y, width, height.
func sidebarFrames(x, y, w, h, width, in, top int32) (bg, sb, ct [4]int32) {
	bg = [4]int32{x + in, y + in, width, max(h-2*in, 0)}
	sb = [4]int32{x + in, y + top, width, max(h-top-in, 0)}
	cx := x + in + width
	ct = [4]int32{cx, y + top, max(w-(cx-x), 0), max(h-top, 0)}
	return bg, sb, ct
}
