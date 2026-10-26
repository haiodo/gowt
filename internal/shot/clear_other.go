//go:build !darwin

package shot

// ClearFocus is for macOS only.
func ClearFocus(int64) {}
