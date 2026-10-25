package gowt

import "fmt"

// Primary marks a push Button as the main action of its window: it is the window's default button
// (Enter presses it) and has the system's prominent look: macOS the accent default button, GTK
// "suggested-action", Windows the WinUI accent button. The last Primary of a window is its default.
func Primary() Option {
	return hook(func(w Widget) error {
		b, ok := pushButton(w)
		if !ok {
			return fmt.Errorf("gowt: Primary applies to a push Button, not to %s", nameOf(w.control()))
		}
		if s := b.GetShell(); s != nil {
			s.SetDefaultButton(b)
		}
		setButtonKind(b, buttonPrimary)
		return nil
	})
}

// Destructive marks a push Button whose action deletes or discards: macOS hasDestructiveAction (red),
// GTK "destructive-action" (red fill), Windows a standard button. With Primary it is also the
// default button, as for "Delete" in a confirmation.
func Destructive() Option {
	return hook(func(w Widget) error {
		b, ok := pushButton(w)
		if !ok {
			return fmt.Errorf("gowt: Destructive applies to a push Button, not to %s", nameOf(w.control()))
		}
		setButtonKind(b, buttonDestructive)
		return nil
	})
}

const (
	buttonPrimary = 1 << iota
	buttonDestructive
)
