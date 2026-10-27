//go:build !darwin

package shot

import (
	"errors"
	"image"
)

// ErrNoImage means CaptureImage cannot serve this capture; use Capture.
var ErrNoImage = errors.New("shot: no in-memory capture")

// CaptureImage is only there on macOS, where the PNG round trip is the cost of a shot.
func CaptureImage(int64) (*image.NRGBA, Frame, error) { return nil, Frame{}, ErrNoImage }
