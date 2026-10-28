package gowt

import (
	"fmt"
	"slices"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// Widget is any gowt control. Layouts and containers take it where they refer to a sibling or a
// child. Its methods are the ones every widget type has.
type Widget interface {
	control() *swt.Control
	SetEnabled(v bool)
	Enabled() bool
	SetVisible(v bool)
	Visible() bool
	SetTooltip(s string)
	SetFont(f Font)
	SetForeground(c Paint)
	SetBackground(c Paint)
	Focus() bool
	Dispose()
	OnDispose(f func())
}

// base holds the methods every control has. Each widget type embeds it, so go doc lists them on
// the type itself.
type base struct{ ctl *swt.Control }

func (w *base) control() *swt.Control { return w.ctl }

// SetEnabled enables or disables the control, a Window or container included (its children follow).
// A disabled control ignores input and is drawn dimmed; Disabled() does the same at creation.
func (w *base) SetEnabled(v bool) { w.ctl.SetEnabled(v) }

// Enabled reports the state set by SetEnabled, not that of the parents.
func (w *base) Enabled() bool { return w.ctl.GetEnabled() }

// SetVisible shows or hides the control. In a Grid or Row layout a hidden control also gives up its
// cell, and the parent is laid out again so the neighbours take the space; SetVisible(true) clears
// that exclusion. Other layouts keep the gap.
func (w *base) SetVisible(v bool) {
	c := w.ctl
	c.SetVisible(v)
	p := c.GetParent()
	if p == nil {
		return
	}
	switch d := c.GetLayoutData().(type) {
	case *swt.GridData:
		d.Exclude = !v
	case *swt.RowData:
		d.Exclude = !v
	case nil:
		if l := p.GetLayout(); l != nil {
			switch l.Impl().(type) {
			case swt.GridLayoutLike:
				g := swt.NewGridData()
				g.Exclude = !v
				c.SetLayoutData(g)
			case *swt.RowLayout:
				r := swt.NewRowData()
				r.Exclude = !v
				c.SetLayoutData(r)
			}
		}
	}
	flushUp(&p.Control)
	p.LayoutOverload1(true)
}

// Visible reports the state set by SetVisible, not that of the parents.
func (w *base) Visible() bool { return w.ctl.GetVisible() }

// SetTooltip sets the hover text; "" removes it. Tooltip() does the same at creation.
func (w *base) SetTooltip(s string) { w.ctl.SetToolTipText(s) }

// SetFont gives the control the font f. Empty Name and zero Size keep the control's current ones (the
// system font for a default control); a zero Size with a Scale is the system font size times Scale.
// Controls given equal fonts share one native font, released after the last of them is disposed or
// given another font. Typeface() does the same at creation. Lays the parents out when the size
// changes.
func (w *base) SetFont(f Font) { setFont(w.ctl, f, sizing) }

// setFont gives c the font f; run decides whether the change lays the parents out.
func setFont(c *swt.Control, f Font, run func(*swt.Control, func())) {
	a := appOf(c)
	k := f.from(c.GetFont(), c.GetDisplay())
	run(c, func() { c.SetFont(a.font(k)) })
	old, had := ownedFonts[c]
	ownedFonts[c] = k
	if had {
		a.releaseFont(old)
	} else {
		c.AddDisposeListener(&fontRelease{a, c})
	}
}

// sizing runs set; when that changes c's preferred size, c's parents are laid out on the next
// event-loop turn or at App.Flush, one pass for all the controls changed in that turn.
func sizing(c *swt.Control, set func()) {
	a := appOf(c)
	if a == nil || &c.GetShell().Control == c { // no layout places a Window
		set()
		return
	}
	before := preferred(c)
	set()
	after := preferred(c)
	if after.X != before.X || after.Y != before.Y {
		a.relayout(c)
	}
}

// relayout queues the layout of c's parents for the next event-loop turn, once for each control.
func (a *App) relayout(c *swt.Control) {
	// Linear: a turn changes few controls; thousands of them in one turn would want a set.
	if slices.Contains(a.relayouts, c) {
		return
	}
	a.relayouts = append(a.relayouts, c)
	if len(a.relayouts) == 1 {
		a.display.AsyncExec(jrt.NewRunnable(a.live(a.runRelayouts)))
	}
}

// preferred is the size c asks its layout for; a WRAP control is measured at its current width,
// as GridLayout does for a filled cell.
func preferred(c *swt.Control) *swt.Point {
	w := int32(swt.DEFAULT)
	if x := c.GetSize().X; c.GetStyle()&swt.WRAP != 0 && x > 0 {
		w = x
	}
	return c.ComputeSizeWHintHHintChanged(w, swt.DEFAULT, false)
}

// runRelayouts lays out the parent chains of the controls sizing queued, one pass per shell.
func (a *App) runRelayouts() {
	list := a.relayouts
	a.relayouts = nil
	for i, c := range list {
		if c == nil || c.IsDisposed() {
			continue
		}
		refitMax(c)
		sh := c.GetShell()
		group := []*swt.Control{c}
		for j, d := range list[i+1:] {
			if d != nil && !d.IsDisposed() && d.GetShell() == sh {
				refitMax(d)
				group = append(group, d)
				list[i+1+j] = nil
			}
		}
		sh.LayoutOverload4(group, swt.NONE)
	}
}

// ownedFonts holds the font key SetFont gave each control; UI thread only.
var ownedFonts = map[*swt.Control]Font{}

type fontRelease struct {
	a *App
	c *swt.Control
}

func (r *fontRelease) WidgetDisposed(*swt.DisposeEvent) {
	r.a.releaseFont(ownedFonts[r.c])
	delete(ownedFonts, r.c)
}

// sharedFont is a native font and the number of its users: controls given it by SetFont, and canvases
// and styled texts that drew with it.
type sharedFont struct {
	f    *swt.Font
	refs int
}

// font returns the native font for k, a Font filled by from, making it on first use. Every call
// needs its releaseFont.
func (a *App) font(k Font) *swt.Font {
	s := a.fonts[k]
	if s == nil {
		s = &sharedFont{f: k.swtFont(a.display)}
		a.fonts[k] = s
		if k.Px > 0 {
			a.pxFonts[s.f] = k.Px
		}
	}
	s.refs++
	return s.f
}

// releaseFont drops one use of k and disposes the native font after the last one.
func (a *App) releaseFont(k Font) {
	s := a.fonts[k]
	if s == nil {
		return
	}
	if s.refs--; s.refs == 0 {
		delete(a.pxFonts, s.f)
		s.f.Dispose()
		delete(a.fonts, k)
	}
}

// SystemFont returns the family and size of the system UI font, which Font.Scale multiplies.
func (a *App) SystemFont() Font {
	return Font{}.from(a.display.GetSystemFont(), a.display)
}

// SetForeground sets the text colour of the control; an Adaptive one and SecondaryText follow the
// system theme, nil restores the default. Foreground() does the same at creation.
func (w *base) SetForeground(c Paint) { setPaint(w.ctl, c, true) }

// SetBackground sets the background colour of the control; an Adaptive one follows the system theme,
// nil restores the default. On a container (Window, VBox and the other boxes, Group, Pages, Section,
// Scroll, Split) the children without a background of their own, such as labels and nested boxes,
// show it too; text fields, lists, tables and buttons keep theirs. Background() does the same at creation.
func (w *base) SetBackground(c Paint) { setPaint(w.ctl, c, false) }

// Focus gives the control the keyboard focus and reports whether it took it.
func (w *base) Focus() bool { return w.ctl.SetFocus() }

// Dispose removes the control and its children from the window and lays the parent out again, so
// the neighbours take the space. The widget must not be used afterwards; a second Dispose does nothing.
func (w *base) Dispose() {
	if w.ctl.IsDisposed() {
		return
	}
	p := w.ctl.GetParent()
	w.ctl.Dispose()
	if p != nil && !p.IsDisposed() {
		flushUp(&p.Control)
		p.LayoutOverload1(true)
	}
}

// OnDispose runs f when the control is disposed: by Dispose, with its parent, or when its window closes.
func (w *base) OnDispose(f func()) { w.ctl.AddDisposeListener(disposeFunc(f)) }

type disposeFunc func()

func (f disposeFunc) WidgetDisposed(*swt.DisposeEvent) { f() }

// Style options. Each applies to the widgets whose "Options:" line lists it; elsewhere it is
// reported to App.OnError. Check (gowt.go) also makes Table and Tree rows checkable.

// MultiSelect allows several selected items in a List, Table or Tree.
func MultiSelect() Option { return style{id: sMultiSelect, set: swt.MULTI} }

// Simple makes a Combo show its list permanently.
func Simple() Option { return style{id: sSimple, set: swt.SIMPLE} }

// Vertical rotates a Scale, Slider or Progress, stacks a Split, and makes a Sash a vertical bar.
func Vertical() Option { return style{id: sVertical, set: swt.VERTICAL} }

// Smooth makes a Progress an unsegmented bar.
func Smooth() Option { return style{id: sSmooth, set: swt.SMOOTH} }

// Indeterminate animates a Progress without a value.
func Indeterminate() Option { return style{id: sIndeterminate, set: swt.INDETERMINATE} }

// Flat gives a ToolBar the flat look.
func Flat() Option { return style{id: sFlat, set: swt.FLAT} }

// Borderless draws a push or Toggle Button without its frame, for an icon in a section header (+, -,
// an eye): macOS no bezel, a toggle with its symbol in the accent colour while on; GTK "flat";
// Windows the WinUI Subtle button.
func Borderless() Option {
	return hook(func(w Widget) error {
		b, ok := pushButton(w)
		if v, isButton := w.(*ButtonView); isButton && v.b.GetStyle()&swt.TOGGLE != 0 {
			b, ok = v.b, true
		}
		if !ok {
			return fmt.Errorf("gowt: Borderless applies to a push or Toggle Button, not to %s", nameOf(w.control()))
		}
		setButtonBorderless(b)
		return nil
	})
}

// Circle makes a push Button round: give it a square Size. With look.GlassButton on macOS 26 it is a
// glass circle, in either order. GTK: "circular". Windows keeps the rectangle.
func Circle() Option {
	return hook(func(w Widget) error {
		b, ok := pushButton(w)
		if !ok {
			return fmt.Errorf("gowt: Circle applies to a push Button, not to %s", nameOf(w.control()))
		}
		setButtonCircle(b)
		return nil
	})
}

// AccessibleLabel names a widget for screen readers, which an icon-only Button has no text for.
// macOS: accessibilityLabel; GTK: the ATK name; Windows has no accessibility in the toolkit.
func AccessibleLabel(s string) Option {
	return hook(func(w Widget) error {
		setAccessibleLabel(w.control(), s)
		return nil
	})
}

// ID names a widget, a Panel or a Window for the QA driver, which picks controls by it. It is the
// same in every build and has no effect on the control. In an Each, take it from the row's data.
func ID(id string) Option {
	return apply(func(c *swt.Control) { c.SetDataKeyValue("gowt.id", id) })
}

func pushButton(w Widget) (*swt.Button, bool) {
	b, ok := w.(*ButtonView)
	if !ok || b.b.GetStyle()&(swt.CHECK|swt.RADIO|swt.TOGGLE|swt.ARROW) != 0 {
		return nil, false
	}
	return b.b, true
}

// Wrap breaks long text of a Label or StyledText into lines, and lets a ToolBar wrap its items to further rows.
func Wrap() Option { return style{id: sWrap, set: swt.WRAP} }

// Bottom puts the tabs of Tabs or CTabs below the pages.
func Bottom() Option { return style{id: sBottom, set: swt.BOTTOM} }

// Closable adds a close button to a CTabs tab.
func Closable() Option { return style{id: sClosable, set: swt.CLOSE} }

// DropDown makes a DateTime date field a drop-down calendar, or a ToolBar item a drop-down.
func DropDown() Option { return style{id: sDropDown, set: swt.DROP_DOWN} }

// Right right-aligns a table Column.
func Right() Option { return style{id: sRight, set: swt.RIGHT} }

// Center centers a table Column.
func Center() Option { return style{id: sCenter, set: swt.CENTER} }

// AsTime and AsCalendar turn a DateTime from a date field into a time field or a month calendar.
func AsTime() Option { return style{id: sAsTime, set: swt.TIME} }

// AsCalendar makes a DateTime a month calendar.
func AsCalendar() Option { return style{id: sAsCalendar, set: swt.CALENDAR} }

type selector interface {
	AddSelectionListener(swt.SelectionListener)
}

func onSelect(w selector, f func(*swt.SelectionEvent)) {
	w.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(f))
}

func onActivate(w selector, f func(*swt.SelectionEvent)) {
	w.AddSelectionListener(swt.SelectionListenerWidgetDefaultSelectedAdapter(f))
}

func toInts(a []int32) []int {
	r := make([]int, len(a))
	for i, v := range a {
		r[i] = int(v)
	}
	return r
}

func toInt32s(a []int) []int32 {
	r := make([]int32, len(a))
	for i, v := range a {
		r[i] = int32(v)
	}
	return r
}

// SetImage sets the icon; the label does not own img. Lays the parents out when the size changes.
func (l *LabelView) SetImage(img *Image) { sizing(&l.l.Control, func() { l.l.SetImage(img.i) }) }

// SetImage sets the icon; the button does not own img. Lays the parents out when the size changes.
func (b *ButtonView) SetImage(img *Image) { sizing(&b.b.Control, func() { b.b.SetImage(img.i) }) }
