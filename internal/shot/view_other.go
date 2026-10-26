//go:build !darwin

package shot

// CaptureView is Capture where the window holds the popover's own pixels.
func CaptureView(handle int64, path string) (Frame, error) { return Capture(handle, path) }
