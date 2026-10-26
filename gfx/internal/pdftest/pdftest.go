// Package pdftest holds the PDF checks every gfx backend must pass, so that cg, cairo and the
// Windows writer are held to the same behavior. A backend test calls Run with its renderer.
package pdftest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/pdfcheck"
)

// Options say which text face to draw and which font name the file must then list.
type Options struct {
	Family   string // a family the backend has
	WantFont string // a part of the BaseFont name that must appear; "" skips the check
}

func picture(m gfx.TextMeasurer, family string) *gfx.Picture {
	r := gfx.NewRecorderWith(m)
	r.FillRect(gfx.R(0, 0, 100, 60), gfx.RGB(220, 230, 255))
	r.FillRRect(gfx.R(10, 10, 60, 30), 8, gfx.RGBA(200, 30, 30, 0.7))
	r.StrokePath(gfx.EllipsePath(gfx.R(20, 20, 70, 30)), gfx.StrokeStyle{Width: 2, Dash: []float64{4, 2}}, gfx.Black)
	r.FillGradient(gfx.RectPath(gfx.R(110, 0, 40, 60)), gfx.NewLinearGradient(gfx.Point{X: 110}, gfx.Point{X: 150},
		gfx.Stop{Offset: 0, Color: gfx.RGB(255, 255, 0)}, gfx.Stop{Offset: 1, Color: gfx.RGB(0, 0, 255)}))
	r.Text("Hello PDF", 8, 80, gfx.Font{Family: family, Size: 16}, gfx.Black)
	r.Text("Bold", 100, 80, gfx.Font{Family: family, Size: 16, Weight: 700}, gfx.RGB(0, 90, 0))
	r.BeginLayer(0.5)
	r.FillRect(gfx.R(0, 90, 40, 20), gfx.RGB(255, 0, 0))
	r.FillRect(gfx.R(20, 95, 40, 20), gfx.RGB(0, 0, 255))
	r.EndLayer()
	img := gfx.NewRaster(4, 4)
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+3] = byte(i*8), 128, 255
	}
	r.DrawImage(img, 80, 90, 30, 30)
	return r.Finish()
}

func save(t *testing.T, name string, data []byte) {
	if dir := os.Getenv("GFX_PDF_OUT"); dir != "" {
		os.WriteFile(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+"_"+name+".pdf"), data, 0o644)
	}
}

// Run writes pages of two sizes, then a scene on an A4 page, and reads the files back.
func Run(t *testing.T, r interface {
	gfx.PDFRenderer
	gfx.TextMeasurer
}, o Options) {
	t.Helper()
	pic := picture(r, o.Family)
	items := []gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}}

	t.Run("pages", func(t *testing.T) {
		var buf bytes.Buffer
		doc, err := r.NewPDF(&buf)
		if err != nil {
			t.Fatal(err)
		}
		doc.NewPage(200, 300).Draw(1, gfx.Point{}, items)
		doc.NewPage(612, 792).Draw(2, gfx.Point{X: 10, Y: 10}, items)
		doc.NewPage(100, 100).Draw(1, gfx.Point{}, nil)
		if err := doc.Close(); err != nil {
			t.Fatal(err)
		}
		save(t, "pages", buf.Bytes())
		info, err := pdfcheck.Inspect(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		want := []pdfcheck.Size{{W: 200, H: 300}, {W: 612, H: 792}, {W: 100, H: 100}}
		if len(info.Pages) != len(want) {
			t.Fatalf("pages = %v, want %v", info.Pages, want)
		}
		for i := range want {
			if info.Pages[i] != want[i] {
				t.Errorf("page %d is %v, want %v", i+1, info.Pages[i], want[i])
			}
		}
		if len(info.Fonts) == 0 {
			t.Error("no font in the file: the text was drawn as something else")
		} else if o.WantFont != "" && !strings.Contains(strings.Join(info.Fonts, " "), o.WantFont) {
			t.Errorf("fonts = %v, want one with %q", info.Fonts, o.WantFont)
		}
		if info.Images == 0 {
			t.Error("the raster image is not in the file")
		}
	})

	t.Run("scene on A4", func(t *testing.T) {
		s := gfx.NewScene()
		n := gfx.NewNode(1)
		s.Update(func(tx *gfx.Tx) {
			tx.Add(nil, n)
			tx.SetPicture(n, pic)
		})
		var buf bytes.Buffer
		err := gfx.WriteSubtreePDF(r, &buf, s, nil, gfx.Rect{}, gfx.PDFOptions{PageW: 595, PageH: 842, Margin: 36, Background: gfx.White})
		if err != nil {
			t.Fatal(err)
		}
		save(t, "a4", buf.Bytes())
		info, err := pdfcheck.Inspect(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if len(info.Pages) != 1 || info.Pages[0] != (pdfcheck.Size{W: 595, H: 842}) {
			t.Errorf("pages = %v, want one A4", info.Pages)
		}
	})

	t.Run("page sized to content", func(t *testing.T) {
		var buf bytes.Buffer
		if err := gfx.WritePicturePDF(r, &buf, pic, gfx.PDFOptions{Scale: 2, Margin: 5}); err != nil {
			t.Fatal(err)
		}
		info, err := pdfcheck.Inspect(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		b := pic.Bounds
		want := pdfcheck.Size{W: b.W()*2 + 10, H: b.H()*2 + 10}
		if len(info.Pages) != 1 || abs(info.Pages[0].W-want.W) > 0.01 || abs(info.Pages[0].H-want.H) > 0.01 {
			t.Errorf("pages = %v, want one of %v", info.Pages, want)
		}
	})

	t.Run("empty area is an error", func(t *testing.T) {
		var buf bytes.Buffer
		if err := gfx.WritePicturePDF(r, &buf, gfx.NewRecorder().Finish(), gfx.PDFOptions{}); err == nil {
			t.Error("an empty picture wrote a PDF")
		}
	})
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
