package pdf

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf16"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/haiodo/gowt/gfx"
)

type faceKind int

const (
	kindStd     faceKind = iota // a PDF standard font, Latin-1
	kindEmbed                   // the font file embedded, text stays text
	kindOutline                 // glyph outlines as paths
)

type faceKey struct {
	name, family string
	weight       int
	italic       bool
}

type face struct {
	kind faceKind
	name string // resource name, for std and embedded
	std  string // BaseFont of a standard font
	font *sfnt.Font
	buf  sfnt.Buffer
	upem float64
	data []byte
	ps   string
	used map[sfnt.GlyphIndex]rune
}

// face finds the face of f: the file Fonts gives, or the standard Helvetica of the same weight and slant.
func (r *resources) face(d *Document, f gfx.Font) *face {
	k := faceKey{f.Name, f.Family, f.Weight, f.Italic}
	if fc, ok := r.faces[k]; ok {
		return fc
	}
	fc := r.open(d, f)
	r.faces[k] = fc
	return fc
}

func (r *resources) open(d *Document, f gfx.Font) *face {
	if d.fonts != nil {
		if data, ok := d.fonts.FontFile(f); ok {
			if fc := newFace(data); fc != nil {
				if fc.kind == kindEmbed {
					r.addFont(fc)
				}
				return fc
			}
		}
	}
	name := "Helvetica"
	switch bold, it := f.Weight >= 600, f.Italic; {
	case bold && it:
		name = "Helvetica-BoldOblique"
	case bold:
		name = "Helvetica-Bold"
	case it:
		name = "Helvetica-Oblique"
	}
	for _, fc := range r.fonts {
		if fc.kind == kindStd && fc.std == name {
			return fc
		}
	}
	fc := &face{kind: kindStd, std: name}
	r.addFont(fc)
	return fc
}

func (r *resources) addFont(fc *face) {
	fc.name = fmt.Sprintf("F%d", len(r.fonts)+1)
	r.fonts = append(r.fonts, fc)
}

// newFace parses data; nil when it is no font file sfnt can read.
func newFace(data []byte) *face {
	if len(data) < 12 {
		return nil
	}
	fc := &face{data: data, kind: kindOutline, used: map[sfnt.GlyphIndex]rune{}}
	var err error
	switch string(data[:4]) {
	case "ttcf":
		var c *sfnt.Collection
		if c, err = sfnt.ParseCollection(data); err == nil {
			fc.font, err = c.Font(0)
		}
	default:
		fc.font, err = sfnt.Parse(data)
		if sig := binary.BigEndian.Uint32(data); (sig == 0x00010000 || string(data[:4]) == "true") && fsType(data)&0x0002 == 0 {
			fc.kind = kindEmbed
		}
	}
	if err != nil || fc.font == nil {
		return nil
	}
	fc.upem = float64(fc.font.UnitsPerEm())
	fc.ps = psName(fc.font, &fc.buf)
	return fc
}

func psName(f *sfnt.Font, b *sfnt.Buffer) string {
	n, _ := f.Name(b, sfnt.NameIDPostScript)
	var out strings.Builder
	for _, r := range n {
		if r < 128 && (r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			out.WriteRune(r)
		}
	}
	if out.Len() == 0 {
		return "EmbeddedFont"
	}
	return out.String()
}

// fsType is the embedding permission of the OS/2 table; 0 (installable) when the table is missing.
func fsType(data []byte) uint16 {
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n && 12+16*i+16 <= len(data); i++ {
		rec := data[12+16*i:]
		if string(rec[:4]) != "OS/2" {
			continue
		}
		if off := int(binary.BigEndian.Uint32(rec[8:])); off+10 <= len(data) {
			return binary.BigEndian.Uint16(data[off+8:])
		}
	}
	return 0
}

func (fc *face) ppem() fixed.Int26_6 { return fixed.Int26_6(fc.upem * 64) }

// units converts a fixed-point value measured at ppem = units per em to font units.
func units(v fixed.Int26_6) float64 { return float64(v) / 64 }

func (fc *face) milli(v float64) float64 { return v * 1000 / fc.upem }

