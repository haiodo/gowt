package gowt

import "github.com/haiodo/gowt/swt"

// SetAppName sets the name the system shows for the process (macOS menu bar and Dock, Linux
// WM class). Call it before Run.
func SetAppName(name string) { swt.DisplaySetAppName(name) }

// Background sets the background color of a widget.
func Background(c RGB) Option {
	return Option{apply: func(w *swt.Control) { w.SetBackgroundWithColor(c.color()) }}
}

// SetTitle changes the window title.
func (w *Window) SetTitle(s string) { w.shell.SetText(s) }

// SetItem replaces the text of item i, keeping the selection.
func (l *List) SetItem(i int, s string) { l.l.SetItem(int32(i), s) }

// panel is the name under which Window, Group, Split and CoolBar embed Panel. An exported
// embedded field called Panel would hide the Panel constructor method.
type panel = Panel
