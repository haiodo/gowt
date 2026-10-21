//go:build !darwin

package gowt

// watchRows does nothing: GTK and Win32 scroll frames do not grow with the rows outside the window as
// AppKit's do.
func (s *ScrollView) watchRows() {}
