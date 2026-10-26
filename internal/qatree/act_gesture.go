package qatree

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/swt"
)

// gesture hands a scroll or pinch event to the control's own input router (a gfx view registers one
// under qaview.InputKey); no OS event is involved.
func gesture(e *entry, action string, args []byte) error {
	c, err := controlOf(e)
	if err != nil {
		return err
	}
	in, ok := c.GetDataKey(qaview.InputKey).(qaview.Input)
	if !ok {
		return fmt.Errorf("qatree: %s: %s %s takes no scroll or pinch events (only a gfx view does)", action, e.n.Class, e.n.Path)
	}
	switch action {
	case "scrollevent":
		var a struct {
			X, Y, DX, DY    float64
			Precise         bool
			Phase, Momentum string
			Mods            string
		}
		if err := parse(args, &a); err != nil {
			return err
		}
		m, err := parseMods(a.Mods)
		if err != nil {
			return err
		}
		return in.Scroll(a.X, a.Y, a.DX, a.DY, a.Precise, a.Phase, a.Momentum, m)
	default:
		var a struct {
			X, Y          float64
			Magnification float64
			Phase         string
			Mods          string
		}
		if err := parse(args, &a); err != nil {
			return err
		}
		m, err := parseMods(a.Mods)
		if err != nil {
			return err
		}
		return in.Magnify(a.X, a.Y, a.Magnification, a.Phase, m)
	}
}

// parseMods reads "Shift+Mod": the modifiers of keyEvent without a key.
func parseMods(s string) (m qaview.Mods, err error) {
	if s == "" {
		return m, nil
	}
	for _, p := range strings.Split(s, "+") {
		switch strings.ToLower(p) {
		case "mod":
			if runtime.GOOS == "darwin" {
				m.Cmd = true
			} else {
				m.Ctrl = true
			}
		case "cmd":
			if runtime.GOOS != "darwin" {
				return m, fmt.Errorf("qatree: mods %q: Cmd exists on macOS only, use Mod", s)
			}
			m.Cmd = true
		case "ctrl":
			m.Ctrl = true
		case "alt":
			m.Alt = true
		case "shift":
			m.Shift = true
		default:
			return m, fmt.Errorf("qatree: mods %q: unknown modifier %q, want Mod, Cmd, Ctrl, Alt or Shift", s, p)
		}
	}
	return m, nil
}

// pointer delivers mouse events at a point of the control: through the router of a gfx view, else as
// the listener events MouseDown, MouseMove, MouseUp and MouseDoubleClick of the control. A press
// focuses the control first, as a click does, but without activating the window.
func pointer(d *swt.Display, e *entry, args []byte) error {
	c, err := controlOf(e)
	if err != nil {
		return err
	}
	var a struct {
		Kind     string
		X, Y     float64
		Button   int
		Count    int
		Mods     string
		ToX, ToY float64
		Steps    int
	}
	if err := parse(args, &a); err != nil {
		return err
	}
	m, err := parseMods(a.Mods)
	if err != nil {
		return err
	}
	if a.Button == 0 {
		a.Button = 1
	}
	if a.Button < 1 || a.Button > 3 {
		return fmt.Errorf("qatree: pointer: button %d, want 1, 2 or 3", a.Button)
	}
	count := max(a.Count, 1)
	deliver := func(kind string, x, y float64, button, count int) error {
		return notifyPointer(c, kind, x, y, button, count, m)
	}
	if in, ok := c.GetDataKey(qaview.InputKey).(qaview.Input); ok {
		deliver = func(kind string, x, y float64, button, count int) error {
			return in.Pointer(kind, x, y, button, count, m)
		}
	}
	press := func(count int) error {
		ensureFocus(d, c)
		return deliver("down", a.X, a.Y, a.Button, count)
	}
	release := func() error { return deliver("up", a.X, a.Y, a.Button, 0) }
	var steps []func() error
	switch a.Kind {
	case "down":
		steps = append(steps, func() error { return press(count) })
	case "up":
		steps = append(steps, release)
	case "move":
		steps = append(steps, func() error { return deliver("move", a.X, a.Y, a.Button, 0) })
	case "click":
		steps = append(steps, func() error { return press(count) }, release)
	case "doubleclick":
		steps = append(steps, func() error { return press(1) }, release, func() error { return press(2) }, release)
	case "drag":
		n := a.Steps
		if n <= 0 {
			n = 4
		}
		steps = append(steps, func() error { return press(count) })
		for i := 1; i <= n; i++ {
			f := float64(i) / float64(n)
			x, y := a.X+(a.ToX-a.X)*f, a.Y+(a.ToY-a.Y)*f
			steps = append(steps, func() error { return deliver("move", x, y, a.Button, 0) })
		}
		steps = append(steps, func() error { return deliver("up", a.ToX, a.ToY, a.Button, 0) })
	default:
		return fmt.Errorf("qatree: pointer: kind %q, want down, move, up, click, doubleclick or drag", a.Kind)
	}
	for _, f := range steps {
		if err := f(); err != nil {
			return err
		}
		if c.IsDisposed() {
			break
		}
	}
	if a.Button == 3 && (a.Kind == "down" || a.Kind == "click" || a.Kind == "doubleclick") && !c.IsDisposed() {
		// The popup menu opens on the press; QA shows it to the app (OnShow) without a native menu.
		if m := c.GetMenu(); m != nil && m.GetStyle()&swt.POP_UP != 0 {
			showPopup(m)
		}
	}
	flush(d)
	return nil
}

// notifyPointer is the semantic delivery to a control that has no input router: the listener events
// of a mouse press, move and release.
func notifyPointer(c *swt.Control, kind string, x, y float64, button, count int, m qaview.Mods) error {
	var mask int32
	if m.Shift {
		mask |= swt.SHIFT
	}
	if m.Ctrl {
		mask |= swt.CTRL
	}
	if m.Alt {
		mask |= swt.ALT
	}
	if m.Cmd {
		mask |= swt.COMMAND
	}
	held := [...]int32{0, swt.BUTTON1, swt.BUTTON2, swt.BUTTON3}[button]
	ev := swt.NewEvent()
	ev.X, ev.Y, ev.Button, ev.Count, ev.StateMask = int32(x), int32(y), int32(button), int32(count), mask
	switch kind {
	case "down":
		c.NotifyListeners(swt.MouseDown, ev)
		if count == 2 && !c.IsDisposed() {
			dc := *ev
			c.NotifyListeners(swt.MouseDoubleClick, &dc)
		}
	case "move":
		ev.Button, ev.Count, ev.StateMask = 0, 0, mask|held
		c.NotifyListeners(swt.MouseMove, ev)
	case "up":
		ev.Count, ev.StateMask = 1, mask|held
		c.NotifyListeners(swt.MouseUp, ev)
	}
	return nil
}
