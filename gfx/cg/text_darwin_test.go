//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/internal/fonttest"
)

func TestTracking(t *testing.T) { fonttest.Tracking(t, NewRenderer(Options{TextCache: true})) }

func TestFaces(t *testing.T) {
	fonttest.Faces(t, NewRenderer(Options{TextCache: true}), "Helvetica Neue")
}

func TestFamilyMatchesPostScriptName(t *testing.T) {
	r := NewRenderer(Options{})
	w := func(f gfx.Font) float64 {
		f.Size = 40
		return r.MeasureText("Hamburgefonstiv", f).Width
	}
	for _, c := range []struct {
		f    gfx.Font
		name string
	}{
		{gfx.Font{Family: "Helvetica Neue", Weight: 700}, "HelveticaNeue-Bold"},
		{gfx.Font{Family: "Helvetica Neue", Weight: 400}, "HelveticaNeue"},
		{gfx.Font{Family: "Helvetica Neue", Weight: 500}, "HelveticaNeue-Medium"},
		{gfx.Font{Family: "Helvetica Neue", Weight: 400, Italic: true}, "HelveticaNeue-Italic"},
		{gfx.Font{Family: "Helvetica Neue", Weight: 700, Italic: true}, "HelveticaNeue-BoldItalic"},
	} {
		if got, want := w(c.f), w(gfx.Font{Name: c.name}); got != want {
			t.Errorf("%+v measures %v, %s measures %v", c.f, got, c.name, want)
		}
	}
}
