package gowt

import (
	"fmt"
	"time"
)

// Event options. Each names one concept for every widget that has it; the handler's parameter is
// what the widget reports. A handler of a type the widget does not report goes to App.OnError.
// The methods of the same names add a handler after creation.

// OnClick runs f when a Button is pressed (func()), or with the href of the anchor a Link's user
// clicked (func(href string)).
func OnClick[F func() | func(string)](f F) Option {
	return hook(func(w Widget) error {
		switch w := w.(type) {
		case *ButtonView:
			if g, ok := any(f).(func()); ok {
				w.OnClick(g)
				return nil
			}
		case *LinkView:
			if g, ok := any(f).(func(string)); ok {
				w.OnClick(g)
				return nil
			}
		}
		return misfit("OnClick", f, w)
	})
}

// OnChange runs f with the new value after the user changed it: the text of a Text, Combo or
// StyledText (func(string)), the state of a Switch or a Check button (func(bool)), the value of a
// Scale, Slider or Spinner (func(int)), the value of a DateTime (func(time.Time)).
func OnChange[F func(string) | func(bool) | func(int) | func(time.Time)](f F) Option {
	return hook(func(w Widget) error {
		switch g := any(f).(type) {
		case func(string):
			if w, ok := w.(interface{ OnChange(func(string)) }); ok {
				w.OnChange(g)
				return nil
			}
		case func(bool):
			if b, ok := w.(*ButtonView); ok {
				b.OnClick(func() { g(b.Checked()) })
				return nil
			}
			if w, ok := w.(interface{ OnChange(func(bool)) }); ok {
				w.OnChange(g)
				return nil
			}
		case func(int):
			if w, ok := w.(interface{ OnChange(func(int)) }); ok {
				w.OnChange(g)
				return nil
			}
		case func(time.Time):
			if w, ok := w.(interface{ OnChange(func(time.Time)) }); ok {
				w.OnChange(g)
				return nil
			}
		}
		return misfit("OnChange", f, w)
	})
}

// OnSelect runs f when the selection changes: with the index of the selected item of a Combo,
// List, Table, Tabs or CTabs (func(int)), with the selected node of a Tree (func(*Node)).
func OnSelect[F func(int) | func(*Node)](f F) Option {
	return hook(func(w Widget) error {
		switch g := any(f).(type) {
		case func(int):
			if w, ok := w.(interface{ OnSelect(func(int)) }); ok {
				w.OnSelect(g)
				return nil
			}
		case func(*Node):
			if t, ok := w.(*TreeView); ok {
				t.OnSelect(g)
				return nil
			}
		}
		return misfit("OnSelect", f, w)
	})
}

// OnActivate runs f on Enter in a single-line Text (func()), and on a double click or Enter on an
// item of a List or Table (func(int)) or a node of a Tree (func(*Node)).
func OnActivate[F func() | func(int) | func(*Node)](f F) Option {
	return hook(func(w Widget) error {
		switch g := any(f).(type) {
		case func():
			if t, ok := w.(*TextView); ok {
				t.OnActivate(g)
				return nil
			}
		case func(int):
			if w, ok := w.(interface{ OnActivate(func(int)) }); ok {
				w.OnActivate(g)
				return nil
			}
		case func(*Node):
			if t, ok := w.(*TreeView); ok {
				t.OnActivate(g)
				return nil
			}
		}
		return misfit("OnActivate", f, w)
	})
}

// Initial values. They set the state without running the event handlers.

// Hint sets the placeholder a Text shows while it is empty.
func Hint(s string) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TextView); ok {
			t.SetHint(s)
			return nil
		}
		return fmt.Errorf("gowt: Hint applies to Text, not to %s", nameOf(w.control()))
	})
}

// Checked sets the state of a Switch or of a Check or Radio button.
func Checked(on bool) Option {
	return hook(func(w Widget) error {
		if w, ok := w.(interface{ SetChecked(bool) }); ok {
			w.SetChecked(on)
			return nil
		}
		return fmt.Errorf("gowt: Checked applies to Switch and Button, not to %s", nameOf(w.control()))
	})
}

// Selected selects item i of a Combo, List or Table, or tab i of a Tabs or CTabs.
func Selected(i int) Option {
	return hook(func(w Widget) error {
		switch w := w.(type) {
		case *ListView:
			w.Select(i)
		case *TableView:
			w.Select(i)
		case interface{ Select(int) }:
			w.Select(i)
		default:
			return fmt.Errorf("gowt: Selected applies to Combo, List, Table, Tabs and CTabs, not to %s", nameOf(w.control()))
		}
		return nil
	})
}

// Value sets the starting value: the text of a Text, Combo or StyledText (string), the value of a
// Scale, Slider, Spinner or Progress (int; without it the minimum), the date or time of a DateTime.
func Value[T string | int | time.Time](v T) Option {
	return hook(func(w Widget) error {
		if w, ok := w.(interface{ SetValue(T) }); ok {
			w.SetValue(v)
			return nil
		}
		if w, ok := w.(interface{ SetText(T) }); ok && !isLabel(w) {
			w.SetText(v)
			return nil
		}
		return fmt.Errorf("gowt: Value(%T) does not apply to %s", v, nameOf(w.control()))
	})
}

// isLabel tells the widgets whose SetText is a caption, not a value.
func isLabel(w any) bool {
	switch w.(type) {
	case *LabelView, *ButtonView, *SwitchView, *LinkView:
		return true
	}
	return false
}

func misfit(opt string, f any, w Widget) error {
	return fmt.Errorf("gowt: %s(%T) does not apply to %s", opt, f, nameOf(w.control()))
}
