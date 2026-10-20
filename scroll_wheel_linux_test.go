package gowt

import "github.com/haiodo/gowt/internal/gtk"

// postWheelDown queues a wheel notch over the scrolled content, as the X server would deliver it.
func postWheelDown(s *ScrollView) bool {
	gtk.PostWheel(s.body.c.Handle, true)
	return true
}
