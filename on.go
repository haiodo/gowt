package gowt

import (
	"fmt"
	"math"
	"time"

	"github.com/haiodo/gowt/swt"
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
// StyledText (func(string)), the state of a Switch, a Check or Toggle button or a Disclosure
// (func(bool)), the value of a Scale, Slider or Spinner (func(int)), the value of a Number
// (func(float64), or func(int) rounded), the value of a DateTime (func(time.Time)).
func OnChange[F func(string) | func(bool) | func(int) | func(float64) | func(time.Time)](f F) Option {
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
			if n, ok := w.(*NumberView); ok {
				n.OnChange(func(v float64) { g(int(math.Round(v))) })
				return nil
			}
		case func(float64):
			if n, ok := w.(*NumberView); ok {
				n.OnChange(g)
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

// OnEdit runs f when a Text receives a Modify event outside a facade text write.
func OnEdit(f func(string)) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TextView); ok {
			t.OnEdit(f)
			return nil
		}
		return misfit("OnEdit", f, w)
	})
}

// OnSelect runs f when the selection changes: with the index of the selected item of a Combo,
// List, Table, Tabs or CTabs or of the segment of a Segmented (func(int)), with the selected node of
// a Tree (func(*Node)), or the start and length of a StyledText selection (func(int, int)).
func OnSelect[F func(int) | func(*Node) | func(int, int)](f F) Option {
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
		case func(int, int):
			if t, ok := w.(*StyledTextView); ok {
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

// Checked sets the state of a Switch or of a Check, Radio or Toggle button.
func Checked(on bool) Option {
	return hook(func(w Widget) error {
		if w, ok := w.(interface{ SetChecked(bool) }); ok {
			w.SetChecked(on)
			return nil
		}
		return fmt.Errorf("gowt: Checked applies to Switch and Button, not to %s", nameOf(w.control()))
	})
}

// Selected selects item i of a Combo, List or Table, tab i of a Tabs or CTabs, or segment i of a Segmented.
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
			return fmt.Errorf("gowt: Selected applies to Combo, List, Table, Tabs, CTabs and Segmented, not to %s", nameOf(w.control()))
		}
		return nil
	})
}

// Value sets the starting value: the text of a Text, Combo or StyledText (string), the value of a
// Scale, Slider, Spinner or Progress (int; without it the minimum), of a Number (float64 or int), the
// date or time of a DateTime.
func Value[T string | int | float64 | time.Time](v T) Option {
	return hook(func(w Widget) error {
		if n, ok := w.(*NumberView); ok {
			if i, ok := any(v).(int); ok {
				n.SetValue(float64(i))
				return nil
			}
		}
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

// OnFocus runs f with true when the widget gets keyboard focus and with false when it loses it. A
// container does not report the focus of its children: put the option on the widget that takes keys.
func OnFocus(f func(focused bool)) Option {
	return hook(func(w Widget) error {
		l := focusFunc(f)
		w.control().AddListener(swt.FocusIn, l)
		w.control().AddListener(swt.FocusOut, l)
		return nil
	})
}

type focusFunc func(bool)

func (f focusFunc) HandleEvent(e *swt.Event) { f(e.Type == swt.FocusIn) }

func misfit(opt string, f any, w Widget) error {
	return fmt.Errorf("gowt: %s(%T) does not apply to %s", opt, f, nameOf(w.control()))
}

// OnMouseDown runs f on a Canvas mouse press.
func OnMouseDown(f func(Mouse)) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnMouseDown(f)
			return nil
		}
		return misfit("OnMouseDown", f, w)
	})
}

// OnMouseMove runs f on a Canvas mouse move.
func OnMouseMove(f func(Mouse)) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnMouseMove(f)
			return nil
		}
		return misfit("OnMouseMove", f, w)
	})
}

// OnMouseUp runs f on a Canvas mouse release.
func OnMouseUp(f func(Mouse)) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnMouseUp(f)
			return nil
		}
		return misfit("OnMouseUp", f, w)
	})
}

// OnResize runs f with the Canvas client size; see CanvasView.OnResize.
func OnResize(f func(int, int)) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnResize(f)
			return nil
		}
		return misfit("OnResize", f, w)
	})
}

// OnCellClick runs f on a Table or Tree mouse press.
func OnCellClick[F func(TableClick) | func(TreeClick)](f F) Option {
	return hook(func(w Widget) error {
		switch g := any(f).(type) {
		case func(TableClick):
			if t, ok := w.(*TableView); ok {
				t.OnCellClick(g)
				return nil
			}
		case func(TreeClick):
			if t, ok := w.(*TreeView); ok {
				t.OnCellClick(g)
				return nil
			}
		}
		return misfit("OnCellClick", f, w)
	})
}

// OnExpand runs f when a Tree node expands.
func OnExpand(f func(*Node)) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TreeView); ok {
			t.OnExpand(f)
			return nil
		}
		return misfit("OnExpand", f, w)
	})
}

// OnCollapse runs f when a Tree node collapses.
func OnCollapse(f func(*Node)) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TreeView); ok {
			t.OnCollapse(f)
			return nil
		}
		return misfit("OnCollapse", f, w)
	})
}

// OnCaret runs f when the StyledText caret moves.
func OnCaret(f func(int)) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*StyledTextView); ok {
			t.OnCaret(f)
			return nil
		}
		return misfit("OnCaret", f, w)
	})
}

// OnScroll runs f when a Scroll origin changes; see ScrollView.OnScroll.
func OnScroll(f func(int, int)) Option {
	return hook(func(w Widget) error {
		if s, ok := w.(*ScrollView); ok {
			s.OnScroll(f)
			return nil
		}
		return misfit("OnScroll", f, w)
	})
}

// OnClose runs f when a CTabs close button is pressed; false keeps the tab.
func OnClose(f func(int) bool) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*CTabsView); ok {
			t.OnClose(f)
			return nil
		}
		return misfit("OnClose", f, w)
	})
}

// OnMouseLeave runs f when the pointer leaves a Canvas.
func OnMouseLeave(f func(Mouse)) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnMouseLeave(f)
			return nil
		}
		return misfit("OnMouseLeave", f, w)
	})
}

// OnWheel runs f for a Canvas wheel event; true consumes the event.
func OnWheel(f func(Wheel) bool) Option {
	return hook(func(w Widget) error {
		if c, ok := w.(*CanvasView); ok {
			c.OnWheel(f)
			return nil
		}
		return misfit("OnWheel", f, w)
	})
}
