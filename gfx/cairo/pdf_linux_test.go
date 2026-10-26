//go:build linux

package cairo

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/internal/pdftest"
	"github.com/haiodo/gowt/internal/pdfcheck"
)

func TestPDFShared(t *testing.T) {
	pdftest.Run(t, newRenderer(t, Options{}), pdftest.Options{Family: "DejaVu Sans", WantFont: "DejaVu"})
}

func TestPDFTextIsText(t *testing.T) {
	r := newRenderer(t, Options{})
	rec := gfx.NewRecorderWith(r)
	rec.Text("Привет, PDF", 10, 30, gfx.Font{Family: "DejaVu Sans", Size: 20}, gfx.Black)
	var buf bytes.Buffer
	if err := gfx.WritePicturePDF(r, &buf, rec.Finish(), gfx.PDFOptions{Margin: 10}); err != nil {
		t.Fatal(err)
	}
	info, err := pdfcheck.Inspect(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(info.Fonts, " "), "DejaVuSans") {
		t.Errorf("fonts = %v, want DejaVu Sans embedded", info.Fonts)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestPDFWriteError(t *testing.T) {
	r := newRenderer(t, Options{})
	rec := gfx.NewRecorderWith(gfx.EstimateText{})
	rec.FillRect(gfx.R(0, 0, 50, 50), gfx.Black)
	err := gfx.WritePicturePDF(r, failWriter{}, rec.Finish(), gfx.PDFOptions{})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Errorf("err = %v, want the writer's error", err)
	}
}
