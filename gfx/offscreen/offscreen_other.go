//go:build !darwin && !linux

package offscreen

import (
	"errors"

	"github.com/haiodo/gowt/gfx"
)

// New fails until a backend for this OS lands (Windows: Direct2D).
func New(Options) (gfx.Renderer, error) {
	return nil, errors.New("gfx/offscreen: no headless renderer on this OS yet")
}
