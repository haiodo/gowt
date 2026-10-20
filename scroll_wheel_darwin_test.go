package gowt

import "github.com/haiodo/gowt/internal/cocoa"

// postWheelDown hands the scroll view a three-line wheel event built from a CGEvent.
func postWheelDown(s *ScrollView) bool { return cocoa.SendScrollWheel(s.s.View, -3) }
