package gowt

import (
	"time"

	"github.com/haiodo/gowt/swt"
)

// postWheelDown moves the pointer over the scrolled content and sends a three-line wheel notch
// through SendInput, which Windows delivers to the window under the pointer.
func postWheelDown(s *ScrollView) bool {
	d := s.s.GetDisplay()
	s.s.GetShell().ForceActive()
	b := s.s.GetBounds()
	p := s.s.GetParent().ToDisplay(b.X+b.Width/2, b.Y+b.Height/2)
	move := swt.NewEvent()
	move.Type, move.X, move.Y = swt.MouseMove, p.X, p.Y
	if !d.Post(move) {
		return false
	}
	time.Sleep(50 * time.Millisecond)
	wheel := swt.NewEvent()
	wheel.Type, wheel.Detail, wheel.Count = swt.MouseWheel, swt.SCROLL_LINE, -3
	return d.Post(wheel)
}
