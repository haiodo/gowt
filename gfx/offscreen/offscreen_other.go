//go:build !darwin && !linux && !windows

package offscreen

import (
	"errors"

	"github.com/haiodo/gowt/gfx"
)

// RegisterFonts fails: there is no backend for this OS.
func RegisterFonts(string) (int, error) { return 0, errNoBackend }

var errNoBackend = errors.New("gfx/offscreen: no headless renderer on this OS yet")

// New fails: there is no backend for this OS.
func New(Options) (gfx.Renderer, error) {
	return nil, errNoBackend
}
