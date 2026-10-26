package gfx

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"math"
	"strconv"
	"strings"
)

// SVG returns p as the d attribute of an SVG path.
func (p *Path) SVG() string {
	var b bytes.Buffer
	pts := p.Pts
	for _, v := range p.Verbs {
		switch v {
		case MoveTo:
			fmt.Fprintf(&b, "M%s %s", num(pts[0].X), num(pts[0].Y))
			pts = pts[1:]
		case LineTo:
			fmt.Fprintf(&b, "L%s %s", num(pts[0].X), num(pts[0].Y))
			pts = pts[1:]
		case QuadTo:
			fmt.Fprintf(&b, "Q%s %s %s %s", num(pts[0].X), num(pts[0].Y), num(pts[1].X), num(pts[1].Y))
			pts = pts[2:]
		case CubicTo:
			fmt.Fprintf(&b, "C%s %s %s %s %s %s", num(pts[0].X), num(pts[0].Y), num(pts[1].X), num(pts[1].Y), num(pts[2].X), num(pts[2].Y))
			pts = pts[3:]
		case ClosePath:
			b.WriteByte('Z')
		}
	}
	return b.String()
}

// fillRule is the SVG attribute for p, empty for the default.
func (p *Path) fillRule(attr string) string {
	if p.EvenOdd {
		return " " + attr + `="evenodd"`
	}
	return ""
}

func num(v float64) string {
	if math.Abs(v) < 5e-5 {
		return "0"
	}
	return strconv.FormatFloat(math.Round(v*1000)/1000, 'f', -1, 64)
}

func rgb(c Color) string {
	return fmt.Sprintf("rgb(%d,%d,%d)", q8(c.R), q8(c.G), q8(c.B))
}

func matrix(m Affine) string {
	return fmt.Sprintf("matrix(%s %s %s %s %s %s)", num(m[0]), num(m[1]), num(m[2]), num(m[3]), num(m[4]), num(m[5]))
}

type svgWriter struct {
	defs, body bytes.Buffer
	n          int
}

func (w *svgWriter) id(prefix string) string {
	w.n++
	return prefix + strconv.Itoa(w.n)
}

func (w *svgWriter) clip(m Affine, kind ClipKind, r Rect, radius float64, p *Path) string {
	id := w.id("c")
	var shape string
	switch kind {
	case ClipRect:
		shape = fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" transform="%s"/>`, num(r.Min.X), num(r.Min.Y), num(r.W()), num(r.H()), matrix(m))
	case ClipRRect:
		shape = fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" transform="%s"/>`, num(r.Min.X), num(r.Min.Y), num(r.W()), num(r.H()), num(radius), matrix(m))
	default:
		shape = fmt.Sprintf(`<path d="%s"%s transform="%s"/>`, p.SVG(), p.fillRule("clip-rule"), matrix(m))
	}
	fmt.Fprintf(&w.defs, `<clipPath id="%s">%s</clipPath>`, id, shape)
	return id
}

