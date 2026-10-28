package look_test

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// A Divider Split as the sidebar: over any shade the glass takes from the content behind it, the
// hairline is darker than the glass in light mode, lighter in dark. GOWT_PIXELS_DIR keeps the PNGs.
func TestSidebarSplitDivider(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, mode := range []struct {
		name string
		look gowt.Appearance
		sign int
	}{{"light", gowt.AppearanceLight, -1}, {"dark", gowt.AppearanceDark, 1}} {
		for _, g := range []uint8{255, 190, 120, 40} {
			t.Run(fmt.Sprint(mode.name, g), func(t *testing.T) {
				path := filepath.Join(dir, fmt.Sprintf("sidebar-divider-%s-%d.png", mode.name, g))
				img := captureSidebarSplit(t, mode.look, gowt.RGB{R: g, G: g, B: g}, path)
				line, glass, ok := hairline(img, mode.sign)
				t.Logf("hairline %d over glass %d", line, glass)
				if !ok || mode.sign*(line-glass) < 8 {
					t.Errorf("no hairline across the sidebar: line %d, glass %d (found %v)", line, glass, ok)
				}
			})
		}
	}
}

// captureSidebarSplit captures a window whose sidebar is a vertical Divider Split of a tree over a
// label and a tree, beside a content of colour bg.
func captureSidebarSplit(t *testing.T, mode gowt.Appearance, bg gowt.RGB, path string) image.Image {
	t.Helper()
	var errs []error
	var shotErr, err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("sidebar divider")
			var split *gowt.SplitView
			var content *gowt.Panel
			w.Add(gowt.Split(gowt.Ref(&split), gowt.Vertical(), gowt.Divider(), gowt.Grow(),
				gowt.Init(func(s *gowt.SplitView) { s.SetWeights(40, 60) }),
				gowt.Tree(gowt.NoBorder(), gowt.Init(func(t *gowt.TreeView) { t.Node("one") })),
				gowt.VBox(gowt.Label("Items"), gowt.Tree(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TreeView) { t.Node("two") })))),
				gowt.VBox(gowt.Ref(&content), gowt.Background(bg)))
			look.SetSidebar(w, split, content, 240)
			w.SetSize(700, 500)
			w.Show()
			app.After(700*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				shotErr = shot.WindowPNG(shotHandle(w), path)
			})
		})
	})
	for _, e := range append(errs, err, shotErr) {
		if e != nil {
			t.Fatal(e)
		}
	}
	return decodePNG(t, path)
}

// hairline returns the row of the sidebar column x=200 that stands out most, in the direction sign,
// from the glass 6 points above and below it; the title strip and rounded corners are left out.
func hairline(img image.Image, sign int) (line, glass int, ok bool) {
	b := img.Bounds()
	scale := float64(b.Dx()) / 700
	x, off := b.Min.X+int(200*scale), int(6*scale)
	green := func(y int) int { _, g, _, _ := img.At(x, y).RGBA(); return int(g >> 8) }
	best := 0
	for y := b.Min.Y + int(80*scale); y < b.Max.Y-int(80*scale); y++ {
		if d := min(sign*(green(y)-green(y-off)), sign*(green(y)-green(y+off))); d > best {
			best, line, glass, ok = d, green(y), green(y-off), true
		}
	}
	return line, glass, ok
}
