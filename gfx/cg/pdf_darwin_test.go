//go:build darwin

package cg

import (
	"bytes"
	"math"
	"os"
	"strings"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/internal/pdftest"
	"github.com/haiodo/gowt/gfx/pdf"
	"github.com/haiodo/gowt/internal/pdfcheck"
)

// shapesPicture draws without text, so the three renderers can be compared pixel by pixel.
func shapesPicture() *gfx.Picture {
	r := gfx.NewRecorderWith(gfx.EstimateText{})
	r.FillRect(gfx.R(10, 10, 100, 50), gfx.RGB(200, 30, 30))
	r.FillRRect(gfx.R(20, 20, 40, 20), 6, gfx.RGBA(0, 0, 255, 0.5))
	r.StrokePath(gfx.EllipsePath(gfx.R(30, 70, 60, 40)), gfx.StrokeStyle{Width: 3, Dash: []float64{6, 3}, Cap: gfx.CapRound}, gfx.Black)
	r.FillGradient(gfx.RectPath(gfx.R(120, 10, 60, 60)), gfx.NewLinearGradient(gfx.Point{X: 120}, gfx.Point{X: 180},
		gfx.Stop{Offset: 0, Color: gfx.RGB(255, 255, 0)}, gfx.Stop{Offset: 0.5, Color: gfx.RGB(0, 255, 0)}, gfx.Stop{Offset: 1, Color: gfx.RGB(0, 0, 255)}))
	r.FillGradient(gfx.EllipsePath(gfx.R(120, 80, 60, 40)), gfx.NewRadialGradient(gfx.Point{X: 150, Y: 100}, 0, gfx.Point{X: 150, Y: 100}, 30,
		gfx.Stop{Offset: 0, Color: gfx.White}, gfx.Stop{Offset: 1, Color: gfx.Black}))
	r.Save()
	r.Translate(60, 130)
	r.Rotate(0.3)
	r.ClipRRect(gfx.R(0, 0, 50, 30), 8)
	r.FillRect(gfx.R(-10, -10, 80, 60), gfx.RGB(0, 120, 120))
	r.Restore()
	r.BeginLayer(0.4)
	r.FillRect(gfx.R(10, 150, 60, 30), gfx.RGB(255, 0, 0))
	r.FillRect(gfx.R(40, 160, 60, 30), gfx.RGB(0, 0, 255))
	r.EndLayer()
	img := gfx.NewRaster(2, 2)
	copy(img.Pix, []byte{255, 0, 0, 255, 0, 255, 0, 128, 0, 0, 255, 255, 255, 255, 255, 0})
	r.DrawImage(img, 120, 150, 40, 40)
	return r.Finish()
}

var (
	pdfDocFromProvider func(p uintptr) uintptr
	pdfProviderCreate  func(info uintptr, data unsafe.Pointer, size int, release uintptr) uintptr
	pdfGetPage         func(doc uintptr, n int) uintptr
	pdfPageCount       func(doc uintptr) int
	pdfDrawPage        func(ctx, page uintptr)
	pdfBox             func(page uintptr, box int32) cgRect
	pdfDocRelease      func(doc uintptr)
	pdfProviderRelease func(p uintptr)
)

// renderPDF draws the pages' first page with CoreGraphics at scale 1 into a Raster.
func renderPDF(t *testing.T, data []byte, w, h int) *gfx.Raster {
	t.Helper()
	bind()
	if pdfDocFromProvider == nil {
		reg := func(fn any, name string) { purego.RegisterLibFunc(fn, purego.RTLD_DEFAULT, name) }
		reg(&pdfDocFromProvider, "CGPDFDocumentCreateWithProvider")
		reg(&pdfProviderCreate, "CGDataProviderCreateWithData")
		reg(&pdfGetPage, "CGPDFDocumentGetPage")
		reg(&pdfPageCount, "CGPDFDocumentGetNumberOfPages")
		reg(&pdfDrawPage, "CGContextDrawPDFPage")
		reg(&pdfBox, "CGPDFPageGetBoxRect")
		reg(&pdfDocRelease, "CGPDFDocumentRelease")
		reg(&pdfProviderRelease, "CGDataProviderRelease")
	}
	prov := pdfProviderCreate(0, unsafe.Pointer(&data[0]), len(data), 0)
	defer pdfProviderRelease(prov)
	doc := pdfDocFromProvider(prov)
	if doc == 0 {
		t.Fatal("CoreGraphics cannot read the PDF")
	}
	defer pdfDocRelease(doc)
	if pdfPageCount(doc) < 1 {
		t.Fatal("no page")
	}
	r := NewRenderer(Options{})
	s := r.NewSurface(w, h)
	defer s.Close()
	cgFillColor(s.ctx, 1, 1, 1, 1)
	cgFillRect(s.ctx, cgRect{0, 0, float64(w), float64(h)})
	pdfDrawPage(s.ctx, pdfGetPage(doc, 1)) // media box origin is 0,0 and y is up, like the bitmap
	return gfx.FromBGRA(s.Pixels(), w, h)
}

