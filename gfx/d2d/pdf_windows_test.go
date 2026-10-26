//go:build windows

package d2d

import (
	"testing"

	"github.com/haiodo/gowt/gfx/internal/pdftest"
)

func TestPDFShared(t *testing.T) {
	r, err := NewRenderer(Options{})
	if err != nil {
		t.Skip(err)
	}
	defer r.Close()
	pdftest.Run(t, r, pdftest.Options{Family: "Segoe UI", WantFont: "Segoe"})
}
