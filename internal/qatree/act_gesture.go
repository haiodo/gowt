package qatree

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/haiodo/gowt/internal/qaview"
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
