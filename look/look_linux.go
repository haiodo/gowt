package look

import (
	"fmt"

	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

func setBackdrop(*swt.Shell, Backdrop) {}

func setRoundedCorners(*swt.Shell, bool) {}

func topInset(*swt.Shell) int { return 0 }

func classic() {}

func setFullSizeContent(*swt.Shell, bool) {}

func setDarkContent(*swt.Display, bool) {}

func setToolbar(*swt.Shell, []ToolbarItem) bool { return false }

func setGlassGroup(*swt.Composite, float64) {}

func sidebarInset() int32 { return 0 }

func setSidebarChrome(*swt.Shell, *swt.Composite, *swt.Control) {}

func syncSidebar(*swt.Shell, *swt.Composite, [4]int32, bool) {}

func inspectorInset() int32 { return 0 }

func syncInspector(*swt.Composite, [4]int32) {}

// sidebarDivider leaves a gap for the sash: an SWT sash here paints its own background and would cover
// the content edge.
func sidebarDivider() int32 { return 4 }

// itemsMinEdge has no native toolbar to measure; the side row of Sidebar items is measured instead.
func itemsMinEdge(*swt.Shell) (int32, bool) { return 0, false }

// startUnfocused does nothing: SWT keeps its choice, the first control.
func startUnfocused(*swt.Shell) {}

// GTK has no glass: a capsule is a rounded surface drawn by GTK CSS. The entries inside are made
// transparent by a descendant rule, so children added later are covered too.
const glassCSS = `
.gowt-glass {
	background-color: @theme_base_color;
	background-image: none;
	border: 1px solid transparent;
	border-radius: 12px;
}
.gowt-glass.gowt-glass-border { border-color: @borders; }
.gowt-glass.gowt-glass-ring { border-color: @theme_selected_bg_color; box-shadow: inset 0 0 0 1px @theme_selected_bg_color; }
.gowt-glass entry, .gowt-glass entry:focus, .gowt-glass textview, .gowt-glass textview text,
.gowt-glass scrolledwindow, .gowt-glass spinbutton {
	background-color: transparent;
	background-image: none;
	border: none;
	border-image: none;
	box-shadow: none;
	outline: none;
}
.gowt-glass entry { padding-left: 10px; padding-right: 10px; }
.gowt-glass textview text { padding: 0 10px; }
button.gowt-glass-button {
	background-image: none;
	background-color: alpha(currentColor, 0.06);
	border-radius: 9999px;
	box-shadow: none;
}
button.gowt-glass-button:hover { background-color: alpha(currentColor, 0.12); }
button.gowt-glass-button.gowt-glass-border { border: 1px solid @borders; }
`

var glassLoaded bool

func loadGlassCSS() {
	if !glassLoaded {
		glassLoaded = true
		gtk.AddGlobalCSS(glassCSS)
	}
}

const (
	glassFocusKey  = "gowt.glass.focus"
	glassRadiusKey = "gowt.glass.radius"
	glassFrameKey  = "gowt.glass.frame"
)

// glassFrame draws the CSS border and ring: the SWT composite paints the CSS background only.
type glassFrame struct{ c *swt.Composite }

func (f *glassFrame) HandleEvent(e *swt.Event) {
	if !gtk.HasStyleClass(f.c.Handle, "gowt-glass") {
		return
	}
	s := f.c.GetSize()
	gtk.RenderFrame(f.c.Handle, e.Gc.GetGCData().Cairo, float64(s.X), float64(s.Y))
}

func setGlass(c *swt.Composite, on bool) {
	if on {
		loadGlassCSS()
		gtk.AddStyleClass(c.Handle, "gowt-glass")
		if c.GetDataKey(glassFrameKey) == nil {
			f := &glassFrame{c}
			c.SetDataKeyValue(glassFrameKey, f)
			c.AddListener(swt.Paint, f)
		}
	} else {
		gtk.RemoveStyleClass(c.Handle, "gowt-glass")
	}
	c.Redraw()
}

// setGlassRadius: Capsule is a radius far above half the side, which CSS clamps to a half circle
// and keeps on resize.
func setGlassRadius(c *swt.Composite, r float64) {
	loadGlassCSS()
	px := 9999.0
	if r != Capsule {
		px = r
	}
	css := fmt.Sprintf(".gowt-glass { border-radius: %gpx; }", px)
	p, _ := c.GetDataKey(glassRadiusKey).(int64)
	c.SetDataKeyValue(glassRadiusKey, gtk.SetWidgetCSS(c.Handle, css, p))
}

func glassButton(c *swt.Control) {
	loadGlassCSS()
	gtk.AddStyleClass(c.Handle, "gowt-glass-button")
}

func setGlassBorder(c *swt.Control, on bool) {
	loadGlassCSS()
	if on {
		gtk.AddStyleClass(c.Handle, "gowt-glass-border")
	} else {
		gtk.RemoveStyleClass(c.Handle, "gowt-glass-border")
	}
}

// setGlassFocusRing follows the focus control through a display filter: a later child is covered.
func setGlassFocusRing(c *swt.Control, on bool) {
	f, _ := c.GetDataKey(glassFocusKey).(*focusRing)
	switch {
	case on && f == nil:
		loadGlassCSS()
		f = &focusRing{c: c}
		c.SetDataKeyValue(glassFocusKey, f)
		d := c.GetDisplay()
		d.AddFilter(swt.FocusIn, f)
		d.AddFilter(swt.FocusOut, f)
		c.AddListener(swt.Dispose, f)
	case !on && f != nil:
		f.detach()
		c.SetDataKeyValue(glassFocusKey, nil)
		gtk.RemoveStyleClass(c.Handle, "gowt-glass-ring")
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
	if f.c.IsDisposed() {
		return
	}
	if focused {
		gtk.AddStyleClass(f.c.Handle, "gowt-glass-ring")
	} else {
		gtk.RemoveStyleClass(f.c.Handle, "gowt-glass-ring")
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
