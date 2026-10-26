//go:build linux

package cairo

import (
	"testing"

	"github.com/haiodo/gowt/gfx/internal/fonttest"
)

func TestTracking(t *testing.T) { fonttest.Tracking(t, newRenderer(t, Options{})) }

func TestFaces(t *testing.T) { fonttest.Faces(t, newRenderer(t, Options{}), "DejaVu Sans") }
