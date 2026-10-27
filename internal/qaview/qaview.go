// Package qaview is how a control that paints on its own clock (a gfx view) meets the QA driver and
// the capture commands without either importing the other: the driver waits for its frames and hands
// it scroll, pinch and pointer events. Everything here runs on the UI thread.
package qaview

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"sync"

	"github.com/haiodo/gowt/swt"
)

// InputKey is the data key a control registers its Input under.
const InputKey = "gowt.qa.input"

// Mods are the keys held during a gesture.
type Mods struct{ Shift, Ctrl, Alt, Cmd bool }

// Input is what a control takes through InputKey. x and y are in the control's coordinates. Phase and
// momentum are "", "mayBegin", "began", "changed", "ended" or "cancelled"; "" is a mouse wheel.
type Input interface {
	// Scroll delivers one scroll event as the control's own router gets it.
	Scroll(x, y, dx, dy float64, precise bool, phase, momentum string, m Mods) error
	// Magnify delivers one pinch step: magnification is the zoom change, 0.1 for 10 percent more.
	Magnify(x, y, magnification float64, phase string, m Mods) error
	// Pointer delivers one mouse event: kind is "down", "move" or "up". On a move, button is the
	// button held, 0 for a hover.
	Pointer(kind string, x, y float64, button, count int, m Mods) error
}

// Source is a control that presents frames on its own clock.
type Source interface {
	// Request asks for a frame of the control when it is visible in shell; nil when it is not. The
	// returned done reports whether a frame came after the request and nothing is left to apply.
	Request(shell *swt.Shell) (done func() bool)
}

var (
	mu      sync.Mutex
	sources = map[Source]struct{}{}
)

// Add registers s and returns the function that removes it.
func Add(s Source) (remove func()) {
	mu.Lock()
	sources[s] = struct{}{}
	mu.Unlock()
	return func() { mu.Lock(); delete(sources, s); mu.Unlock() }
}

// Request asks every source in shell for a frame. The returned done is true once all of them
// presented one; it is true at once when there is no source.
func Request(shell *swt.Shell) (done func() bool) {
	mu.Lock()
	all := make([]Source, 0, len(sources))
	for s := range sources {
		all = append(all, s)
	}
	mu.Unlock()
	var dones []func() bool
	for _, s := range all {
		if d := s.Request(shell); d != nil {
			dones = append(dones, d)
		}
	}
	return func() bool {
		for _, d := range dones {
			if !d() {
				return false
			}
		}
		return true
	}
}

// Painter is a Source whose pixels a window capture can miss (macOS: CALayer content is not in the
// view capture). Overlay draws them into img, the capture of the shell's frame view, in place.
type Painter interface {
	Overlay(shell *swt.Shell, img *image.NRGBA) error
}

