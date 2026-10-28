//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// pressSegment clicks segment i as the user does: the toggle flips, then the selection event.
func pressSegment(s *SegmentedView, i int) {
	b := s.seg.buttons[i]
	b.SetSelection(!b.GetSelection())
	notify(b, swt.Selection)
}

// selectedSegment is the button that is on, -1 for none.
func selectedSegment(s *SegmentedView) int {
	for i, b := range s.seg.buttons {
		if b.GetSelection() {
			return i
		}
	}
	return -1
}
