//go:build !darwin

package shot

// PlaceOnRetina is for macOS only.
func PlaceOnRetina(int64) bool { return false }
