package gowt

import "github.com/haiodo/gowt/swt"

// Widget is any gowt control. Layouts and containers take it where they refer to a sibling or a child.
type Widget interface{ control() *swt.Control }

func (p *panel) control() *swt.Control  { return &p.c.Control }
func (l *Label) control() *swt.Control  { return &l.l.Control }
func (b *Button) control() *swt.Control { return &b.b.Control }
func (t *Text) control() *swt.Control   { return &t.t.Control }
func (s *Switch) control() *swt.Control { return &s.b.Control }

// SetEnabled enables or disables any control, a Window or container included (its children follow).
// A disabled control ignores input and is drawn dimmed; Disabled() does the same at creation.
func SetEnabled(w Widget, v bool) { w.control().SetEnabled(v) }

// Enabled reports the state set by SetEnabled, not that of the parents.
func Enabled(w Widget) bool { return w.control().GetEnabled() }

// SetVisible shows or hides a control. In a Grid or Row layout a hidden control also gives up its
// cell, and the parent is laid out again so the neighbours take the space. Other layouts keep the gap.
func SetVisible(w Widget, v bool) {
	c := w.control()
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
		// without data only Grid has a default to exclude; Row and the others ignore the hint
		if l := p.GetLayout(); l != nil {
			if _, ok := l.Impl().(*swt.GridLayout); ok {
				g := swt.NewGridData()
				g.Exclude = !v
				c.SetLayoutData(g)
			}
		}
	}
	p.LayoutOverload1(true)
}

// Visible reports the state set by SetVisible, not that of the parents.
func Visible(w Widget) bool { return w.control().GetVisible() }

// SetFont gives a control the font f. Empty Name and zero Size keep the control's current ones.
// The font is released with the control.
func SetFont(w Widget, f Font) {
	c := w.control()
	if cur := c.GetFont().GetFontData(); len(cur) > 0 {
		if f.Name == "" {
			f.Name = cur[0].GetName()
		}
		if f.Size == 0 {
			f.Size = int(cur[0].GetHeight())
		}
	}
	var st int32
	if f.Bold {
		st |= swt.BOLD
	}
	if f.Italic {
		st |= swt.ITALIC
	}
	sf := swt.NewFontDeviceNameHeightStyle(c.GetDisplay(), f.Name, int32(f.Size), st)
	c.SetFont(sf)
	c.AddDisposeListener(&fontRelease{sf})
}

type fontRelease struct{ f *swt.Font }

func (r *fontRelease) WidgetDisposed(*swt.DisposeEvent) { r.f.Dispose() }

// SetForeground sets the text color of a control.
func SetForeground(w Widget, c RGB) { w.control().SetForeground(c.color()) }

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

// Wrap lets a ToolBar wrap its items to further rows.
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
