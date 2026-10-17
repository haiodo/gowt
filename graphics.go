package gowt

import (
	"io"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// RGB is a color, 0-255 per channel.
type RGB struct{ R, G, B uint8 }

// Paint is a widget colour: an RGB, the same in both themes, or an Adaptive with one per theme.
// Only this package implements it.
type Paint interface{ in(dark bool) RGB }

// Adaptive is a colour for each system theme. A widget given one through Foreground, Background or
// their setters takes the other colour when the system switches between light and dark.
type Adaptive struct{ Light, Dark RGB }

func (c RGB) in(bool) RGB { return c }

func (c Adaptive) in(dark bool) RGB {
	if dark {
		return c.Dark
	}
	return c.Light
}

func (c RGB) color() *swt.Color { return swt.NewColorRedGreenBlue(int32(c.R), int32(c.G), int32(c.B)) }

func rgbOf(r *swt.RGB) RGB { return RGB{uint8(r.Red), uint8(r.Green), uint8(r.Blue)} }

// Image is a bitmap. The App owns every image, whichever call made it, and disposes the ones still
// alive when Run returns. Dispose frees one earlier, once nothing shows it: widgets that show an image
// (SetImage) neither copy nor own it.
type Image struct {
	i *swt.Image
	a *App
}

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

// ImageFromProvider creates an image from p, drawn at every zoom the provider answers for (for
// example svg.NewImageDataProvider).
func (a *App) ImageFromProvider(p swt.ImageDataProvider) *Image {
	return a.track(swt.NewImageDeviceImageDataProvider(a.display, p))
}

func (a *App) track(i *swt.Image) *Image {
	img := &Image{i, a}
	a.images[img] = struct{}{}
	return img
}

func (a *App) disposeImages() {
	for img := range a.images {
		img.Dispose()
	}
}

// Size returns the width and height in points.
func (i *Image) Size() (w, h int) {
	b := i.i.GetBounds()
	return int(b.Width), int(b.Height)
}

// Dispose releases the image now instead of when Run returns; calling it again is a no-op.
func (i *Image) Dispose() {
	delete(i.a.images, i)
	if !i.i.IsDisposed() {
		i.i.Dispose()
	}
}

// Unwrap returns the underlying image.
func (i *Image) Unwrap() *swt.Image { return i.i }

// GC draws on a Canvas. It is valid only inside the paint callback it was passed to.
// Colors and fonts are set per call and need no disposal.
type GC struct {
	g *swt.GC
	c *CanvasView
}

// SetColor sets the color of lines and text.
func (g *GC) SetColor(c RGB) { g.g.SetForeground(c.color()) }

// SetFill sets the color of filled shapes and the text background.
func (g *GC) SetFill(c RGB) { g.g.SetBackground(c.color()) }

// SetLineWidth sets the outline width in points.
func (g *GC) SetLineWidth(w int) { g.g.SetLineWidth(int32(w)) }

// Line draws a line from (x1, y1) to (x2, y2).
func (g *GC) Line(x1, y1, x2, y2 int) { g.g.DrawLine(int32(x1), int32(y1), int32(x2), int32(y2)) }

// Rect outlines the rectangle with top left (x, y).
func (g *GC) Rect(x, y, w, h int) { g.g.DrawRectangle(int32(x), int32(y), int32(w), int32(h)) }

// FillRect fills the rectangle with top left (x, y).
func (g *GC) FillRect(x, y, w, h int) { g.g.FillRectangle(int32(x), int32(y), int32(w), int32(h)) }

// Oval outlines the oval inscribed in the rectangle (x, y, w, h).
func (g *GC) Oval(x, y, w, h int) { g.g.DrawOval(int32(x), int32(y), int32(w), int32(h)) }

// FillOval fills the oval inscribed in the rectangle (x, y, w, h).
func (g *GC) FillOval(x, y, w, h int) { g.g.FillOval(int32(x), int32(y), int32(w), int32(h)) }

// Text draws s with its top left at (x, y) over the fill color; tabs and newlines are honored.
func (g *GC) Text(s string, x, y int) { g.g.DrawText(s, int32(x), int32(y)) }

// SetFont sets the font of Text and TextSize for the rest of the paint callback. Empty Name, zero
// Size and Scale mean what they do for Widget.SetFont; Font{} is the canvas's font itself. The canvas
// shares the native fonts with the widgets and releases them when it is disposed.
func (g *GC) SetFont(f Font) {
	if f == (Font{}) {
		g.g.SetFont(g.c.c.GetFont())
		return
	}
	g.g.SetFont(g.c.fonts.get(f.from(g.c.c.GetFont(), g.c.c.GetDisplay())))
}

// TextSize measures s as Text would draw it.
func (g *GC) TextSize(s string) (w, h int) {
	p := g.g.TextExtent(s)
	return int(p.X), int(p.Y)
}

// Image draws img with its top left at (x, y).
func (g *GC) Image(img *Image, x, y int) { g.g.DrawImage(img.i, int32(x), int32(y)) }

// Unwrap returns the underlying GC.
func (g *GC) Unwrap() *swt.GC { return g.g }

// CanvasView is a blank area drawn by the paint callback.
type CanvasView struct {
	base
	c     *swt.Canvas
	fonts *usedFonts // by GC.SetFont
}

type painter func(*swt.PaintEvent)

func (f painter) PaintControl(e *swt.PaintEvent) { f(e) }

// Canvas creates a drawing area. onPaint draws the whole area (clipped to the damaged part) and
// must not keep g; it may be nil. Call Redraw after the model changes.
// Options: Border.
func Canvas(onPaint func(g *GC), opts ...Option) Option {
	return leaf("Canvas", sBorder, opts, func(p *swt.Composite) Widget {
		c := swt.NewCanvasParentStyle(p, resolve(swt.DOUBLE_BUFFERED, opts))
		v := &CanvasView{base{&c.Control}, c, newUsedFonts(&c.Control)}
		if onPaint != nil {
			c.AddPaintListener(painter(func(e *swt.PaintEvent) { onPaint(&GC{e.Gc, v}) }))
		}
		return v
	})
}

// Mouse is a pointer event on a Canvas, in the canvas's coordinates.
type Mouse struct {
	X, Y   int
	Button int // SWT's numbering: 1 left, 2 middle, 3 right; 0 for a move
	Count  int // 2 for the second press of a double click
	Mods       // keys held down; a macOS Ctrl-click is Button 1 with Ctrl
}

func mouseOf(e *swt.MouseEvent) Mouse {
	return Mouse{X: int(e.X), Y: int(e.Y), Button: int(e.Button), Count: int(e.Count), Mods: modsOf(e)}
}

// OnMouseDown runs f when a button is pressed.
func (c *CanvasView) OnMouseDown(f func(Mouse)) {
	c.c.AddMouseListener(swt.MouseListenerMouseDownAdapter(func(e *swt.MouseEvent) { f(mouseOf(e)) }))
}

// OnMouseUp runs f when a button is released.
func (c *CanvasView) OnMouseUp(f func(Mouse)) {
	c.c.AddMouseListener(swt.MouseListenerMouseUpAdapter(func(e *swt.MouseEvent) { f(mouseOf(e)) }))
}

type mover func(*swt.MouseEvent)

func (f mover) MouseMove(e *swt.MouseEvent) { f(e) }

// OnMouseMove runs f when the pointer moves over the canvas.
func (c *CanvasView) OnMouseMove(f func(Mouse)) {
	c.c.AddMouseMoveListener(mover(func(e *swt.MouseEvent) { f(mouseOf(e)) }))
}

// Redraw schedules a repaint, which calls the paint callback.
func (c *CanvasView) Redraw() { c.c.Redraw() }

// Unwrap returns the underlying swt.Canvas for the full API.
func (c *CanvasView) Unwrap() *swt.Canvas { return c.c }

// Size returns the drawable width and height.
func (c *CanvasView) Size() (w, h int) {
	r := c.c.GetClientArea()
	return int(r.Width), int(r.Height)
}
