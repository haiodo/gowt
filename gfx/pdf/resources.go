package pdf

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/haiodo/gowt/gfx"
)

// resources is the one resource dictionary every page and form of a document shares, which keeps
// the names unique and the group forms free of a dictionary of their own.
type resources struct {
	faces   map[faceKey]*face
	fonts   []*face
	xobjs   []named
	images  map[*gfx.Raster]string
	gs      map[int]string
	gsList  []named
	pats    []named
	nX, nP  int
	nG, nFm int
}

type named struct {
	name string
	obj  int
	body string // inline dictionary for an ExtGState
}

func (r *resources) init() {
	r.faces = map[faceKey]*face{}
	r.images = map[*gfx.Raster]string{}
	r.gs = map[int]string{}
}

// alpha returns the name of the state dictionary with constant fill and stroke alpha a.
func (r *resources) alpha(a float64) string {
	k := int(math.Round(math.Min(math.Max(a, 0), 1) * 1000))
	if n, ok := r.gs[k]; ok {
		return n
	}
	r.nG++
	n := fmt.Sprintf("G%d", r.nG)
	r.gs[k] = n
	r.gsList = append(r.gsList, named{name: n, body: fmt.Sprintf("<< /ca %s /CA %s >>", num(float64(k)/1000), num(float64(k)/1000))})
	return n
}

func (r *resources) xobject(obj int, prefix string) string {
	r.nX++
	n := fmt.Sprintf("%s%d", prefix, r.nX)
	r.xobjs = append(r.xobjs, named{name: n, obj: obj})
	return n
}

// image embeds a Raster once: 8-bit RGB with a soft mask when any pixel is not opaque.
func (r *resources) image(d *Document, img *gfx.Raster) string {
	if n, ok := r.images[img]; ok {
		return n
	}
	rgb := make([]byte, 0, img.W*img.H*3)
	var alpha []byte
	opaque := true
	for i := 0; i+3 < len(img.Pix) && i/4 < img.W*img.H; i += 4 {
		rgb = append(rgb, img.Pix[i], img.Pix[i+1], img.Pix[i+2])
		if img.Pix[i+3] != 255 {
			opaque = false
		}
	}
	dict := fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Interpolate true", img.W, img.H)
	if !opaque {
		alpha = make([]byte, 0, img.W*img.H)
		for i := 3; i < len(img.Pix) && len(alpha) < img.W*img.H; i += 4 {
			alpha = append(alpha, img.Pix[i])
		}
		mask := d.stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Interpolate true", img.W, img.H), alpha, true)
		dict += fmt.Sprintf(" /SMask %d 0 R", mask)
	}
	n := r.xobject(d.stream(dict, rgb, true), "Im")
	r.images[img] = n
	return n
}

// pattern makes a shading pattern for g in the space the current matrix ctm leads to. alpha is the
// constant opacity to paint it with.
func (r *resources) pattern(d *Document, g *gfx.Gradient, ctm gfx.Affine) (name string, alpha float64, ok bool) {
	if len(g.Stops) == 0 {
		return "", 0, false
	}
	alpha = g.Stops[0].Color.A
	for _, s := range g.Stops {
		if s.Color.A != alpha {
			alpha = 1
			break
		}
	}
	stops := append([]gfx.Stop(nil), g.Stops...)
	for i := range stops {
		stops[i].Offset = math.Min(math.Max(stops[i].Offset, 0), 1)
		if i > 0 && stops[i].Offset <= stops[i-1].Offset {
			stops[i].Offset = math.Min(stops[i-1].Offset+1e-6, 1)
		}
	}
	if stops[0].Offset > 0 {
		stops = append([]gfx.Stop{{Offset: 0, Color: stops[0].Color}}, stops...)
	}
	if last := stops[len(stops)-1]; last.Offset < 1 {
		stops = append(stops, gfx.Stop{Offset: 1, Color: last.Color})
	}
	fn := func(a, b gfx.Color) string {
		return fmt.Sprintf("<< /FunctionType 2 /Domain [0 1] /C0 [%s %s %s] /C1 [%s %s %s] /N 1 >>", num(a.R), num(a.G), num(a.B), num(b.R), num(b.G), num(b.B))
	}
	var function string
	if len(stops) == 2 {
		function = fn(stops[0].Color, stops[1].Color)
	} else {
		var fns, bounds, enc strings.Builder
		for i := 0; i+1 < len(stops); i++ {
			fns.WriteString(fn(stops[i].Color, stops[i+1].Color) + " ")
			enc.WriteString("0 1 ")
			if i > 0 {
				bounds.WriteString(num(stops[i].Offset) + " ")
			}
		}
		function = fmt.Sprintf("<< /FunctionType 3 /Domain [0 1] /Functions [%s] /Bounds [%s] /Encode [%s] >>", fns.String(), bounds.String(), enc.String())
	}
	var shading string
	if g.Kind == gfx.RadialGradient {
		shading = fmt.Sprintf("/ShadingType 3 /Coords [%s %s %s %s %s %s]", num(g.P0.X), num(g.P0.Y), num(g.R0), num(g.P1.X), num(g.P1.Y), num(g.R1))
	} else {
		shading = fmt.Sprintf("/ShadingType 2 /Coords [%s %s %s %s]", num(g.P0.X), num(g.P0.Y), num(g.P1.X), num(g.P1.Y))
	}
	obj := d.add(fmt.Sprintf("<< /Type /Pattern /PatternType 2 /Matrix [%s %s %s %s %s %s] /Shading << %s /ColorSpace /DeviceRGB /Extend [true true] /Function %s >> >>",
		num(ctm[0]), num(ctm[1]), num(ctm[2]), num(ctm[3]), num(ctm[4]), num(ctm[5]), shading, function))
	r.nP++
	name = fmt.Sprintf("P%d", r.nP)
	r.pats = append(r.pats, named{name: name, obj: obj})
	return name, alpha, true
}

// write finishes the fonts and fills the shared resource object.
func (r *resources) write(d *Document) {
	var b bytes.Buffer
	b.WriteString("<<")
	if len(r.fonts) > 0 {
		b.WriteString(" /Font <<")
		for _, f := range r.fonts {
			fmt.Fprintf(&b, " /%s %d 0 R", f.name, f.finish(d))
		}
		b.WriteString(" >>")
	}
	if len(r.xobjs) > 0 {
		b.WriteString(" /XObject <<")
		for _, x := range r.xobjs {
			fmt.Fprintf(&b, " /%s %d 0 R", x.name, x.obj)
		}
		b.WriteString(" >>")
	}
	if len(r.gsList) > 0 {
		b.WriteString(" /ExtGState <<")
		for _, g := range r.gsList {
			fmt.Fprintf(&b, " /%s %s", g.name, g.body)
		}
		b.WriteString(" >>")
	}
	if len(r.pats) > 0 {
		b.WriteString(" /Pattern <<")
		for _, p := range r.pats {
			fmt.Fprintf(&b, " /%s %d 0 R", p.name, p.obj)
		}
		b.WriteString(" >>")
	}
	b.WriteString(" >>")
	d.set(resObj, b.String())
}
