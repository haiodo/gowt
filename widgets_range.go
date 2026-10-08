package gowt

import (
	"time"

	"github.com/haiodo/gowt/swt"
)

type ranged interface {
	SetMinimum(int32)
	SetMaximum(int32)
	GetMaximum() int32
}

// setRange orders the two calls so SWT never sees min >= max (it ignores such calls).
func setRange(r ranged, min, max int) {
	if int32(min) >= r.GetMaximum() {
		r.SetMaximum(int32(max))
		r.SetMinimum(int32(min))
		return
	}
	r.SetMinimum(int32(min))
	r.SetMaximum(int32(max))
}

// Scale is a slider with tick marks; Vertical() rotates it.
type Scale struct{ s *swt.Scale }

// Scale adds a scale over [min, max] set to value.
func (p *Panel) Scale(min, max, value int, opts ...Option) *Scale {
	s := swt.NewScale(p.c, resolve(swt.NONE, opts))
	setRange(s, min, max)
	s.SetSelection(int32(value))
	applyOpts(&s.Control, opts)
	return &Scale{s}
}

func (s *Scale) control() *swt.Control { return &s.s.Control }

// Value returns the current position.
func (s *Scale) Value() int { return int(s.s.GetSelection()) }

// SetValue moves the scale to v.
func (s *Scale) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Scale for the full API.
func (s *Scale) Unwrap() *swt.Scale { return s.s }

// OnChange runs f with the new value.
func (s *Scale) OnChange(f func(value int)) {
	onSelect(s.s, func(*swt.SelectionEvent) { f(s.Value()) })
}

// Slider is a scroll-bar style control; Vertical() rotates it.
type Slider struct{ s *swt.Slider }

// Slider adds a slider over [min, max] set to value.
func (p *Panel) Slider(min, max, value int, opts ...Option) *Slider {
	s := swt.NewSlider(p.c, resolve(swt.NONE, opts))
	setRange(s, min, max)
	s.SetSelection(int32(value))
	applyOpts(&s.Control, opts)
	return &Slider{s}
}

func (s *Slider) control() *swt.Control { return &s.s.Control }

// Value returns the current position.
func (s *Slider) Value() int { return int(s.s.GetSelection()) }

// SetValue moves the slider to v.
func (s *Slider) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Slider for the full API.
func (s *Slider) Unwrap() *swt.Slider { return s.s }

// OnChange runs f with the new value.
func (s *Slider) OnChange(f func(value int)) {
	onSelect(s.s, func(*swt.SelectionEvent) { f(s.Value()) })
}

// Spinner is an integer edit field with arrows.
type Spinner struct{ s *swt.Spinner }

// Spinner adds a spinner over [min, max] set to value.
func (p *Panel) Spinner(min, max, value int, opts ...Option) *Spinner {
	s := swt.NewSpinner(p.c, resolve(swt.NONE, opts))
	setRange(s, min, max)
	s.SetSelection(int32(value))
	applyOpts(&s.Control, opts)
	return &Spinner{s}
}

func (s *Spinner) control() *swt.Control { return &s.s.Control }

// Value returns the current number.
func (s *Spinner) Value() int { return int(s.s.GetSelection()) }

// SetValue sets the number to v.
func (s *Spinner) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Spinner for the full API.
func (s *Spinner) Unwrap() *swt.Spinner { return s.s }

// OnChange runs f with the new value after the arrows or typing changed it.
func (s *Spinner) OnChange(f func(value int)) {
	s.s.AddModifyListener(&modifier{func() { f(s.Value()) }})
}

// Progress is a progress bar; Vertical() rotates it, Indeterminate() animates it without a value.
type Progress struct{ b *swt.ProgressBar }

// Progress adds a progress bar over [0, max].
func (p *Panel) Progress(max int, opts ...Option) *Progress {
	b := swt.NewProgressBar(p.c, resolve(swt.NONE, opts))
	b.SetMaximum(int32(max))
	applyOpts(&b.Control, opts)
	return &Progress{b}
}

func (p *Progress) control() *swt.Control { return &p.b.Control }

// Value returns the current progress.
func (p *Progress) Value() int { return int(p.b.GetSelection()) }

// SetValue sets the progress to v, within [0, max].
func (p *Progress) SetValue(v int) { p.b.SetSelection(int32(v)) }

// SetMax sets the upper bound.
func (p *Progress) SetMax(v int) { p.b.SetMaximum(int32(v)) }

// Unwrap returns the underlying swt.ProgressBar for the full API.
func (p *Progress) Unwrap() *swt.ProgressBar { return p.b }

// DateTime is a date field, a time field (AsTime) or a month calendar (AsCalendar).
type DateTime struct{ d *swt.DateTime }

// DateTime adds a date picker; DropDown() makes the date field a drop-down calendar.
func (p *Panel) DateTime(opts ...Option) *DateTime {
	d := swt.NewDateTime(p.c, resolve(swt.NONE, opts))
	applyOpts(&d.Control, opts)
	return &DateTime{d}
}

func (d *DateTime) control() *swt.Control { return &d.d.Control }

// Value is the shown date, or the shown time of day on today's date for AsTime, in the local zone.
func (d *DateTime) Value() time.Time {
	if d.d.GetStyle()&swt.TIME != 0 {
		n := time.Now()
		return time.Date(n.Year(), n.Month(), n.Day(), int(d.d.GetHours()), int(d.d.GetMinutes()), int(d.d.GetSeconds()), 0, time.Local)
	}
	return time.Date(int(d.d.GetYear()), time.Month(d.d.GetMonth()+1), int(d.d.GetDay()), 0, 0, 0, 0, time.Local)
}

// SetValue shows t's time of day for AsTime, its date otherwise.
func (d *DateTime) SetValue(t time.Time) {
	if d.d.GetStyle()&swt.TIME != 0 {
		d.d.SetTime(int32(t.Hour()), int32(t.Minute()), int32(t.Second()))
		return
	}
	d.d.SetDate(int32(t.Year()), int32(t.Month())-1, int32(t.Day()))
}

// Unwrap returns the underlying swt.DateTime for the full API.
func (d *DateTime) Unwrap() *swt.DateTime { return d.d }

// OnChange runs f with the new value.
func (d *DateTime) OnChange(f func(t time.Time)) {
	onSelect(d.d, func(*swt.SelectionEvent) { f(d.Value()) })
}

// Link is a label with clickable parts: text uses <a href="...">anchor</a>.
type Link struct{ l *swt.Link }

// Link adds a link label; onClick gets the href of the clicked anchor (its text if there is no href). onClick may be nil.
func (p *Panel) Link(text string, onClick func(href string), opts ...Option) *Link {
	l := swt.NewLink(p.c, resolve(swt.NONE, opts))
	l.SetText(text)
	applyOpts(&l.Control, opts)
	w := &Link{l}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

func (l *Link) control() *swt.Control { return &l.l.Control }

// SetText replaces the text; anchors use <a href="...">.
func (l *Link) SetText(s string) { l.l.SetText(s) }

// Unwrap returns the underlying swt.Link for the full API.
func (l *Link) Unwrap() *swt.Link { return l.l }

// OnClick runs f with the clicked anchor's href.
func (l *Link) OnClick(f func(href string)) {
	onSelect(l.l, func(e *swt.SelectionEvent) { f(e.Text) })
}