func (w *svgWriter) paint(c Color, g *Gradient, attr string) string {
	if g == nil {
		s := fmt.Sprintf(`%s="%s"`, attr, rgb(c))
		if c.A < 1 {
			s += fmt.Sprintf(` %s-opacity="%s"`, attr, num(c.A))
		}
		return s
	}
	id := w.id("g")
	if g.Kind == LinearGradient {
		fmt.Fprintf(&w.defs, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s">`, id, num(g.P0.X), num(g.P0.Y), num(g.P1.X), num(g.P1.Y))
	} else {
		fmt.Fprintf(&w.defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s" fx="%s" fy="%s" fr="%s">`, id, num(g.P1.X), num(g.P1.Y), num(g.R1), num(g.P0.X), num(g.P0.Y), num(g.R0))
	}
	for _, s := range g.Stops {
		fmt.Fprintf(&w.defs, `<stop offset="%s" stop-color="%s" stop-opacity="%s"/>`, num(s.Offset), rgb(s.Color), num(s.Color.A))
	}
	if g.Kind == LinearGradient {
		w.defs.WriteString("</linearGradient>")
	} else {
		w.defs.WriteString("</radialGradient>")
	}
	return fmt.Sprintf(`%s="url(#%s)"`, attr, id)
}

func (w *svgWriter) stroke(st *StrokeStyle) string {
	s := fmt.Sprintf(` stroke-width="%s"`, num(st.Width))
	switch st.Cap {
	case CapRound:
		s += ` stroke-linecap="round"`
	case CapSquare:
		s += ` stroke-linecap="square"`
	}
	switch st.Join {
	case JoinRound:
		s += ` stroke-linejoin="round"`
	case JoinBevel:
		s += ` stroke-linejoin="bevel"`
	default:
		limit := st.MiterLimit
		if limit <= 0 {
			limit = 10
		}
		s += fmt.Sprintf(` stroke-miterlimit="%s"`, num(limit))
	}
	if len(st.Dash) > 0 {
		s += ` stroke-dasharray="`
		for i, d := range st.Dash {
			if i > 0 {
				s += " "
			}
			s += num(d)
		}
		s += `"`
		if st.DashPhase != 0 {
			s += fmt.Sprintf(` stroke-dashoffset="%s"`, num(st.DashPhase))
		}
	}
	return s
}

type svgGroup byte

const (
	grpSave svgGroup = iota
	grpPlain
	grpLayer
)

// picture writes the commands of p; the groups it opens are closed before it returns.
func (w *svgWriter) picture(p *Picture) {
	var stack []svgGroup
	open := func(g svgGroup, tag string) {
		w.body.WriteString(tag)
		stack = append(stack, g)
	}
	var it Iter
	it.Reset(p)
	for it.Next() {
		o := it.Op()
		switch o.Kind {
		case OpSave:
			stack = append(stack, grpSave)
		case OpRestore:
			for len(stack) > 0 {
				g := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if g == grpSave {
					break
				}
				w.body.WriteString("</g>")
			}
		case OpTranslate:
			open(grpPlain, fmt.Sprintf(`<g transform="translate(%s %s)">`, num(o.A), num(o.B)))
		case OpScale:
			open(grpPlain, fmt.Sprintf(`<g transform="scale(%s %s)">`, num(o.A), num(o.B)))
		case OpRotate:
			open(grpPlain, fmt.Sprintf(`<g transform="rotate(%s)">`, num(o.A*180/math.Pi)))
		case OpTransform:
			open(grpPlain, fmt.Sprintf(`<g transform="%s">`, matrix(o.M)))
		case OpClipRect:
			open(grpPlain, fmt.Sprintf(`<g clip-path="url(#%s)">`, w.clip(Identity(), ClipRect, R(o.A, o.B, o.C, o.D), 0, nil)))
		case OpClipRRect:
			open(grpPlain, fmt.Sprintf(`<g clip-path="url(#%s)">`, w.clip(Identity(), ClipRRect, R(o.A, o.B, o.C, o.D), o.E, nil)))
		case OpClipPath:
			open(grpPlain, fmt.Sprintf(`<g clip-path="url(#%s)">`, w.clip(Identity(), ClipPath, Rect{}, 0, o.Path)))
		case OpBeginLayer:
			open(grpLayer, fmt.Sprintf(`<g opacity="%s">`, num(o.A)))
		case OpEndLayer:
			if n := len(stack); n > 0 && stack[n-1] == grpLayer {
				stack = stack[:n-1]
				w.body.WriteString("</g>")
			}
		case OpFill:
			fmt.Fprintf(&w.body, `<path d="%s"%s %s/>`, o.Path.SVG(), o.Path.fillRule("fill-rule"), w.paint(o.Color, nil, "fill"))
		case OpFillGrad:
			fmt.Fprintf(&w.body, `<path d="%s"%s %s/>`, o.Path.SVG(), o.Path.fillRule("fill-rule"), w.paint(Color{}, o.Grad, "fill"))
		case OpStroke:
			fmt.Fprintf(&w.body, `<path d="%s" fill="none" %s%s/>`, o.Path.SVG(), w.paint(o.Color, nil, "stroke"), w.stroke(o.Style))
		case OpStrokeGrad:
			fmt.Fprintf(&w.body, `<path d="%s" fill="none" %s%s/>`, o.Path.SVG(), w.paint(Color{}, o.Grad, "stroke"), w.stroke(o.Style))
		case OpShadowBox:
			w.shadow(o)
		case OpFillShadow:
			w.shadowPath(o.Path, 0, o.Shadow, "fill", o.Path.fillRule("fill-rule"), "")
			fmt.Fprintf(&w.body, `<path d="%s"%s %s/>`, o.Path.SVG(), o.Path.fillRule("fill-rule"), w.paint(o.Color, nil, "fill"))
		case OpStrokeShadow:
			w.shadowPath(o.Path, o.Style.Width/2*strokeReach(o.Path, *o.Style), o.Shadow, "stroke", ` fill="none"`, w.stroke(o.Style))
			fmt.Fprintf(&w.body, `<path d="%s" fill="none" %s%s/>`, o.Path.SVG(), w.paint(o.Color, nil, "stroke"), w.stroke(o.Style))
		case OpText, OpTextShadow:
			if o.Kind == OpTextShadow && o.Shadow.Color.A > 0 {
				id := w.blur(o.Shadow.Blur, "")
				fmt.Fprintf(&w.body, `<text x="%s" y="%s" transform="translate(%s %s)"%s font-size="%s"%s %s filter="url(#%s)">%s</text>`,
					num(o.A), num(o.B), num(o.Shadow.DX), num(o.Shadow.DY), svgFont(o.Font), num(o.Font.Size), svgSpacing(o.Font), w.paint(o.Shadow.Color, nil, "fill"), id, html.EscapeString(o.Text))
			}
			fmt.Fprintf(&w.body, `<text x="%s" y="%s"%s font-size="%s"%s %s>%s</text>`,
				num(o.A), num(o.B), svgFont(o.Font), num(o.Font.Size), svgSpacing(o.Font), w.paint(o.Color, nil, "fill"), html.EscapeString(o.Text))
		case OpImage:
			if r, ok := o.Image.(*Raster); ok {
				if data, err := r.PNG(); err == nil {
					fmt.Fprintf(&w.body, `<image x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="none" href="data:image/png;base64,%s"/>`,
						num(o.A), num(o.B), num(o.C), num(o.D), base64.StdEncoding.EncodeToString(data))
				}
			}
		}
	}
	for _, g := range stack {
		if g != grpSave {
			w.body.WriteString("</g>")
		}
	}
}

// blur defines a Gaussian blur filter and returns its id. CoreGraphics' blur is about twice the
// standard deviation. region is the filter region in user space; empty means the element box grown
// by half its size on each side, which a shape with a zero-height box (a horizontal line) cannot use.
func (w *svgWriter) blur(blur float64, region string) string {
	id := w.id("f")
	if region == "" {
		region = `x="-50%" y="-50%" width="200%" height="200%"`
	}
	fmt.Fprintf(&w.defs, `<filter id="%s" %s><feGaussianBlur stdDeviation="%s"/></filter>`, id, region, num(blur/2))
	return id
}

// shadowPath writes the blurred, offset copy of p in the shadow color; grow is how far the shape
// reaches past the bounds of p. kind is the paint attribute, extra the attributes after it.
func (w *svgWriter) shadowPath(p *Path, grow float64, sh Shadow, kind, rule, stroke string) {
	if sh.Color.A <= 0 {
		return
	}
	b := p.Bounds().Expand(grow + 1.5*sh.Blur)
	id := w.blur(sh.Blur, fmt.Sprintf(`filterUnits="userSpaceOnUse" x="%s" y="%s" width="%s" height="%s"`, num(b.Min.X), num(b.Min.Y), num(b.W()), num(b.H())))
	fmt.Fprintf(&w.body, `<path d="%s" transform="translate(%s %s)"%s %s%s filter="url(#%s)"/>`,
		p.SVG(), num(sh.DX), num(sh.DY), rule, w.paint(sh.Color, nil, kind), stroke, id)
}

func (w *svgWriter) shadow(o *Op) {
	sh := o.Shadow
	if sh.Color.A > 0 {
		id := w.blur(sh.Blur, "")
		fmt.Fprintf(&w.body, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" %s filter="url(#%s)"/>`,
			num(o.A+sh.DX), num(o.B+sh.DY), num(o.C), num(o.D), num(o.E), w.paint(sh.Color, nil, "fill"), id)
	}
	fmt.Fprintf(&w.body, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" %s/>`, num(o.A), num(o.B), num(o.C), num(o.D), num(o.E), w.paint(o.Color, nil, "fill"))
}

func (w *svgWriter) finish(out io.Writer, area Rect, scale float64) error {
	if noArea(area) {
		area = Rect{}
	}
	_, err := fmt.Fprintf(out, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="%s %s %s %s"><defs>%s</defs>%s</svg>`,
		num(math.Ceil(area.W()*scale)), num(math.Ceil(area.H()*scale)), num(area.Min.X), num(area.Min.Y), num(area.W()), num(area.H()), w.defs.String(), w.body.String())
	return err
}

// WriteSVG writes p as an SVG document sized to its bounds times scale.
func (p *Picture) WriteSVG(out io.Writer, scale float64) error {
	var w svgWriter
	w.picture(p)
	return w.finish(out, p.Bounds, scale)
}

// WriteSubtreeSVG writes the subtree of root, as RenderSubtree draws it, as an SVG document; a nil
// root writes the whole scene. area is the scene rectangle to write, or the zero Rect for the
// subtree bounds.
func WriteSubtreeSVG(out io.Writer, s *Scene, root *Node, area Rect, scale float64) error {
	return WriteSubtreeSVGWith(out, s, root, area, scale, SVGOptions{})
}

// SVGOptions are the extras of WriteSubtreeSVGWith. The zero value is WriteSubtreeSVG.
type SVGOptions struct {
	// Background, when its A is above zero, is filled over the whole area behind the content.
	Background Color
}

// WriteSubtreeSVGWith is WriteSubtreeSVG with opt.
func WriteSubtreeSVGWith(out io.Writer, s *Scene, root *Node, area Rect, scale float64, opt SVGOptions) error {
	s.Commit()
	if root == nil {
		root = s.Root()
	}
	if noArea(area) {
		s.Read(func() { area = root.SubtreeBounds() })
	}
	var b Items
	s.CollectSubtree(root, area, &b)
	var w svgWriter
	if opt.Background.A > 0 {
		fmt.Fprintf(&w.body, `<rect x="%s" y="%s" width="%s" height="%s" %s/>`,
			num(area.Min.X), num(area.Min.Y), num(area.W()), num(area.H()), w.paint(opt.Background, nil, "fill"))
	}
	for _, it := range b.List {
		var closing int
		for _, c := range it.Clips {
			fmt.Fprintf(&w.body, `<g clip-path="url(#%s)">`, w.clip(c.M, c.Kind, c.Rect, c.Radius, c.Path))
			closing++
		}
		fmt.Fprintf(&w.body, `<g transform="%s"`, matrix(it.M))
		if it.Alpha < 1 {
			fmt.Fprintf(&w.body, ` opacity="%s"`, num(it.Alpha))
		}
		w.body.WriteString(">")
		w.picture(it.Pic)
		w.body.WriteString("</g>")
		for ; closing > 0; closing-- {
			w.body.WriteString("</g>")
		}
	}
	return w.finish(out, area, scale)
}

// svgDefaultFamily stands for the default sans face of the backends.
const svgDefaultFamily = "Helvetica Neue, Helvetica, Arial, sans-serif"

// svgFont is the family, weight and style attributes of f; a Name is a face name and leaves the
// weight and style to it, as the backends do.
func svgFont(f Font) string {
	if f.Name != "" {
		return fmt.Sprintf(` font-family="%s"`, html.EscapeString(f.Name))
	}
	var b strings.Builder
	if f.Family != "" {
		fmt.Fprintf(&b, ` font-family="%s"`, html.EscapeString(f.Family))
	} else {
		b.WriteString(` font-family="` + svgDefaultFamily + `"`)
	}
	if f.Weight != 0 {
		fmt.Fprintf(&b, ` font-weight="%d"`, f.Weight)
	}
	if f.Italic {
		b.WriteString(` font-style="italic"`)
	}
	return b.String()
}

func svgSpacing(f Font) string {
	if f.Spacing == 0 {
		return ""
	}
	return fmt.Sprintf(` letter-spacing="%s"`, num(f.Spacing))
}
