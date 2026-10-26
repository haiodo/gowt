package main

import (
	"runtime"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

// palette is every colour and font the charts draw with. It is rebuilt when the appearance or the
// accent changes and handed to each chart, which records its pictures again.
type palette struct {
	dark                         bool
	card                         gfx.Color // the surface the charts sit on, read from the native card
	accent, orange, purple, red  gfx.Color
	grid, tick, label, labelSoft gfx.Color
	bubble, bubbleEdge, text     gfx.Color
	small, regular, strong       gfx.Font
}

// fontNames are the family names each backend resolves to the system UI font and its semibold.
// gfx has no name for "the system font", so the app picks by OS.
func fontNames() (regular, strong string) {
	switch runtime.GOOS {
	case "darwin":
		return ".AppleSystemUIFont", ".AppleSystemUIFontDemi"
	case "windows":
		return "Segoe UI", "Segoe UI-Semibold"
	}
	return "Sans", "Sans-Bold"
}

// newPalette builds the colours for the appearance. px is the caption size in points
// (gowt.Type.Caption.Px), so chart text follows the density like the native labels.
func newPalette(dark bool, card, accent gfx.Color, px float64) *palette {
	p := &palette{dark: dark, card: card, accent: accent}
	if dark {
		p.orange, p.purple, p.red = gfx.RGB(255, 159, 10), gfx.RGB(191, 90, 242), gfx.RGB(255, 105, 97)
		p.grid, p.tick = gfx.White.WithAlpha(0.10), gfx.White.WithAlpha(0.05)
		p.label, p.labelSoft = gfx.White.WithAlpha(0.62), gfx.White.WithAlpha(0.38)
		p.bubble, p.bubbleEdge = gfx.RGB(58, 58, 60), gfx.White.WithAlpha(0.14)
		p.text = gfx.White.WithAlpha(0.92)
	} else {
		p.orange, p.purple, p.red = gfx.RGB(255, 149, 0), gfx.RGB(175, 82, 222), gfx.RGB(255, 59, 48)
		p.grid, p.tick = gfx.Black.WithAlpha(0.10), gfx.Black.WithAlpha(0.045)
		p.label, p.labelSoft = gfx.Black.WithAlpha(0.60), gfx.Black.WithAlpha(0.38)
		p.bubble, p.bubbleEdge = gfx.RGB(255, 255, 255), gfx.Black.WithAlpha(0.12)
		p.text = gfx.Black.WithAlpha(0.88)
	}
	reg, strong := fontNames()
	p.small = gfx.Font{Name: reg, Size: px}
	p.regular = gfx.Font{Name: reg, Size: px + 1}
	p.strong = gfx.Font{Name: strong, Size: px + 1}
	return p
}

// mix is a blended with b, t of the way.
func mix(a, b gfx.Color, t float64) gfx.Color {
	return gfx.Color{R: a.R + (b.R-a.R)*t, G: a.G + (b.G-a.G)*t, B: a.B + (b.B-a.B)*t, A: a.A + (b.A-a.A)*t}
}

// track is the empty part of a bar: the card pulled a little towards the text colour.
func (c *palette) track() gfx.Color {
	if c.dark {
		return mix(c.card, gfx.White, 0.16)
	}
	return mix(c.card, gfx.Black, 0.10)
}

// setCanvasBackground gives a GTK canvas the colour it is drawn on. Without one SWT fills its whole
// area again before every paint, which was a fifth of the main thread's time while the charts scroll.
func (p *palette) setCanvasBackground(c *swt.Canvas) {
	if runtime.GOOS != "linux" {
		return
	}
	k := func(v float64) int32 { return int32(v*255 + 0.5) }
	c.SetBackgroundWithColor(swt.NewColorRedGreenBlue(k(p.card.R), k(p.card.G), k(p.card.B)))
}
