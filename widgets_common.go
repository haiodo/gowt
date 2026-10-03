package gowt

import "github.com/haiodo/gowt/swt"

// Widget is any gowt control. Layouts and containers take it where they refer to a sibling or a child.
type Widget interface{ control() *swt.Control }

func (p *Panel) control() *swt.Control  { return &p.c.Control }
func (l *Label) control() *swt.Control  { return &l.l.Control }
func (b *Button) control() *swt.Control { return &b.b.Control }
func (t *Text) control() *swt.Control   { return &t.t.Control }

// Style options. Each applies to the widgets whose SWT style has that bit and is silently ignored
// elsewhere. Check (gowt.go) also makes Table and Tree rows checkable.

// MultiSelect allows several selected items in a List, Table or Tree.
func MultiSelect() Option   { return Option{style: swt.MULTI} }
func Simple() Option        { return Option{style: swt.SIMPLE} }
func Vertical() Option      { return Option{style: swt.VERTICAL} }
func Smooth() Option        { return Option{style: swt.SMOOTH} }
func Indeterminate() Option { return Option{style: swt.INDETERMINATE} }
func Flat() Option          { return Option{style: swt.FLAT} }
func Wrap() Option          { return Option{style: swt.WRAP} }
func Bottom() Option        { return Option{style: swt.BOTTOM} }
func Closable() Option      { return Option{style: swt.CLOSE} }
func DropDown() Option      { return Option{style: swt.DROP_DOWN} }
func Right() Option         { return Option{style: swt.RIGHT} }
func Center() Option        { return Option{style: swt.CENTER} }

// AsTime and AsCalendar turn a DateTime from a date field into a time field or a month calendar.
func AsTime() Option     { return Option{style: swt.TIME} }
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