// OverlayPNG lets the painters of shell draw into the PNG that WindowPNG wrote at path. It does
// nothing when no source paints, and with GOWT_SNAP_SCREEN=1, where the window server's own picture
// has the layers.
func OverlayPNG(shell *swt.Shell, path string) error {
	if os.Getenv("GOWT_SNAP_SCREEN") == "1" {
		return nil
	}
	if len(painters()) == 0 {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	img := image.NewNRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, src.Bounds().Min, draw.Src)
	if err := OverlayImage(shell, img); err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func painters() []Painter {
	mu.Lock()
	defer mu.Unlock()
	var ps []Painter
	for s := range sources {
		if p, ok := s.(Painter); ok {
			ps = append(ps, p)
		}
	}
	return ps
}

// OverlayImage lets the painters of shell draw into img, the capture of its frame view, in place.
func OverlayImage(shell *swt.Shell, img *image.NRGBA) error {
	for _, p := range painters() {
		if err := p.Overlay(shell, img); err != nil {
			return err
		}
	}
	return nil
}

// PopoverKey is the data key of the shell of a Popover.
const PopoverKey = "gowt.qa.popover"

// Popover is what a popover's shell tells the QA driver. On macOS the shell is never shown (its content
// lives in an NSPopover), so the driver cannot go by the shell's own visibility.
type Popover interface {
	// OnScreen reports whether the popover is shown.
	OnScreen() bool
	// Owner is the window the popover belongs to.
	Owner() *swt.Shell
	// Close hides the popover as a click outside would.
	Close()
	// Opening reports a popover that was shown and is not on screen yet.
	Opening() bool
	// Escape is what Esc in the popover does: OnEscape, which may keep it open, then Close.
	Escape()
	// Bounds is the popover's content on the screen, y down.
	Bounds() (x, y, w, h int)
}

var popovers = map[*swt.Shell]struct{}{}

// AddPopover registers the shell of a popover and returns the function that removes it. Display.GetShells
// does not list a popover shell on macOS while its window holds a placeholder, so the driver looks here.
func AddPopover(s *swt.Shell) (remove func()) {
	mu.Lock()
	popovers[s] = struct{}{}
	mu.Unlock()
	return func() { mu.Lock(); delete(popovers, s); mu.Unlock() }
}

// PopoverOpening reports whether a registered popover is still opening, which a settle waits for.
func PopoverOpening() bool {
	mu.Lock()
	var all []*swt.Shell
	for s := range popovers {
		all = append(all, s)
	}
	mu.Unlock()
	for _, s := range all {
		if p, ok := PopoverOf(s); ok && !s.IsDisposed() && p.Opening() {
			return true
		}
	}
	return false
}

// PopoverShells lists the registered popover shells that are on screen.
func PopoverShells() []*swt.Shell {
	mu.Lock()
	var all []*swt.Shell
	for s := range popovers {
		all = append(all, s)
	}
	mu.Unlock()
	var out []*swt.Shell
	for _, s := range all {
		if !s.IsDisposed() && OnScreen(s) {
			out = append(out, s)
		}
	}
	return out
}

// PopoverOf returns the popover whose shell s is, shown or not.
func PopoverOf(s *swt.Shell) (Popover, bool) {
	p, ok := s.GetDataKey(PopoverKey).(Popover)
	return p, ok
}

// OnScreen reports whether s is a popover that is shown.
func OnScreen(s *swt.Shell) bool {
	p, ok := PopoverOf(s)
	return ok && p.OnScreen()
}

// Popovers returns the shells of the popovers on screen that belong to the window owner.
func Popovers(d *swt.Display, owner *swt.Shell) []*swt.Shell {
	var out []*swt.Shell
	for _, s := range PopoverShells() {
		if p, _ := PopoverOf(s); p.Owner() == owner {
			out = append(out, s)
		}
	}
	return out
}

// ToolbarSearchKey is the data key of a shell whose native toolbar has a search field (macOS
// NSSearchToolbarItem), which is no swt control and so is not in the control tree.
const ToolbarSearchKey = "gowt.qa.toolbarsearch"

// ToolbarSearch is what such a shell tells the QA driver.
type ToolbarSearch interface {
	// Present reports whether the toolbar has the field now; a rebuilt toolbar has none for a moment.
	Present() bool
	ID() string
	Placeholder() string
	Text() string
	// SetText replaces the text without running the search.
	SetText(string)
	Focus() bool
	Focused() bool
	// Submit runs the search with the text, as Return does.
	Submit()
	// Collapse folds an empty field back into its button, as Esc does.
	Collapse()
}

// ToolbarSearchOf returns the toolbar search field of s.
func ToolbarSearchOf(s *swt.Shell) (ToolbarSearch, bool) {
	t, ok := s.GetDataKey(ToolbarSearchKey).(ToolbarSearch)
	return t, ok
}

// ToolbarKey is the data key of a shell with a native toolbar, whose items are no swt controls.
const ToolbarKey = "gowt.qa.toolbar"

// ToolbarButton is a push item or a group segment of a native toolbar. Bounds is in the coordinates
// of the shell's client area, all 0 where the toolbar shows no frame for it.
type ToolbarButton struct {
	ID, Title string
	Enabled   bool
	Bounds    [4]int
}

// Toolbar is what such a shell tells the QA driver.
type Toolbar interface {
	Buttons() []ToolbarButton
	// Click runs the handler of button n of Buttons.
	Click(n int) bool
}

// ToolbarOf returns the native toolbar of s.
func ToolbarOf(s *swt.Shell) (Toolbar, bool) {
	t, ok := s.GetDataKey(ToolbarKey).(Toolbar)
	return t, ok
}
