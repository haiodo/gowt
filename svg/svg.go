// Package svg is gowt's optional SVG rasterizer (the counterpart of SWT's org.eclipse.swt.svg
// fragment): importing it for its side effect makes swt.Image load SVG at every zoom.
// It draws with oksvg+rasterx (pure Go): paths, basic shapes, gradients, strokes; no text, filters or masks.
package svg

import (
	"bytes"
	"errors"
	"image"
	"regexp"
	"strings"

	"github.com/haiodo/gowt/swt"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func init() { swt.SetSVGRasterizer(rasterizer{}) }

type rasterizer struct{}

// Recolor sets what currentColor means in a monochrome icon (Lucide, Tabler); color is #rrggbb.
func Recolor(svg []byte, color string) []byte {
	return bytes.ReplaceAll(svg, []byte("currentColor"), []byte(color))
}

// NewImageDataProvider is the zoom-aware source of an icon logically width x height: every zoom is
// drawn from the vector at width*zoom/100 pixels. For a new theme colour build a new provider.
func NewImageDataProvider(svg []byte, width, height int32, color string) swt.ImageDataProvider {
	svg = Recolor(svg, color)
	return &provider{svg, width, height}
}

type provider struct {
	svg           []byte
	width, height int32
}

func (p *provider) GetImageData(zoom int32) *swt.ImageData {
	img, err := rasterizer{}.Rasterize(p.svg, p.width*zoom/100, p.height*zoom/100)
	if err != nil {
		return nil
	}
	return swt.NewImageDataFromNRGBA(img)
}

// oksvg cannot read currentColor itself and takes #000 when the document has none.
func parse(svg []byte) (*oksvg.SvgIcon, error) {
	svg = Recolor(svg, "#000000")
	svg = fixPathData(svg)
	return oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.IgnoreErrorMode)
}

var pathD = regexp.MustCompile(`(\sd=")([^"]*)"`)

func fixPathData(svg []byte) []byte {
	return pathD.ReplaceAllFunc(svg, func(m []byte) []byte {
		sub := pathD.FindSubmatch(m)
		return []byte(string(sub[1]) + normalizePath(string(sub[2])) + `"`)
	})
}

// normalizePath rewrites path data into what oksvg parses right: numbers glued as "3.676.56" are
// split, flags written as "011" are split, and an arc with repeated parameter sets becomes one arc
// per set (oksvg draws every set after the first from the first set's radii).
func normalizePath(d string) string {
	var out strings.Builder
	i := 0
	number := func() (string, bool) {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
			i++
		}
		start := i
		if i < len(d) && (d[i] == '-' || d[i] == '+') {
			i++
		}
		dot := false
		for i < len(d) {
			c := d[i]
			if c >= '0' && c <= '9' {
				i++
			} else if c == '.' && !dot {
				dot = true
				i++
			} else if (c == 'e' || c == 'E') && i > start && i+1 < len(d) && (d[i+1] == '-' || d[i+1] == '+' || d[i+1] >= '0' && d[i+1] <= '9') {
				i += 2
			} else {
				break
			}
		}
		return d[start:i], i > start
	}
	flag := func() (string, bool) {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
			i++
		}
		if i < len(d) && (d[i] == '0' || d[i] == '1') {
			i++
			return d[i-1 : i], true
		}
		return "", false
	}
	for i < len(d) {
		c := d[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') || c == 'e' {
			i++
			continue
		}
		i++
		out.WriteString(" " + string(c))
		if c != 'a' && c != 'A' {
			for {
				n, ok := number()
				if !ok {
					break
				}
				out.WriteString(" " + n)
			}
			continue
		}
		for first := true; ; first = false {
			var set [7]string
			ok := true
			for k := 0; k < 7 && ok; k++ {
				if k == 3 || k == 4 {
					set[k], ok = flag()
				} else {
					set[k], ok = number()
				}
			}
			if !ok {
				break
			}
			if !first {
				out.WriteString(" " + string(c))
			}
			out.WriteString(" " + strings.Join(set[:], " "))
		}
	}
	return out.String()
}

func (rasterizer) Size(svg []byte) (float64, float64, error) {
	ic, err := parse(svg)
	if err != nil {
		return 0, 0, err
	}
	return ic.ViewBox.W, ic.ViewBox.H, nil
}

func (rasterizer) Rasterize(svg []byte, w, h int32) (*image.NRGBA, error) {
	ic, err := parse(svg)
	if err != nil {
		return nil, err
	}
	if ic.ViewBox.W <= 0 || ic.ViewBox.H <= 0 {
		return nil, errors.New("svg: no width/height or viewBox")
	}
	ic.SetTarget(0, 0, float64(w), float64(h))
	// oksvg draws stroke widths in document units, not scaled by the target transform.
	sx := float64(w) / ic.ViewBox.W
	for i := range ic.SVGPaths {
		ic.SVGPaths[i].LineWidth *= sx
		ic.SVGPaths[i].DashOffset *= sx
		for j := range ic.SVGPaths[i].Dash {
			ic.SVGPaths[i].Dash[j] *= sx
		}
	}
	rgba := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	sc := rasterx.NewScannerGV(int(w), int(h), rgba, rgba.Bounds())
	ic.Draw(rasterx.NewDasher(int(w), int(h), sc), 1)
	return unpremultiply(rgba), nil
}

func unpremultiply(src *image.RGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Rect)
	for i := 0; i < len(src.Pix); i += 4 {
		a := uint32(src.Pix[i+3])
		dst.Pix[i+3] = byte(a)
		if a == 0 {
			continue
		}
		for c := 0; c < 3; c++ {
			dst.Pix[i+c] = byte(min(255, (uint32(src.Pix[i+c])*255+a/2)/a))
		}
	}
	return dst
}
