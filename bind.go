package gowt

import (
	"fmt"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// Watch calls apply with the value of r now and again whenever it changes, once per batch. Only r
// is tracked: signals that apply reads do not rerun it. The binding lives as long as w, or until
// the Effect or Scope it was made in is disposed. The Bind options cover the common properties.
func Watch[T any](w Widget, r rx.Reader[T], apply func(T)) {
	own(w, func() {
		rx.Effect(func() {
			v := r.Get()
			rx.Untrack(func() { apply(v) })
		})
	})
}

// Bind keeps the widget's value and s in step both ways, as Svelte's bind:value: a user's edit sets
// s, a Set of s changes the widget. The value is the text of a Text or Combo (string), the state of
// a Switch or a Check or Toggle button or whether a Disclosure is expanded (bool), the value of a Scale, Slider, Spinner or Progress (one
// way), or the selected index of a Combo, List or Segmented (int). A Text is rewritten only when s differs from it, so typing keeps the caret.
func Bind[T string | bool | int](s *rx.State[T]) Option {
	return hook(func(w Widget) error {
		switch s := any(s).(type) {
		case *rx.State[string]:
			return bindString(w, s)
		case *rx.State[bool]:
			return bindBool(w, s)
		case *rx.State[int]:
			return bindInt(w, s)
		}
		return nil
	})
}

func bindString(w Widget, s *rx.State[string]) error {
	switch w := w.(type) {
	case *TextView:
		twoWay(w, s, w.Text, w.SetText, onModify(w.t))
	case *ComboView:
		twoWay(w, s, w.Text, w.SetText, onModify(w.c))
	default:
		return fmt.Errorf("gowt: Bind(*rx.State[string]) applies to Text and Combo, not to %s", nameOf(w.control()))
	}
	return nil
}

func bindBool(w Widget, s *rx.State[bool]) error {
	switch w := w.(type) {
	case *ButtonView:
		twoWay(w, s, w.Checked, w.SetChecked, onSelection(w.b))
	case *SwitchView:
		twoWay(w, s, w.Checked, w.SetChecked, onSelection(w.b))
	case *DisclosureView:
		twoWay(w, s, w.Expanded, w.SetExpanded, w.listen)
	default:
		return fmt.Errorf("gowt: Bind(*rx.State[bool]) applies to Switch, Button and Disclosure, not to %s", nameOf(w.control()))
	}
	return nil
}

func bindInt(w Widget, s *rx.State[int]) error {
	switch w := w.(type) {
	case *ScaleView:
		twoWay(w, s, w.Value, w.SetValue, onSelection(w.s))
	case *SliderView:
		twoWay(w, s, w.Value, w.SetValue, onSelection(w.s))
	case *SpinnerView:
		twoWay(w, s, w.Value, w.SetValue, onModify(w.s))
	case *ComboView:
		twoWay(w, s, w.Index, w.Select, onSelection(w.c))
	case *ListView:
		twoWay(w, s, w.Index, func(i int) { w.Select(i) }, onSelection(w.l))
	case *SegmentedView:
		twoWay(w, s, w.Index, w.Select, w.listen)
	case *ProgressView:
		twoWay(w, s, w.Value, w.SetValue, func(func()) func() { return func() {} }) // the user does not move a progress bar
	default:
		return fmt.Errorf("gowt: Bind(*rx.State[int]) applies to Scale, Slider, Spinner, Progress, Combo, List and Segmented, not to %s", nameOf(w.control()))
	}
	return nil
}

// twoWay keeps get/set and s in step: listen registers a callback for the user's changes and
// returns its removal. The widget is written only when s differs from what it last showed.
func twoWay[T comparable](w Widget, s *rx.State[T], get func() T, set func(T), listen func(func()) (remove func())) {
	last := get()
	c := w.control()
	own(w, func() {
		remove := listen(func() {
			last = get()
			s.Set(last)
		})
		rx.OnCleanup(func() {
			if !c.IsDisposed() {
				remove()
			}
		})
		rx.Effect(func() {
			if v := s.Get(); v != last {
				last = v
				set(v) // a listener that fires on set sets s to v again, an equal value: no loop
			}
		})
	})
}

type selectable interface {
	AddSelectionListener(swt.SelectionListener)
	RemoveSelectionListener(swt.SelectionListener)
}

func onSelection(x selectable) func(func()) func() {
	return func(f func()) func() {
		l := swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) { f() })
		x.AddSelectionListener(l)
		return func() { x.RemoveSelectionListener(l) }
	}
}

type modifiable interface {
	AddModifyListener(swt.ModifyListener)
	RemoveModifyListener(swt.ModifyListener)
}

func onModify(x modifiable) func(func()) func() {
	return func(f func()) func() {
		m := &modifier{f}
		x.AddModifyListener(m)
		return func() { x.RemoveModifyListener(m) }
	}
}

// BindText shows r as the text of a Label, Button or Link. As with SetText, a text that changes the
// widget's size lays its parents out again on the next event-loop turn, so a text that grows is not cut.
func BindText(r rx.Reader[string]) Option {
	return hook(func(w Widget) error {
		t, ok := w.(interface{ SetText(string) })
		if _, field := w.(*TextView); !ok || field {
			return fmt.Errorf("gowt: BindText applies to Label, Button and Link, not to %s; a Text takes Bind", nameOf(w.control()))
		}
		last := ""
		first := true
		Watch(w, r, func(s string) {
			if s == last && !first {
				return
			}
			first, last = false, s
			t.SetText(s)
		})
		return nil
	})
}

// BindEnabled enables or disables the widget as r changes.
func BindEnabled(r rx.Reader[bool]) Option {
	return hook(func(w Widget) error {
		Watch(w, r, w.SetEnabled)
		return nil
	})
}

// BindVisible shows or hides the widget as r changes, with SetVisible's handling of the layouts.
func BindVisible(r rx.Reader[bool]) Option {
	return hook(func(w Widget) error {
		Watch(w, r, w.SetVisible)
		return nil
	})
}

// own runs body in an rx.Scope that is disposed with w; inside a Boundary its panics go there. body makes the effects of one binding and
// removes with rx.OnCleanup whatever it added to the widget.
func own(w Widget, body func()) {
	c := w.control()
	u := &unbinder{c: c}
	scope := rx.Scope
	if b := boundaryOf(c); b != nil {
		scope = func(f func()) func() { return rx.Catch(f, b.caught) }
	}
	u.stop = scope(func() {
		rx.OnCleanup(u.detach)
		body()
	})
	c.AddDisposeListener(u)
}

type unbinder struct {
	c     *swt.Control
	stop  func()
	fired bool
}

func (u *unbinder) WidgetDisposed(*swt.DisposeEvent) {
	u.fired = true
	u.stop()
}

// detach runs when the owning Effect or Scope goes first; the widget lives on without the binding.
func (u *unbinder) detach() {
	if !u.fired && !u.c.IsDisposed() {
		u.c.RemoveDisposeListener(u)
	}
}
