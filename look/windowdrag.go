package look

import "github.com/haiodo/gowt/swt"

// DragsWindow makes c a window drag area for a custom title bar (see SetFullSizeContent): a left
// press on c or on its labels and plain panels moves the window as a title bar would, and a double
// click zooms (macOS follows the AppleActionOnDoubleClick setting, including minimize and none) or maximizes and restores.
// On Windows and Linux the double click is a plain maximize toggle, not the window manager's own title bar action, and the move
// itself is left to the window manager, so details such as edge snapping differ between desktops.
// Buttons, text fields and other controls, and Canvas, keep their clicks. Children are taken as they
// are when DragsWindow runs, so call it after building c. macOS, Windows and Linux (X11 and Wayland through GTK 3).
func DragsWindow(c Container) {
	comp := c.AsComposite()
	sh := comp.GetShell()
	var walk func(*swt.Control)
	walk = func(w *swt.Control) {
		switch v := w.Impl().(type) {
		case *swt.Label:
		case *swt.Group:
			for _, ch := range v.GetChildren() {
				walk(ch)
			}
		case *swt.Composite:
			for _, ch := range v.GetChildren() {
				walk(ch)
			}
		default:
			return
		}
		w.AddMouseListener(swt.MouseListenerMouseDownAdapter(func(e *swt.MouseEvent) {
			if e.Button != 1 {
				return
			}
			if e.Count >= 2 {
				doubleClick(sh)
				return
			}
			beginMove(sh, w, e)
		}))
	}
	walk(&comp.Control)
}

// toggleMaximized is the double click action where the OS has no setting of its own to read.
func toggleMaximized(s *swt.Shell) { s.SetMaximized(!s.GetMaximized()) }
