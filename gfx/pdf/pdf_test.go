package pdf_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/internal/pdftest"
	"github.com/haiodo/gowt/gfx/pdf"
	"github.com/haiodo/gowt/internal/pdfcheck"
)

type goFonts struct{}

func (goFonts) FontFile(f gfx.Font) ([]byte, bool) {
	if f.Family == "none" {
		return nil, false
	}
	if f.Weight >= 600 {
		return gobold.TTF, true
	}
	return goregular.TTF, true
}

// sample records a picture that uses every kind of command the writer knows.
func sample() *gfx.Picture {
	r := gfx.NewRecorderWith(gfx.EstimateText{})
	r.FillRect(gfx.R(10, 10, 100, 50), gfx.RGB(200, 30, 30))
	r.FillRRect(gfx.R(20, 20, 40, 20), 6, gfx.RGBA(0, 0, 255, 0.5))
	r.StrokePath(gfx.EllipsePath(gfx.R(30, 70, 60, 40)), gfx.StrokeStyle{Width: 2, Dash: []float64{4, 2}, Cap: gfx.CapRound}, gfx.Black)
	r.FillGradient(gfx.RectPath(gfx.R(120, 10, 60, 60)), gfx.NewLinearGradient(gfx.Point{X: 120}, gfx.Point{X: 180}, gfx.Stop{Offset: 0, Color: gfx.RGB(255, 255, 0)}, gfx.Stop{Offset: 0.5, Color: gfx.RGB(0, 255, 0)}, gfx.Stop{Offset: 1, Color: gfx.RGB(0, 0, 255)}))
	r.FillGradient(gfx.EllipsePath(gfx.R(120, 80, 60, 40)), gfx.NewRadialGradient(gfx.Point{X: 150, Y: 100}, 0, gfx.Point{X: 150, Y: 100}, 30, gfx.Stop{Offset: 0, Color: gfx.White}, gfx.Stop{Offset: 1, Color: gfx.Black}))
	r.Save()
	r.ClipRect(gfx.R(0, 120, 80, 20))
	r.Text("Hello (PDF) \\ text", 4, 135, gfx.Font{Size: 14}, gfx.Black)
	r.Restore()
	r.Text("Bold", 100, 135, gfx.Font{Size: 14, Weight: 700, Spacing: 1}, gfx.RGB(0, 90, 0))
	r.BeginLayer(0.4)
	r.FillRect(gfx.R(10, 150, 60, 30), gfx.RGB(255, 0, 0))
	r.FillRect(gfx.R(40, 160, 60, 30), gfx.RGB(0, 0, 255))
	r.EndLayer()
	r.FillShadowRRect(120, 150, 50, 30, 5, gfx.White, gfx.Shadow{DX: 2, DY: 3, Blur: 4, Color: gfx.RGBA(0, 0, 0, 0.5)})
	img := gfx.NewRaster(2, 2)
	copy(img.Pix, []byte{255, 0, 0, 255, 0, 255, 0, 128, 0, 0, 255, 255, 255, 255, 255, 0})
	r.DrawImage(img, 10, 200, 40, 40)
	return r.Finish()
}

func write(t *testing.T, fonts pdf.Fonts, pages [][2]float64, p *gfx.Picture) []byte {
	t.Helper()
	var buf bytes.Buffer
	d := pdf.New(&buf, fonts)
	for _, sz := range pages {
		d.NewPage(sz[0], sz[1]).Draw(1.5, gfx.Point{X: -5, Y: -5}, []gfx.Item{{Pic: p, M: gfx.Identity(), Alpha: 1}})
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GFX_PDF_OUT"); dir != "" {
		os.WriteFile(dir+"/"+strings.ReplaceAll(t.Name(), "/", "_")+".pdf", buf.Bytes(), 0o644)
	}
	return buf.Bytes()
}

