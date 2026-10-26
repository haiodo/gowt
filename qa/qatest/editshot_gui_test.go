package qatest_test

import (
	"image/png"
	"os"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

func darkPixels(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, _ := img.At(x, y).RGBA(); r>>8 < 80 && g>>8 < 80 && bl>>8 < 80 {
				n++
			}
		}
	}
	return n
}

// An Edit field over the scene is in the shot with the text typed into it: it replaces red pixels of
// the scene and adds dark ones (the glyphs).
func TestGfxShotShowsEditField(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	before := app.Shot("edit-before")
	app.State("edit-open")
	app.Settle()
	field := app.Find(qa.ID("shot-edit"))
	field.Type("MMMMMMMM")
	after := app.Shot("edit-after")
	b := field.Node().Bounds
	if lost := redPixels(t, before) - redPixels(t, after); lost < b[2]*b[3]/2 {
		t.Errorf("the field %v hides %d red pixels of the scene, want at least half its area", b, lost)
	}
	if glyphs := darkPixels(t, after) - darkPixels(t, before); glyphs < 30 {
		t.Errorf("the typed text adds %d dark pixels, want at least 30", glyphs)
	}
}
