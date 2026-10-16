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

// ScaleView is a slider with tick marks; Vertical() rotates it.
type ScaleView struct {
	base
	s *swt.Scale
}

// Scale creates a scale over [min, max] set to value.
// Options: Vertical.
func Scale(min, max, value int, opts ...Option) Option {
	return leaf("Scale", sVertical, opts, func(p *swt.Composite) Widget {
		s := swt.NewScale(p, resolve(swt.NONE, opts))
		setRange(s, min, max)
		s.SetSelection(int32(value))
		return &ScaleView{base{&s.Control}, s}
	})
}

// Value returns the position within [min, max].
func (s *ScaleView) Value() int { return int(s.s.GetSelection()) }

// SetValue moves the scale to v.
func (s *ScaleView) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Scale for the full API.
func (s *ScaleView) Unwrap() *swt.Scale { return s.s }

// OnChange runs f with the new value.
func (s *ScaleView) OnChange(f func(value int)) {
	onSelect(s.s, func(*swt.SelectionEvent) { f(s.Value()) })
}

// SliderView is a scroll-bar style control; Vertical() rotates it.
type SliderView struct {
	base
	s *swt.Slider
}

// Slider creates a slider over [min, max] set to value.
// Options: Vertical.
func Slider(min, max, value int, opts ...Option) Option {
	return leaf("Slider", sVertical, opts, func(p *swt.Composite) Widget {
		s := swt.NewSlider(p, resolve(swt.NONE, opts))
		setRange(s, min, max)
		s.SetSelection(int32(value))
		return &SliderView{base{&s.Control}, s}
	})
}

// Value returns the position within [min, max].
func (s *SliderView) Value() int { return int(s.s.GetSelection()) }

// SetValue moves the slider to v.
func (s *SliderView) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Slider for the full API.
func (s *SliderView) Unwrap() *swt.Slider { return s.s }

// OnChange runs f with the new value.
func (s *SliderView) OnChange(f func(value int)) {
	onSelect(s.s, func(*swt.SelectionEvent) { f(s.Value()) })
}

// SpinnerView is an integer edit field with arrows.
type SpinnerView struct {
	base
	s *swt.Spinner
}

// Spinner creates a spinner over [min, max] set to value.
// Options: ReadOnly, Border.
func Spinner(min, max, value int, opts ...Option) Option {
	return leaf("Spinner", sReadOnly|sBorder, opts, func(p *swt.Composite) Widget {
		s := swt.NewSpinner(p, resolve(swt.NONE, opts))
		setRange(s, min, max)
		s.SetSelection(int32(value))
		return &SpinnerView{base{&s.Control}, s}
	})
}

// Value returns the number within [min, max].
func (s *SpinnerView) Value() int { return int(s.s.GetSelection()) }

// SetValue sets the number to v.
func (s *SpinnerView) SetValue(v int) { s.s.SetSelection(int32(v)) }

// Unwrap returns the underlying swt.Spinner for the full API.
func (s *SpinnerView) Unwrap() *swt.Spinner { return s.s }

// OnChange runs f with the new value after the arrows or typing changed it.
func (s *SpinnerView) OnChange(f func(value int)) {
	s.s.AddModifyListener(&modifier{func() { f(s.Value()) }})
}

// ProgressView is a progress bar; Vertical() rotates it, Indeterminate() animates it without a value.
type ProgressView struct {
	base
	b *swt.ProgressBar
}

// Progress creates a progress bar over [0, max].
// Options: Vertical, Smooth, Indeterminate.
func Progress(max int, opts ...Option) Option {
	return leaf("Progress", sVertical|sSmooth|sIndeterminate, opts, func(p *swt.Composite) Widget {
		b := swt.NewProgressBar(p, resolve(swt.NONE, opts))
		b.SetMaximum(int32(max))
		return &ProgressView{base{&b.Control}, b}
	})
}

// Value returns the progress within [0, max].
func (p *ProgressView) Value() int { return int(p.b.GetSelection()) }

// SetValue sets the progress to v, within [0, max].
func (p *ProgressView) SetValue(v int) { p.b.SetSelection(int32(v)) }

// SetMax sets the upper bound.
func (p *ProgressView) SetMax(v int) { p.b.SetMaximum(int32(v)) }

// Unwrap returns the underlying swt.ProgressBar for the full API.
func (p *ProgressView) Unwrap() *swt.ProgressBar { return p.b }

// DateTimeView is a date field, a time field (AsTime) or a month calendar (AsCalendar).
type DateTimeView struct {
	base
	d *swt.DateTime
}

// DateTime creates a date picker; DropDown() makes the date field a drop-down calendar.
// Options: DropDown, AsTime, AsCalendar.
func DateTime(opts ...Option) Option {
	return leaf("DateTime", sDropDown|sAsTime|sAsCalendar, opts, func(p *swt.Composite) Widget {
		d := swt.NewDateTime(p, resolve(swt.NONE, opts))
		return &DateTimeView{base{&d.Control}, d}
	})
}

// Value is the shown date, or the shown time of day on today's date for AsTime, in the local zone.
func (d *DateTimeView) Value() time.Time {
	if d.d.GetStyle()&swt.TIME != 0 {
		n := time.Now()
		return time.Date(n.Year(), n.Month(), n.Day(), int(d.d.GetHours()), int(d.d.GetMinutes()), int(d.d.GetSeconds()), 0, time.Local)
	}
	return time.Date(int(d.d.GetYear()), time.Month(d.d.GetMonth()+1), int(d.d.GetDay()), 0, 0, 0, 0, time.Local)
}

// SetValue shows t's time of day for AsTime, its date otherwise.
func (d *DateTimeView) SetValue(t time.Time) {
	if d.d.GetStyle()&swt.TIME != 0 {
		d.d.SetTime(int32(t.Hour()), int32(t.Minute()), int32(t.Second()))
		return
	}
	d.d.SetDate(int32(t.Year()), int32(t.Month())-1, int32(t.Day()))
}

// Unwrap returns the underlying swt.DateTime for the full API.
func (d *DateTimeView) Unwrap() *swt.DateTime { return d.d }

// OnChange runs f with the new value.
func (d *DateTimeView) OnChange(f func(t time.Time)) {
	onSelect(d.d, func(*swt.SelectionEvent) { f(d.Value()) })
}

// LinkView is a label with clickable parts: text uses <a href="...">anchor</a>.
type LinkView struct {
	base
	l *swt.Link
}

// Link creates a link label; onClick gets the href of the clicked anchor (its text if there is no href). onClick may be nil.
// Options: common only.
func Link(text string, onClick func(href string), opts ...Option) Option {
	return leaf("Link", 0, opts, func(p *swt.Composite) Widget {
		l := swt.NewLink(p, resolve(swt.NONE, opts))
		l.SetText(text)
		w := &LinkView{base{&l.Control}, l}
		if onClick != nil {
			w.OnClick(onClick)
		}
		return w
	})
}

// SetText replaces the text; anchors use <a href="...">.
func (l *LinkView) SetText(s string) { l.l.SetText(s) }

// Unwrap returns the underlying swt.Link for the full API.
func (l *LinkView) Unwrap() *swt.Link { return l.l }

// OnClick runs f with the clicked anchor's href.
func (l *LinkView) OnClick(f func(href string)) {
	onSelect(l.l, func(e *swt.SelectionEvent) { f(e.Text) })
}
