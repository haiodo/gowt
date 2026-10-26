package d2d

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/gfx/internal/fonttest"
)

func newTestRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer(Options{TextCache: true, FontDir: os.Getenv("GFX_FONT_DIR")})
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(r.Close)
	return r
}

func TestTracking(t *testing.T) { fonttest.Tracking(t, newTestRenderer(t)) }

func TestFaces(t *testing.T) { fonttest.Faces(t, newTestRenderer(t), "Segoe UI") }
