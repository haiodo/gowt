package svg_test

import (
	_ "embed"

	"github.com/haiodo/gowt/svg"
)

//go:embed testdata/lucide-heart.svg
var icon []byte

// A monochrome icon recolored for the current theme; the provider draws it at every zoom.
func ExampleNewImageDataProvider() {
	p := svg.NewImageDataProvider(icon, 16, 16, "#e0e0e0")
	_ = p // pass to swt.NewImageDeviceImageDataProvider
}
