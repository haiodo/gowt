//go:build darwin

package cg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/pen"
)

// TestPenRender draws the screens of the .pen file named by PEN_FILE into PNGs in PEN_OUT, for a
// look at the loader's fidelity. Skipped without both variables: the files are private.
func TestPenRender(t *testing.T) {
	path, out := os.Getenv("PEN_FILE"), os.Getenv("PEN_OUT")
	if path == "" || out == "" {
		t.Skip("PEN_FILE and PEN_OUT not set")
	}
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	d, err := pen.Open(path, pen.Options{Measurer: r})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("system font fallback: %v", d.FontFallback())
	for _, in := range d.Screens() {
		s := gfx.NewScene(gfx.R(-100, -100, in.W+200, in.H+200), 256)
		var sc *pen.Screen
		s.Update(func(tx *gfx.Tx) { sc, err = d.Build(tx, nil, in.Index) })
		if err != nil {
			t.Fatal(err)
		}
		s.Commit()
		img, err := gfx.RenderSubtree(r, s, sc.Root, gfx.R(0, 0, in.W, in.H), 1, gfx.Color{R: 1, G: 1, B: 1, A: 1})
		if err != nil {
			t.Fatal(err)
		}
		png, err := img.PNG()
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(out, fmt.Sprintf("%s-%d.png", filepath.Base(path), in.Index))
		if err := os.WriteFile(name, png, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
