//go:build windows

package fluent

import "github.com/haiodo/gowt/swt"

// painter draws in device pixels on a device context through an SWT GC, which brings GDI+
// anti-aliasing for the rounded shapes.
type painter struct {
	gc   *swt.GC
	cols []*swt.Color
}

func newPainter(d *swt.Display, hdc uintptr) *painter {
	data := swt.NewGCData()
	data.Device = &d.Device
	gc := swt.GCWin32_newHDCData(int64(hdc), data)
	gc.SetAntialias(swt.ON)
	return &painter{gc: gc}
}

func (p *painter) color(c RGB) *swt.Color {
	col := swt.NewColorRedGreenBlue(int32(c.R), int32(c.G), int32(c.B))
	p.cols = append(p.cols, col)
	return col
}

func (p *painter) done() {
	p.gc.Dispose()
	for _, c := range p.cols {
		c.Dispose()
	}
}

func (p *painter) fillRect(c RGB, x, y, w, h int32) {
	p.gc.SetBackground(p.color(c))
	p.gc.FillRectangleInPixels(x, y, w, h)
}

// fillRound fills a rounded rectangle; r is the corner radius.
func (p *painter) fillRound(c RGB, x, y, w, h, r int32) {
	p.gc.SetBackground(p.color(c))
	if r <= 0 {
		p.gc.FillRectangleInPixels(x, y, w, h)
		return
	}
	p.gc.FillRoundRectangleInPixels(x, y, w, h, 2*r, 2*r)
}

// strokeRound draws a line of width lw inside the box (x, y, w, h).
func (p *painter) strokeRound(c RGB, x, y, w, h, r, lw int32) {
	p.gc.SetForeground(p.color(c))
	p.gc.SetLineWidthInPixels(lw)
	if r <= 0 {
		p.gc.DrawRectangleInPixels(x, y, w-1, h-1)
		return
	}
	p.gc.DrawRoundRectangleInPixels(x, y, w-1, h-1, 2*r, 2*r)
}

func (p *painter) line(c RGB, x1, y1, x2, y2, lw int32) {
	p.gc.SetForeground(p.color(c))
	p.gc.SetLineWidthInPixels(lw)
	p.gc.DrawLineInPixels(x1, y1, x2, y2)
}

// drawBuffered paints a w by h image off screen and copies it to dst. GDI+ ignores the clip region of a
// device context, so a non-client paint that must leave the client area alone goes through BitBlt.
func drawBuffered(d *swt.Display, dst uintptr, w, h int32, fn func(p *painter)) {
	mem := call(pCreateCompatDC, dst)
	bmp := call(pCreateCompatBmp, dst, uintptr(w), uintptr(h))
	if mem == 0 || bmp == 0 {
		return
	}
	defer call(pDeleteDC, mem)
	defer call(pDeleteObject, bmp)
	call(pSelectObject, mem, bmp)
	p := newPainter(d, mem)
	fn(p)
	p.done()
	call(pBitBlt, dst, 0, 0, uintptr(w), uintptr(h), mem, 0, 0, 0xCC0020)
}
