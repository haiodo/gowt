//go:build darwin

package cg

import (
	"errors"
	"io"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

var _ gfx.PDFRenderer = (*Renderer)(nil)

// pdfDoc is a PDF context over a growing CFData; the bytes go to w at Close.
type pdfDoc struct {
	r      *Renderer
	w      io.Writer
	data   uintptr
	ctx    uintptr
	inPage bool
}

// NewPDF starts a PDF file written to w by Close. CoreGraphics lays the text out with CoreText and
// embeds the fonts it used. It needs no window.
func (r *Renderer) NewPDF(w io.Writer) (gfx.Document, error) {
	data := cfDataCreateMutable(0, 0)
	consumer := cgConsumerFromData(data)
	aux := pdfInfo(kPDFCreator, cfString("gowt"))
	ctx := cgPDFContextCreate(consumer, nil, aux)
	cgConsumerRelease(consumer)
	cfRelease(aux)
	if ctx == 0 {
		cfRelease(data)
		return nil, errors.New("gfx/cg: CGPDFContextCreate failed")
	}
	return &pdfDoc{r: r, w: w, data: data, ctx: ctx}, nil
}

// pdfInfo makes a one-entry CFDictionary and takes over the reference to the value.
func pdfInfo(key, val uintptr) uintptr {
	d := cfDictionaryCreate(0, &key, &val, 1, kKeyCB, kValCB)
	cfRelease(val)
	return d
}

func (d *pdfDoc) endPage() {
	if d.inPage {
		cgPDFContextEndPage(d.ctx)
		d.inPage = false
	}
}

// NewPage ends the page before it and starts one of wpt by hpt points.
func (d *pdfDoc) NewPage(wpt, hpt float64) gfx.Page {
	d.endPage()
	box := cgRect{0, 0, wpt, hpt}
	info := pdfInfo(kPDFMediaBox, cfDataCreate(0, unsafe.Pointer(&box), int(unsafe.Sizeof(box))))
	cgPDFContextBeginPage(d.ctx, info)
	cfRelease(info)
	d.inPage = true
	return pdfPage{d, wpt, hpt}
}

type pdfPage struct {
	d    *pdfDoc
	w, h float64
}

// Draw paints the items; see gfx.Page.
func (p pdfPage) Draw(scale float64, origin gfx.Point, items []gfx.Item) {
	s := &Surface{r: p.d.r, ctx: p.d.ctx, scale: scale, eff: scale}
	cgSaveGState(s.ctx)
	cgTranslate(s.ctx, 0, p.h)
	cgScale(s.ctx, scale, -scale)
	cgTranslate(s.ctx, -origin.X, -origin.Y)
	s.Draw(items)
	cgRestoreGSt(s.ctx)
}

// Close ends the last page and writes the file.
func (d *pdfDoc) Close() error {
	if d.ctx == 0 {
		return errors.New("gfx/cg: the PDF is closed")
	}
	d.endPage()
	cgPDFContextClose(d.ctx)
	cgContextRelease(d.ctx)
	d.ctx = 0
	n := cfDataLength(d.data)
	p := cfDataBytes(d.data)
	_, err := d.w.Write(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), n))
	cfRelease(d.data)
	return err
}