func TestPagesSizesFontsImages(t *testing.T) {
	data := write(t, goFonts{}, [][2]float64{{300, 400}, {612, 792}}, sample())
	info, err := pdfcheck.Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Pages) != 2 || info.Pages[0] != (pdfcheck.Size{W: 300, H: 400}) || info.Pages[1] != (pdfcheck.Size{W: 612, H: 792}) {
		t.Fatalf("pages = %v", info.Pages)
	}
	if len(info.Fonts) != 2 {
		t.Fatalf("fonts = %v", info.Fonts)
	}
	for _, want := range []string{"GoRegular", "Go-Bold"} {
		if !strings.Contains(strings.Join(info.Fonts, " "), want) {
			t.Errorf("font %s not in %v", want, info.Fonts)
		}
	}
	if info.Images != 2 { // the picture and its soft mask
		t.Errorf("images = %d, want 2", info.Images)
	}
	if !strings.Contains(info.Text, "/ShadingType 2") || !strings.Contains(info.Text, "/ShadingType 3") {
		t.Error("gradients are not shading patterns")
	}
	if !strings.Contains(info.Text, "/Group << /S /Transparency") {
		t.Error("the layer is not a transparency group")
	}
}

func TestXrefOffsets(t *testing.T) {
	data := write(t, goFonts{}, [][2]float64{{200, 200}}, sample())
	s := string(data)
	start, err := strconv.Atoi(strings.TrimSpace(s[strings.LastIndex(s, "startxref")+len("startxref") : strings.LastIndex(s, "%%EOF")]))
	if err != nil || !strings.HasPrefix(s[start:], "xref\n0 ") {
		t.Fatalf("startxref %d does not point at the table", start)
	}
	re := regexp.MustCompile(`(\d{10}) 00000 n \n`)
	for i, m := range re.FindAllStringSubmatch(s[start:], -1) {
		off, _ := strconv.Atoi(m[1])
		if want := strconv.Itoa(i+1) + " 0 obj\n"; !strings.HasPrefix(s[off:], want) {
			t.Fatalf("object %d: offset %d holds %.12q", i+1, off, s[off:])
		}
	}
}

func TestStandardFontFallback(t *testing.T) {
	data := write(t, nil, [][2]float64{{200, 200}}, sample())
	info, err := pdfcheck.Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(info.Fonts, " "); !strings.Contains(got, "Helvetica") || !strings.Contains(got, "Helvetica-Bold") {
		t.Errorf("fonts = %q, want Helvetica and Helvetica-Bold", got)
	}
	if !strings.Contains(info.Text, `(Hello \(PDF\) \\ text)`) {
		t.Error("text is not escaped as a PDF string")
	}
}

func TestUnknownFamilyFallsBack(t *testing.T) {
	r := gfx.NewRecorderWith(gfx.EstimateText{})
	r.Text("x", 0, 10, gfx.Font{Size: 10, Family: "none"}, gfx.Black)
	data := write(t, goFonts{}, [][2]float64{{50, 50}}, r.Finish())
	info, _ := pdfcheck.Inspect(data)
	if len(info.Fonts) != 1 || info.Fonts[0] != "Helvetica" {
		t.Errorf("fonts = %v", info.Fonts)
	}
}

func TestCopyableText(t *testing.T) {
	r := gfx.NewRecorderWith(gfx.EstimateText{})
	r.Text("Привет", 0, 10, gfx.Font{Size: 10}, gfx.Black)
	data := write(t, goFonts{}, [][2]float64{{50, 50}}, r.Finish())
	info, _ := pdfcheck.Inspect(data)
	if !strings.Contains(info.Text, "beginbfchar") || !strings.Contains(info.Text, "041F") { // 'П'
		t.Errorf("no ToUnicode entry for the Cyrillic text")
	}
}

type goRenderer struct{ gfx.EstimateText }

func (goRenderer) NewPDF(w io.Writer) (gfx.Document, error) { return pdf.New(w, goFonts{}), nil }

func TestSharedChecks(t *testing.T) {
	pdftest.Run(t, goRenderer{}, pdftest.Options{Family: "x", WantFont: "Go"})
}
