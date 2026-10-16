package gowt

import (
	"github.com/haiodo/gowt/rx"
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
	SetForeground(c RGB)
	SetBackground(c RGB)
	Focus() bool
	Dispose()
	OnDispose(f func())
	BindVisible(r rx.Reader[bool])
	BindEnabled(r rx.Reader[bool])
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
// that exclusion. Other layouts keep the gap. A layout option applied to the control after
// SetVisible(false) (Cell, InRow) replaces its layout data and so undoes the exclusion.
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
			case *swt.GridLayout:
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
	p.LayoutOverload1(true)
}

// Visible reports the state set by SetVisible, not that of the parents.
func (w *base) Visible() bool { return w.ctl.GetVisible() }

// SetTooltip sets the hover text; "" removes it. Tooltip() does the same at creation.
func (w *base) SetTooltip(s string) { w.ctl.SetToolTipText(s) }

// SetFont gives the control the font f. Empty Name and zero Size keep the control's current ones (the system font for a default control).
// The control owns the font and releases it when replaced by the next SetFont or when it is disposed.
func (w *base) SetFont(f Font) {
	c := w.ctl
	if cur := c.GetFont().GetFontData(); len(cur) > 0 {
		if f.Name == "" {
			f.Name = cur[0].GetName()
		}
		if f.Size == 0 {
			f.Size = int(cur[0].GetHeight())
		}
	}
	sf := f.swtFont(c.GetDisplay())
	c.SetFont(sf)
	old, had := ownedFonts[c]
	ownedFonts[c] = sf
	if had {
		old.Dispose()
	} else {
		c.AddDisposeListener(&fontRelease{c})
	}
}

// ownedFonts holds the font SetFont made for each control; UI thread only.
var ownedFonts = map[*swt.Control]*swt.Font{}

type fontRelease struct{ c *swt.Control }

func (r *fontRelease) WidgetDisposed(*swt.DisposeEvent) {
	ownedFonts[r.c].Dispose()
	delete(ownedFonts, r.c)
}

// SetForeground sets the text color of the control.
func (w *base) SetForeground(c RGB) { w.ctl.SetForeground(c.color()) }

// SetBackground sets the background color of the control; Background() does the same at creation.
func (w *base) SetBackground(c RGB) { w.ctl.SetBackgroundWithColor(c.color()) }

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
		p.LayoutOverload1(true)
	}
}

// OnDispose runs f when the control is disposed: by Dispose, with its parent, or when its window closes.
func (w *base) OnDispose(f func()) { w.ctl.AddDisposeListener(disposeFunc(f)) }

type disposeFunc func()

func (f disposeFunc) WidgetDisposed(*swt.DisposeEvent) { f() }

// Style options. Each applies to the widgets whose SWT style has that bit and is silently ignored
// elsewhere. Check (gowt.go) also makes Table and Tree rows checkable.

// MultiSelect allows several selected items in a List, Table or Tree.
func MultiSelect() Option { return Option{style: swt.MULTI} }

// Simple makes a Combo show its list permanently.
func Simple() Option { return Option{style: swt.SIMPLE} }

// Vertical rotates a Scale, Slider or Progress, stacks a Split, and makes a Sash a vertical bar.
func Vertical() Option { return Option{style: swt.VERTICAL} }

// Smooth makes a Progress an unsegmented bar.
func Smooth() Option { return Option{style: swt.SMOOTH} }

// Indeterminate animates a Progress without a value.
func Indeterminate() Option { return Option{style: swt.INDETERMINATE} }

// Flat gives a ToolBar the flat look.
func Flat() Option { return Option{style: swt.FLAT} }

// Wrap breaks long text of a Label or StyledText into lines, and lets a ToolBar wrap its items to further rows.
func Wrap() Option { return Option{style: swt.WRAP} }

// Bottom puts the tabs of Tabs or CTabs below the pages.
func Bottom() Option { return Option{style: swt.BOTTOM} }

// Closable adds a close button to a CTabs tab.
func Closable() Option { return Option{style: swt.CLOSE} }

// DropDown makes a DateTime date field a drop-down calendar, or a ToolBar item a drop-down.
func DropDown() Option { return Option{style: swt.DROP_DOWN} }

// Right right-aligns a table Column.
func Right() Option { return Option{style: swt.RIGHT} }

// Center centers a table Column.
func Center() Option { return Option{style: swt.CENTER} }

// AsTime and AsCalendar turn a DateTime from a date field into a time field or a month calendar.
func AsTime() Option { return Option{style: swt.TIME} }

// AsCalendar makes a DateTime a month calendar.
func AsCalendar() Option { return Option{style: swt.CALENDAR} }

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

// SetImage sets the icon; the label does not own img.
func (l *Label) SetImage(img *Image) { l.l.SetImage(img.i) }

// SetImage sets the icon; the button does not own img.
func (b *Button) SetImage(img *Image) { b.b.SetImage(img.i) }
