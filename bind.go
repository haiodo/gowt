package gowt

import (
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// Bind calls apply with the value of r now and again whenever it changes, once per batch. Only r
// is tracked: signals that apply reads do not rerun it. The binding lives as long as w, or until
// the Effect or Scope it was made in is disposed.
func Bind[T any](w Widget, r rx.Reader[T], apply func(T)) {
	own(w, func() {
		rx.Effect(func() {
			v := r.Get()
			rx.Untrack(func() { apply(v) })
		})
	})
}

// BindText shows r in the label. A changed text lays the window out again on the next event-loop
// turn, so a label that grows is not clipped.
func (l *Label) BindText(r rx.Reader[string]) {
	last := l.l.GetText()
	Bind(l, r, func(s string) {
		if s == last {
			return
		}
		last = s
		l.l.SetText(s)
		l.l.GetShell().LayoutOverload4([]*swt.Control{&l.l.Control}, swt.DEFER)
	})
}

// BindChecked keeps a Check or Radio button and s in step both ways: a click sets s, a Set of s
// checks or unchecks the button.
func (b *Button) BindChecked(s *rx.State[bool]) {
	last := b.b.GetSelection()
	l := swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
		last = b.b.GetSelection()
		s.Set(last)
	})
	own(b, func() {
		b.b.AddSelectionListener(l)
		rx.OnCleanup(func() {
			if !b.b.IsDisposed() {
				b.b.RemoveSelectionListener(l)
			}
		})
		rx.Effect(func() {
			if v := s.Get(); v != last {
				last = v
				b.b.SetSelection(v)
			}
		})
	})
}

// Bind keeps the field and s in step both ways: every edit sets s, a Set of s replaces the
// content. The content is replaced only when s differs from it, so typing keeps the caret.
func (t *Text) Bind(s *rx.State[string]) {
	last := t.t.GetText()
	m := &modifier{func() {
		last = t.t.GetText()
		s.Set(last)
	}}
	own(t, func() {
		t.t.AddModifyListener(m)
		rx.OnCleanup(func() {
			if !t.t.IsDisposed() {
				t.t.RemoveModifyListener(m)
			}
		})
		rx.Effect(func() {
			if v := s.Get(); v != last {
				last = v
				t.t.SetText(v) // its ModifyText sets s to v again, an equal value: no loop
			}
		})
	})
}

// BindVisible shows or hides w as r changes, with SetVisible's handling of Grid and Row layouts.
func BindVisible(w Widget, r rx.Reader[bool]) {
	Bind(w, r, func(v bool) { SetVisible(w, v) })
}

// BindEnabled enables or disables w as r changes.
func BindEnabled(w Widget, r rx.Reader[bool]) {
	Bind(w, r, func(v bool) { SetEnabled(w, v) })
}

// own runs body in an rx.Scope that is disposed with w. body makes the effects of one binding and
// removes with rx.OnCleanup whatever it added to the widget.
func own(w Widget, body func()) {
	c := w.control()
	u := &unbinder{c: c}
	u.stop = rx.Scope(func() {
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
