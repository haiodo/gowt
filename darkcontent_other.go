//go:build !windows

package gowt

// SetDarkContent is a no-op here: GTK and AppKit theme the content themselves.
func (a *App) SetDarkContent(bool) {}
