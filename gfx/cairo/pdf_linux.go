//go:build linux

package cairo

import (
	"errors"
	"io"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/gfx"
)

var _ gfx.PDFRenderer = (*Renderer)(nil)

const statusWriteError = 11

// The one write callback of the process: the closure it gets is the key of the document's writer.
var (
	pdfCallbackOnce sync.Once
	pdfCallback     uintptr
	pdfSinks        sync.Map // uintptr -> *pdfDoc
	pdfSinkID       uintptr
	pdfSinkMu       sync.Mutex
)

func pdfWrite(closure, data uintptr, length uint32) int32 {
	v, ok := pdfSinks.Load(closure)
	if !ok {
		return statusWriteError
	}
	d := v.(*pdfDoc)
	if d.err != nil {
		return statusWriteError
	}
	if _, err := d.w.Write(unsafe.Slice((*byte)(ptr(data)), length)); err != nil {
		d.err = err
		return statusWriteError
	}
	return 0
}

// pdfDoc is a cairo PDF surface that streams to w; the first page makes the surface because its
// size is the size of the first page.
type pdfDoc struct {
	r    *Renderer
	w    io.Writer
	id   uintptr
	surf uintptr
	cr   uintptr
	err  error
}

// NewPDF starts a PDF file streamed to w. Pango lays the text out and cairo embeds the fonts it
// used as subsets. It needs no display.
func (r *Renderer) NewPDF(w io.Writer) (gfx.Document, error) {
	pdfCallbackOnce.Do(func() { pdfCallback = purego.NewCallback(pdfWrite) })
	d := &pdfDoc{r: r, w: w}
	pdfSinkMu.Lock()
	pdfSinkID++
	d.id = pdfSinkID
	pdfSinkMu.Unlock()
	pdfSinks.Store(d.id, d)
	return d, nil
}

// NewPage ends the page before it and starts one of wpt by hpt points.
func (d *pdfDoc) NewPage(wpt, hpt float64) gfx.Page {
	if d.surf == 0 {
		d.surf = pdfCreateForStream(pdfCallback, d.id, wpt, hpt)
		d.cr = create(d.surf)
	} else {
		showPage(d.cr)
		pdfSetSize(d.surf, wpt, hpt)
	}
	return pdfPage{d}
}

type pdfPage struct{ d *pdfDoc }

// Draw paints the items; see gfx.Page.
func (p pdfPage) Draw(scale float64, origin gfx.Point, items []gfx.Item) {
	t := &Target{r: p.d.r, cr: p.d.cr, scale: scale, eff: scale}
	save(t.cr)
	scaleBy(t.cr, scale)
	translate(t.cr, -origin.X, -origin.Y)
	t.Draw(items)
	restore(t.cr)
}

// Close ends the last page and writes the file.
func (d *pdfDoc) Close() error {
	defer pdfSinks.Delete(d.id)
	if d.surf == 0 {
		return errors.New("gfx/cairo: a PDF needs at least one page")
	}
	showPage(d.cr)
	destroy(d.cr)
	surfaceFinish(d.surf)
	status := surfaceStatus(d.surf)
	surfaceDestroy(d.surf)
	d.surf, d.cr = 0, 0
	if d.err != nil {
		return d.err
	}
	if status != 0 {
		return errors.New("gfx/cairo: the PDF surface failed")
	}
	return nil
}