// diff is the share of pixels where any channel differs by more than tol.
func diff(a, b *gfx.Raster, tol int) float64 {
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			if math.Abs(float64(a.Pix[i+c])-float64(b.Pix[i+c])) > float64(tol) {
				n++
				break
			}
		}
	}
	return float64(n) / float64(a.W*a.H)
}

func TestPDFPagesFontsAndPixels(t *testing.T) {
	r := NewRenderer(Options{})
	pic := shapesPicture()
	const w, h = 200, 220
	// the reference: the picture on a white bitmap
	s := r.NewSurface(w, h)
	defer s.Close()
	s.Begin(1, 0, 0, gfx.White)
	s.Replay(pic, 0, 0)
	want := gfx.FromBGRA(s.Pixels(), w, h)

	items := []gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}}
	var viaCG, viaGo bytes.Buffer
	doc, err := r.NewPDF(&viaCG)
	if err != nil {
		t.Fatal(err)
	}
	doc.NewPage(w, h).Draw(1, gfx.Point{}, items)
	doc.NewPage(300, 400).Draw(2, gfx.Point{}, items)
	if err := doc.Close(); err != nil {
		t.Fatal(err)
	}
	gd := pdf.New(&viaGo, nil)
	gd.NewPage(w, h).Draw(1, gfx.Point{}, items)
	if err := gd.Close(); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GFX_PDF_OUT"); dir != "" {
		os.WriteFile(dir+"/cg.pdf", viaCG.Bytes(), 0o644)
		os.WriteFile(dir+"/gopdf.pdf", viaGo.Bytes(), 0o644)
	}

	info, err := pdfcheck.Inspect(viaCG.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Pages) != 2 || info.Pages[0] != (pdfcheck.Size{W: w, H: h}) || info.Pages[1] != (pdfcheck.Size{W: 300, H: 400}) {
		t.Errorf("pages = %v", info.Pages)
	}
	// CoreGraphics may paint a hair differently at edges and in gradients: 12% of the pixels may
	// differ by more than 24 levels (the dashed ellipse and the image edges), not a shifted shape.
	if d := diff(want, renderPDF(t, viaCG.Bytes(), w, h), 24); d > 0.12 {
		t.Errorf("CoreGraphics PDF differs from the bitmap in %.1f%% of the pixels", d*100)
	}
	if d := diff(want, renderPDF(t, viaGo.Bytes(), w, h), 24); d > 0.12 {
		t.Errorf("gfx/pdf output differs from the bitmap in %.1f%% of the pixels", d*100)
	}
}

func TestPDFTextIsText(t *testing.T) {
	r := NewRenderer(Options{})
	rec := gfx.NewRecorderWith(r)
	rec.Text("Hello PDF", 10, 30, gfx.Font{Name: "Helvetica Neue", Size: 20}, gfx.Black)
	rec.Text("Bold", 10, 60, gfx.Font{Family: "Helvetica Neue", Weight: 700, Size: 20}, gfx.Black)
	var buf bytes.Buffer
	if err := gfx.WritePicturePDF(r, &buf, rec.Finish(), gfx.PDFOptions{Margin: 10}); err != nil {
		t.Fatal(err)
	}
	info, err := pdfcheck.Inspect(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Pages) != 1 {
		t.Fatalf("pages = %v", info.Pages)
	}
	if len(info.Fonts) == 0 || !strings.Contains(strings.Join(info.Fonts, " "), "Helvetica") {
		t.Errorf("fonts = %v, want the embedded Helvetica", info.Fonts)
	}
}

func TestPDFShared(t *testing.T) {
	pdftest.Run(t, NewRenderer(Options{}), pdftest.Options{Family: "Helvetica Neue", WantFont: "Helvetica"})
}
