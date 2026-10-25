// Package kit holds the small pieces the look-check apps share: fixed colours, round
// avatars, pill badges and coloured dots. It uses only the public gowt facade and icons.
package kit

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
)

// Palette in the order of the macOS system colours; fixed so the shots do not depend on the OS accent.
var (
	Blue   = gowt.RGB{R: 10, G: 132, B: 255}
	Green  = gowt.RGB{R: 48, G: 176, B: 78}
	Orange = gowt.RGB{R: 245, G: 140, B: 10}
	Red    = gowt.RGB{R: 235, G: 64, B: 52}
	Purple = gowt.RGB{R: 130, G: 80, B: 230}
	Teal   = gowt.RGB{R: 40, G: 170, B: 190}
	Gray   = gowt.RGB{R: 142, G: 142, B: 147}
)

// The window size every app opens at, so the shots of one app have the same pixels on every OS.
const (
	Width  = 1280
	Height = 800
)

// Faint is the secondary text colour.
func Faint() gowt.Option { return gowt.Foreground(gowt.SecondaryText) }

// badgeFont is Type.Caption in the given weight, the size of text inside avatars and pills.
func badgeFont(w gowt.Weight) gowt.Font {
	f := gowt.Type.Caption
	f.Weight = w
	return f
}

// Avatar is a filled circle with initials, size points square.
func Avatar(initials string, c gowt.RGB, size int) gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		g.SetFill(c)
		g.FillOval(0, 0, size, size)
		g.SetColor(gowt.RGB{R: 255, G: 255, B: 255})
		w, h := g.TextSize(initials)
		g.Text(initials, (size-w)/2, (size-h)/2)
	}, gowt.Size(size, size), gowt.Typeface(badgeFont(gowt.Semibold)))
}

// Pill is a rounded badge w x h points; the text is centred.
func Pill(text string, w, h int, bg, fg gowt.RGB) gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		g.SetFill(bg)
		g.FillOval(0, 0, h, h)
		g.FillOval(w-h, 0, h, h)
		g.FillRect(h/2, 0, w-h, h)
		g.SetColor(fg)
		tw, th := g.TextSize(text)
		g.Text(text, (w-tw)/2, (h-th)/2)
	}, gowt.Size(w, h), gowt.Typeface(badgeFont(gowt.Medium)))
}

// Tint is a light tone of c for a badge background.
func Tint(c gowt.RGB) gowt.RGB {
	return gowt.RGB{R: 255 - (255-c.R)/5, G: 255 - (255-c.G)/5, B: 255 - (255-c.B)/5}
}

// Dot is a coloured circle image size points wide, for table rows and lists.
func Dot(app *gowt.App, c gowt.RGB, size int) *gowt.Image {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="7" fill="currentColor"/></svg>`
	return icons.FromSVG(app, []byte(svg), size, c)
}

// Symbol is a stroked 24x24 path icon, size points wide.
func Symbol(app *gowt.App, path string, size int, c gowt.RGB) *gowt.Image {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`, path)
	return icons.FromSVG(app, []byte(svg), size, c)
}

// Paths for a few icons the named set lacks (Lucide-style strokes).
const (
	PathInbox   = `<polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>`
	PathSend    = `<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>`
	PathHash    = `<line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/>`
	PathClip    = `<path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>`
	PathArchive = `<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>`
	PathMail    = `<rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/>`
	PathChart   = `<path d="M3 3v16a2 2 0 0 0 2 2h16"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/>`
	PathBell    = `<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>`
	PathPalette = `<circle cx="13.5" cy="6.5" r=".5"/><circle cx="17.5" cy="10.5" r=".5"/><circle cx="8.5" cy="7.5" r=".5"/><circle cx="6.5" cy="12.5" r=".5"/><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.93 0 1.5-.67 1.5-1.5 0-.4-.15-.72-.4-.97-.24-.26-.38-.6-.38-.98 0-.83.67-1.5 1.5-1.5H16c3.1 0 5.5-2.4 5.5-5.5C21.5 6 17.5 2 12 2z"/>`
	PathShield  = `<path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/>`
	PathWifi    = `<path d="M12 20h.01"/><path d="M2 8.82a15 15 0 0 1 20 0"/><path d="M5 12.86a10 10 0 0 1 14 0"/><path d="M8.5 16.43a5 5 0 0 1 7 0"/>`
	PathBug     = `<path d="m8 2 1.88 1.88"/><path d="M14.12 3.88 16 2"/><path d="M9 7.13v-1a3.003 3.003 0 1 1 6 0v1"/><path d="M12 20c-3.3 0-6-2.7-6-6v-3a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v3c0 3.3-2.7 6-6 6"/><path d="M12 20v-9"/>`
)