// finish writes the objects of a font used in the document and returns the object number to list.
func (fc *face) finish(d *Document) int {
	if fc.kind == kindStd {
		return d.add(fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>", fc.std))
	}
	gids := make([]sfnt.GlyphIndex, 0, len(fc.used))
	for g := range fc.used {
		gids = append(gids, g)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	var w strings.Builder
	for _, g := range gids {
		adv, err := fc.font.GlyphAdvance(&fc.buf, g, fc.ppem(), font.HintingNone)
		if err != nil {
			continue
		}
		fmt.Fprintf(&w, "%d [%s] ", g, num(math.Round(fc.milli(units(adv)))))
	}
	ids := make([]uint16, len(gids))
	for i, g := range gids {
		ids[i] = uint16(g)
	}
	data := fc.data
	if sub := subsetTTF(data, ids); sub != nil {
		data = sub
	}
	file := d.stream(fmt.Sprintf("/Length1 %d", len(data)), data, true)
	asc, desc, bbox := 800.0, -200.0, [4]float64{0, -200, 1000, 800}
	if m, err := fc.font.Metrics(&fc.buf, fc.ppem(), font.HintingNone); err == nil {
		asc, desc = fc.milli(units(m.Ascent)), -fc.milli(units(m.Descent))
	}
	if b, err := fc.font.Bounds(&fc.buf, fc.ppem(), font.HintingNone); err == nil {
		bbox = [4]float64{fc.milli(units(b.Min.X)), -fc.milli(units(b.Max.Y)), fc.milli(units(b.Max.X)), -fc.milli(units(b.Min.Y))}
	}
	desc2 := d.add(fmt.Sprintf("<< /Type /FontDescriptor /FontName /%s /Flags 32 /FontBBox [%s %s %s %s] /ItalicAngle 0 /Ascent %s /Descent %s /CapHeight %s /StemV 80 /FontFile2 %d 0 R >>",
		fc.ps, num(bbox[0]), num(bbox[1]), num(bbox[2]), num(bbox[3]), num(asc), num(desc), num(asc), file))
	cid := d.add(fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %d 0 R /CIDToGIDMap /Identity /DW 1000 /W [%s] >>",
		fc.ps, desc2, w.String()))
	uni := d.stream("", []byte(fc.toUnicode(gids)), true)
	return d.add(fmt.Sprintf("<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>", fc.ps, cid, uni))
}

// toUnicode maps the glyphs used to the characters they were set for, so the text can be copied.
func (fc *face) toUnicode(gids []sfnt.GlyphIndex) string {
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin 12 dict begin begincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n/CMapName /Adobe-Identity-UCS def /CMapType 2 def\n1 begincodespacerange <0000> <FFFF> endcodespacerange\n")
	for len(gids) > 0 {
		n := min(len(gids), 100)
		fmt.Fprintf(&b, "%d beginbfchar\n", n)
		for _, g := range gids[:n] {
			fmt.Fprintf(&b, "<%04X> <", g)
			for _, u := range utf16.Encode([]rune{fc.used[g]}) {
				fmt.Fprintf(&b, "%04X", u)
			}
			b.WriteString(">\n")
		}
		b.WriteString("endbfchar\n")
		gids = gids[n:]
	}
	b.WriteString("endcmap CMapName currentdict /CMap defineresource pop end end\n")
	return b.String()
}

// text draws op.Text; with shadow it draws the shadow of op.Shadow under it first.
func (rp *replayer) text(op *gfx.Op, shadow bool) {
	if op.Text == "" || op.Font.Size <= 0 {
		return
	}
	fc := rp.d.res.face(rp.d, op.Font)
	if sh := op.Shadow; shadow && sh.Color.A > 0 {
		rp.save()
		rp.cm(gfx.TranslateBy(sh.DX, sh.DY))
		if sh.Blur > 0 {
			c := sh.Color
			c.A = shadowAlpha(sh.Color.A)
			for j := 1; j <= shadowSteps; j++ {
				rp.textRun(fc, op, c, 2*sh.Blur*float64(j)/shadowSteps)
			}
		}
		rp.textRun(fc, op, sh.Color, 0)
		rp.restore()
	}
	rp.textRun(fc, op, op.Color, 0)
}

// textRun sets the text in color c, or strokes its outline with the given width when that is above 0.
func (rp *replayer) textRun(fc *face, op *gfx.Op, c gfx.Color, strokeW float64) {
	if c.A <= 0 {
		return
	}
	rp.save()
	rp.color(c, strokeW > 0)
	if strokeW > 0 {
		rp.printf("%s w 1 J 1 j\n", num(strokeW))
	}
	switch fc.kind {
	case kindOutline:
		rp.outlines(fc, op, strokeW > 0)
	default:
		rp.printf("BT /%s %s Tf %s Tc %d Tr 1 0 0 -1 %s %s Tm %s TJ ET\n", fc.name, num(op.Font.Size), num(op.Font.Spacing),
			map[bool]int{false: 0, true: 1}[strokeW > 0], num(op.A), num(op.B), fc.show(op.Text))
	}
	rp.restore()
}

// show is the operand of TJ for s in the face.
func (fc *face) show(s string) string {
	if fc.kind == kindStd {
		return "[" + winAnsi(s) + "]"
	}
	var arr, run strings.Builder
	arr.WriteString("[")
	flush := func() {
		if run.Len() > 0 {
			arr.WriteString("<" + run.String() + ">")
			run.Reset()
		}
	}
	prev := sfnt.GlyphIndex(0)
	for i, r := range s {
		g, err := fc.font.GlyphIndex(&fc.buf, r)
		if err != nil {
			g = 0
		}
		if i > 0 {
			if k, err := fc.font.Kern(&fc.buf, prev, g, fc.ppem(), font.HintingNone); err == nil && k != 0 {
				flush()
				arr.WriteString(num(-fc.milli(units(k))) + " ")
			}
		}
		fmt.Fprintf(&run, "%04X", g)
		fc.used[g] = r
		prev = g
	}
	flush()
	arr.WriteString("]")
	return arr.String()
}

// outlines draws the glyphs as filled (or stroked) paths, advancing by their widths and the spacing.
func (rp *replayer) outlines(fc *face, op *gfx.Op, stroke bool) {
	k := op.Font.Size / fc.upem / 64
	x := op.A
	prev := sfnt.GlyphIndex(0)
	for i, r := range op.Text {
		g, err := fc.font.GlyphIndex(&fc.buf, r)
		if err != nil {
			g = 0
		}
		if i > 0 {
			if kern, err := fc.font.Kern(&fc.buf, prev, g, fc.ppem(), font.HintingNone); err == nil {
				x += units(kern) * op.Font.Size / fc.upem
			}
		}
		if segs, err := fc.font.LoadGlyph(&fc.buf, g, fc.ppem(), nil); err == nil {
			pt := func(p fixed.Point26_6) (float64, float64) {
				return x + float64(p.X)*k, op.B + float64(p.Y)*k
			}
			var cx, cy float64
			open := false
			for _, sg := range segs {
				switch sg.Op {
				case sfnt.SegmentOpMoveTo:
					if open {
						rp.out.WriteString("h\n")
					}
					cx, cy = pt(sg.Args[0])
					rp.printf("%s %s m\n", num(cx), num(cy))
					open = true
				case sfnt.SegmentOpLineTo:
					cx, cy = pt(sg.Args[0])
					rp.printf("%s %s l\n", num(cx), num(cy))
				case sfnt.SegmentOpQuadTo:
					x1, y1 := pt(sg.Args[0])
					x2, y2 := pt(sg.Args[1])
					rp.printf("%s %s %s %s %s %s c\n", num(cx+2.0/3*(x1-cx)), num(cy+2.0/3*(y1-cy)),
						num(x2+2.0/3*(x1-x2)), num(y2+2.0/3*(y1-y2)), num(x2), num(y2))
					cx, cy = x2, y2
				case sfnt.SegmentOpCubeTo:
					x1, y1 := pt(sg.Args[0])
					x2, y2 := pt(sg.Args[1])
					cx, cy = pt(sg.Args[2])
					rp.printf("%s %s %s %s %s %s c\n", num(x1), num(y1), num(x2), num(y2), num(cx), num(cy))
				}
			}
			if open {
				rp.out.WriteString("h\n")
			}
			if stroke {
				rp.out.WriteString("S\n")
			} else {
				rp.out.WriteString("f\n")
			}
		}
		adv, _ := fc.font.GlyphAdvance(&fc.buf, g, fc.ppem(), font.HintingNone)
		x += units(adv)*op.Font.Size/fc.upem + op.Font.Spacing
		prev = g
	}
}

// winAnsi writes s as PDF literal strings in Windows-1252; what that code page lacks becomes '?'.
func winAnsi(s string) string {
	var b strings.Builder
	b.WriteString("(")
	for _, r := range s {
		c, ok := cp1252(r)
		if !ok {
			c = '?'
		}
		switch {
		case c == '(' || c == ')' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 32 || c > 126:
			fmt.Fprintf(&b, "\\%03o", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString(")")
	return b.String()
}

var cp1252High = map[rune]byte{
	0x20AC: 0x80, 0x201A: 0x82, 0x0192: 0x83, 0x201E: 0x84, 0x2026: 0x85, 0x2020: 0x86, 0x2021: 0x87, 0x02C6: 0x88,
	0x2030: 0x89, 0x0160: 0x8A, 0x2039: 0x8B, 0x0152: 0x8C, 0x017D: 0x8E, 0x2018: 0x91, 0x2019: 0x92, 0x201C: 0x93,
	0x201D: 0x94, 0x2022: 0x95, 0x2013: 0x96, 0x2014: 0x97, 0x02DC: 0x98, 0x2122: 0x99, 0x0161: 0x9A, 0x203A: 0x9B,
	0x0153: 0x9C, 0x017E: 0x9E, 0x0178: 0x9F,
}

func cp1252(r rune) (byte, bool) {
	if r < 0x80 || r >= 0xA0 && r <= 0xFF {
		return byte(r), true
	}
	b, ok := cp1252High[r]
	return b, ok
}
