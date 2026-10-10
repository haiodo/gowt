package svg

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Fixtures are Lucide (ISC) and Tabler (MIT) icons; golden PNGs are rendered by this package
// and were checked by eye against rsvg-convert. Regenerate with GOWT_SVG_UPDATE=1.
func TestGolden(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.svg")
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	for _, f := range files {
		data, _ := os.ReadFile(f)
		for _, size := range []int32{24, 48} {
			img, err := rasterizer{}.Rasterize(data, size, size)
			if err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			golden := f[:len(f)-4] + "." + itoa(size) + ".png"
			if os.Getenv("GOWT_SVG_UPDATE") != "" {
				var b bytes.Buffer
				png.Encode(&b, img)
				os.WriteFile(golden, b.Bytes(), 0o644)
				continue
			}
			gf, err := os.Open(golden)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := png.Decode(gf)
			gf.Close()
			if d, maxd := diff(img, want); d != 0 {
				t.Errorf("%s: %d pixels differ by more than %d (max %d)", golden, d, tolerance, maxd)
			}
		}
	}
}

func itoa(n int32) string { return string(rune('0'+n/10)) + string(rune('0'+n%10)) }

// tolerance is per 8-bit channel: amd64 and arm64 round the anti-aliased edge coverage differently
// (arm64 fuses multiply-add), so goldens rendered on one do not match the other bit for bit.
const tolerance = 4

func diff(a *image.NRGBA, b image.Image) (n, maxd int) {
	for y := 0; y < a.Rect.Dy(); y++ {
		for x := 0; x < a.Rect.Dx(); x++ {
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			d := max(absDiff(r1, r2), absDiff(g1, g2), absDiff(b1, b2), absDiff(a1, a2)) >> 8
			maxd = max(maxd, d)
			if d > tolerance {
				n++
			}
		}
	}
	return n, maxd
}

func absDiff(a, b uint32) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func TestStrokeScalesWithTarget(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14"/></svg>`)
	img, _ := rasterizer{}.Rasterize(data, 96, 96)
	// stroke 2 units at 4x = 8 px wide on row 48
	n := 0
	for x := 0; x < 96; x++ {
		if img.NRGBAAt(x, 48).A == 255 {
			n++
		}
	}
	if n < 7 || n > 8 {
		t.Fatalf("stroke is %d px wide, want 8", n)
	}
}

func TestRecolor(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8" fill="currentColor"><rect width="8" height="8"/></svg>`)
	img, _ := rasterizer{}.Rasterize(Recolor(data, "#336699"), 8, 8)
	if c := img.NRGBAAt(4, 4); c.R != 0x33 || c.G != 0x66 || c.B != 0x99 || c.A != 255 {
		t.Fatalf("got %v", c)
	}
}

func TestNormalizePath(t *testing.T) {
	for in, want := range map[string]string{
		"M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0": " M 2 9.5 a 5.5 5.5 0 0 1 9.591 -3.676 a .56 .56 0 0 0 .818 0",
		"a1 1 0 011 1 2 2 0 0 0-1 1":                          " a 1 1 0 0 1 1 1 a 2 2 0 0 0 -1 1",
	} {
		if got := normalizePath(in); got != want {
			t.Errorf("%q\n got %q\nwant %q", in, got, want)
		}
	}
}
