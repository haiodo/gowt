package gowt

import (
	"io"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// RGB is a color, 0-255 per channel.
type RGB struct{ R, G, B uint8 }

func (c RGB) color() *swt.Color { return swt.NewColorRedGreenBlue(int32(c.R), int32(c.G), int32(c.B)) }

func rgbOf(r *swt.RGB) RGB { return RGB{uint8(r.Red), uint8(r.Green), uint8(r.Blue)} }

// Image is a bitmap. The caller owns it: Dispose when done. Widgets that show an image
// (SetImage) do not take ownership and do not copy it, so keep it alive while shown. Images
// still alive when Run returns are disposed with the App.
type Image struct{ i *swt.Image }

// LoadImage reads an image file (PNG, JPEG, GIF, BMP, ICO, TIFF).
func (a *App) LoadImage(path string) (img *Image, err error) {
	defer catch(&err)
	return a.track(swt.NewImageDeviceFilename(a.display, path)), nil
}

// ImageFrom decodes an image from r.
func (a *App) ImageFrom(r io.Reader) (img *Image, err error) {
	defer catch(&err)
	return a.track(swt.NewImageDeviceStream(a.display, jrt.NewInputStream(r))), nil
}

func (a *App) track(i *swt.Image) *Image {
	img := &Image{i}
	a.images = append(a.images, img)
	return img
}

func (a *App) disposeImages() {
	for _, img := range a.images {
		img.Dispose()
	}
}

// Size returns the width and height in points.
func (i *Image) Size() (w, h int) {
	b := i.i.GetBounds()
	return int(b.Width), int(b.Height)
}

// Dispose releases the image; calling it again is a no-op.
func (i *Image) Dispose() {
	if !i.i.IsDisposed() {
		i.i.Dispose()
	}
}

// Unwrap returns the underlying image.
func (i *Image) Unwrap() *swt.Image { return i.i }

// GC draws on a Canvas. It is valid only inside the paint callback it was passed to.
// Colors are set per call and need no disposal.
type GC struct{ g *swt.GC }

func (g *GC) SetColor(c RGB)     { g.g.SetForeground(c.color()) }
func (g *GC) SetFill(c RGB)      { g.g.SetBackground(c.color()) }
func (g *GC) SetLineWidth(w int) { g.g.SetLineWidth(int32(w)) }

func (g *GC) Line(x1, y1, x2, y2 int) { g.g.DrawLine(int32(x1), int32(y1), int32(x2), int32(y2)) }
func (g *GC) Rect(x, y, w, h int)     { g.g.DrawRectangle(int32(x), int32(y), int32(w), int32(h)) }
func (g *GC) FillRect(x, y, w, h int) { g.g.FillRectangle(int32(x), int32(y), int32(w), int32(h)) }
func (g *GC) Oval(x, y, w, h int)     { g.g.DrawOval(int32(x), int32(y), int32(w), int32(h)) }
func (g *GC) FillOval(x, y, w, h int) { g.g.FillOval(int32(x), int32(y), int32(w), int32(h)) }

// Text draws s with its top left at (x, y) over the fill color; tabs and newlines are honored.
func (g *GC) Text(s string, x, y int) { g.g.DrawText(s, int32(x), int32(y)) }

// TextSize measures s as Text would draw it.
func (g *GC) TextSize(s string) (w, h int) {
	p := g.g.TextExtent(s)
	return int(p.X), int(p.Y)
}

// Image draws img with its top left at (x, y).
func (g *GC) Image(img *Image, x, y int) { g.g.DrawImage(img.i, int32(x), int32(y)) }

// Unwrap returns the underlying GC.
func (g *GC) Unwrap() *swt.GC { return g.g }

// Canvas is a blank area drawn by the paint callback.
type Canvas struct{ c *swt.Canvas }

type painter func(*swt.PaintEvent)

func (f painter) PaintControl(e *swt.PaintEvent) { f(e) }

// Canvas adds a drawing area. onPaint draws the whole area (clipped to the damaged part) and
// must not keep g; it may be nil. Call Redraw after the model changes.
func (p *Panel) Canvas(onPaint func(g *GC), opts ...Option) *Canvas {
	c := swt.NewCanvasParentStyle(p.c, resolve(swt.DOUBLE_BUFFERED, opts))
	applyOpts(&c.Control, opts)
	if onPaint != nil {
		c.AddPaintListener(painter(func(e *swt.PaintEvent) { onPaint(&GC{e.Gc}) }))
	}
	return &Canvas{c}
}

func (c *Canvas) control() *swt.Control { return &c.c.Control }
func (c *Canvas) Redraw()               { c.c.Redraw() }
func (c *Canvas) Unwrap() *swt.Canvas   { return c.c }

// Size returns the drawable width and height.
func (c *Canvas) Size() (w, h int) {
	r := c.c.GetClientArea()
	return int(r.Width), int(r.Height)
}
