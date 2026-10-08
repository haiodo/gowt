package gowt

// Backdrop is the material behind a window's content.
type Backdrop int

const (
	BackdropNone        Backdrop = iota // opaque, the platform default
	BackdropTranslucent                 // blurred desktop behind the window: NSVisualEffectView, Acrylic
	BackdropGlass                       // Liquid Glass on macOS 26, Mica on Windows 11
)

// SetBackdrop sets the window material. A value the OS cannot show falls back to the nearest
// one it can; Linux ignores it.
func (w *Window) SetBackdrop(b Backdrop) { w.setBackdrop(b) }
