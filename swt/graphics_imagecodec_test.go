package swt

import (
	"bytes"
	"embed"
	"testing"

	"github.com/haiodo/gowt/internal/jrt"
)

//go:embed testdata/controlexample
var controlExampleFS embed.FS

func init() { jrt.RegisterResources(controlExampleFS) }

func TestControlExampleImages(t *testing.T) {
	// depth 0 means "don't check" - x/image/bmp's own choice of Paletted vs direct color for a
	// given bit count isn't part of this port's contract, only width/height/transparency are.
	cases := []struct {
		name                 string
		width, height, depth int32
		wantTransparent      bool
	}{
		{"testdata/controlexample/backgroundImage.png", 60, 28, 24, false},
		{"testdata/controlexample/parentBackgroundImage.png", 60, 28, 24, false},
		{"testdata/controlexample/closedFolder.gif", 16, 16, 8, true},
		{"testdata/controlexample/openFolder.gif", 16, 16, 8, true},
		{"testdata/controlexample/target.gif", 16, 16, 8, true},
		{"testdata/controlexample/bold.bmp", 12, 12, 0, false},
		{"testdata/controlexample/bold_mask.bmp", 12, 12, 0, false},
		{"testdata/controlexample/red.bmp", 12, 12, 0, false},
		{"testdata/controlexample/red_mask.bmp", 12, 12, 0, false},
		{"testdata/controlexample/strikeout.bmp", 12, 12, 0, false},
		{"testdata/controlexample/strikeout_mask.bmp", 12, 12, 0, false},
		{"testdata/controlexample/underline.bmp", 12, 12, 0, false},
		{"testdata/controlexample/underline_mask.bmp", 12, 12, 0, false},
		{"testdata/controlexample/italic.bmp", 12, 12, 0, false},
		{"testdata/controlexample/italic_mask.bmp", 12, 12, 0, false},
		{"testdata/controlexample/yellow.bmp", 12, 12, 0, false},
		{"testdata/controlexample/yellow_mask.bmp", 12, 12, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stream := jrt.GetResourceAsStream(c.name)
			if stream == nil {
				t.Fatalf("resource not found: %s", c.name)
			}
			img := NewImageDataStream(stream)
			if img.Width != c.width || img.Height != c.height {
				t.Errorf("got %dx%d, want %dx%d", img.Width, img.Height, c.width, c.height)
			}
			if c.depth != 0 && img.Depth != c.depth {
				t.Errorf("got depth %d, want %d", img.Depth, c.depth)
			}
			if got := img.TransparentPixel != -1 || len(img.AlphaData) > 0; got != c.wantTransparent {
				t.Errorf("got transparent=%v, want %v", got, c.wantTransparent)
			}
		})
	}
}

// TestPNGRoundTrip covers ImageLoader.save/load through the same stdlib-backed codec.
func TestPNGRoundTrip(t *testing.T) {
	stream := jrt.GetResourceAsStream("testdata/controlexample/backgroundImage.png")
	original := NewImageDataStream(stream)

	loader := NewImageLoader()
	loader.Data = []*ImageData{original}
	var buf bytes.Buffer
	loader.Save(jrt.NewOutputStream(&buf), IMAGE_PNG)

	roundTripped := NewImageLoader()
	roundTripped.Load(jrt.NewInputStream(bytes.NewReader(buf.Bytes())))
	if len(roundTripped.Data) != 1 {
		t.Fatalf("got %d frames, want 1", len(roundTripped.Data))
	}
	got := roundTripped.Data[0]
	if got.Width != original.Width || got.Height != original.Height {
		t.Errorf("got %dx%d, want %dx%d", got.Width, got.Height, original.Width, original.Height)
	}
	for y := int32(0); y < original.Height; y++ {
		for x := int32(0); x < original.Width; x++ {
			if got.GetPixel(x, y) != original.GetPixel(x, y) {
				t.Fatalf("pixel (%d,%d) changed across the round trip: got %06x, want %06x",
					x, y, got.GetPixel(x, y), original.GetPixel(x, y))
			}
		}
	}
}
