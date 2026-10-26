// Package pdf writes gfx pictures as PDF without any system library: paths, strokes, gradients,
// clips, images, group opacity and text. It is the PDF backend of Windows, where Direct2D has
// none, and it runs on every OS, so its output can be checked anywhere.
//
// Text is set in the TrueType font that a Fonts value finds for a gfx.Font and embeds the glyphs
// used (no shaping, kerning from the kern table only). A face that cannot be
// embedded is drawn as outlines; a font nobody finds falls back to the PDF standard Helvetica,
// Latin-1 only. Shadows are approximated by stacked translucent strokes, not a Gaussian blur, and
// the alpha of a gradient stop is not used.
package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/gfx"
)

// Fonts finds the font file for a gfx.Font: the bytes of a TrueType or OpenType file (a .ttc
// collection counts, its first face is used). ok is false when there is none.
type Fonts interface {
	FontFile(f gfx.Font) (data []byte, ok bool)
}

// Document is a PDF file being written; it implements gfx.Document.
type Document struct {
	w     io.Writer
	fonts Fonts
	objs  [][]byte // body of object i+1; nil while reserved
	pages []*page
	res   resources
	err   error
	done  bool
}

var _ gfx.Document = (*Document)(nil)

const (
	catalogObj = 1
	pagesObj   = 2
	resObj     = 3
)

// New starts a document that is written to w by Close. fonts may be nil: all text is then set in
// Helvetica.
func New(w io.Writer, fonts Fonts) *Document {
	d := &Document{w: w, fonts: fonts}
	d.objs = make([][]byte, 3)
	d.res.init()
	return d
}

// reserve allocates an object number to fill later.
func (d *Document) reserve() int {
	d.objs = append(d.objs, nil)
	return len(d.objs)
}

func (d *Document) set(n int, body string) { d.objs[n-1] = []byte(body) }

func (d *Document) add(body string) int {
	n := d.reserve()
	d.set(n, body)
	return n
}

// stream adds a stream object; dict holds the entries besides Length and Filter.
func (d *Document) stream(dict string, data []byte, compress bool) int {
	var buf bytes.Buffer
	if compress {
		zw := zlib.NewWriter(&buf)
		zw.Write(data)
		zw.Close()
		data = buf.Bytes()
		dict += " /Filter /FlateDecode"
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "<< %s /Length %d >>\nstream\n", dict, len(data))
	b.Write(data)
	b.WriteString("\nendstream")
	n := d.reserve()
	d.objs[n-1] = b.Bytes()
	return n
}

// NewPage starts a page of wpt by hpt points.
func (d *Document) NewPage(wpt, hpt float64) gfx.Page {
	p := &page{d: d, w: wpt, h: hpt}
	d.pages = append(d.pages, p)
	return p
}

// Close writes the file.
func (d *Document) Close() error {
	if d.done {
		return d.err
	}
	d.done = true
	var kids strings.Builder
	for _, p := range d.pages {
		c := d.stream("", p.buf.Bytes(), true)
		n := d.add(fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %s %s] /Contents %d 0 R /Resources %d 0 R >>", pagesObj, num(p.w), num(p.h), c, resObj))
		fmt.Fprintf(&kids, "%d 0 R ", n)
	}
	d.res.write(d)
	d.set(catalogObj, fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObj))
	d.set(pagesObj, fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(d.pages), kids.String()))
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offs := make([]int, len(d.objs))
	for i, body := range d.objs {
		offs[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(body)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(d.objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(d.objs)+1, catalogObj, xref)
	if _, err := d.w.Write(out.Bytes()); err != nil && d.err == nil {
		d.err = err
	}
	return d.err
}

// num formats a coordinate with at most four decimals.
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 4, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}
