package main

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
)

// grayIcon is the colour of the small sidebar and header symbols.
var grayIcon = gowt.RGB{R: 120, G: 120, B: 128}

// transparent clears a button background back to nothing.
var transparent = gowt.RGB{}

// Lucide-style stroke paths for the symbols the named icon set lacks.
const (
	pathInbox      = `<polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>`
	pathBot        = `<path d="M12 8V4h-4"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/>`
	pathZap        = `<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>`
	pathWrench     = `<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>`
	pathRocket     = `<path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"/><path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/>`
	pathMail       = `<rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/>`
	pathUsers      = `<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>`
	pathBookmark   = `<path d="m19 21-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>`
	pathPin        = `<path d="M12 17v5"/><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a2 2 0 0 0 0-4H8a2 2 0 0 0 0 4h1z"/>`
	pathAt         = `<circle cx="12" cy="12" r="4"/><path d="M16 8v5a3 3 0 0 0 6 0v-1a10 10 0 1 0-4 8"/>`
	pathClip       = `<path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>`
	pathMic        = `<path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><path d="M12 19v3"/>`
	pathThumbsUp   = `<path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"/>`
	pathEye        = `<path d="M2.06 12.35a1 1 0 0 1 0-.7 10.75 10.75 0 0 1 19.88 0 1 1 0 0 1 0 .7 10.75 10.75 0 0 1-19.88 0"/><circle cx="12" cy="12" r="3"/>`
	pathHexagon    = `<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>`
	pathHeadphones = `<path d="M3 14h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7a9 9 0 0 1 18 0v7a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3"/>`
	pathSliders    = `<path d="m21 4h-7"/><path d="m10 4H3"/><path d="m21 12h-9"/><path d="m8 12H3"/><path d="m21 20h-5"/><path d="m12 20H3"/><path d="M14 2v4"/><path d="M8 10v4"/><path d="M16 18v4"/>`
	pathRefresh    = `<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>`
	pathArrowUp    = `<path d="m5 12 7-7 7 7"/><path d="M12 19V5"/>`
	pathSend       = `<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>`
)

// symbol is a stroked 24x24 path icon, size points wide, in one colour.
func symbol(app *gowt.App, path string, size int, c gowt.RGB) *gowt.Image {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`, path)
	return icons.FromSVG(app, []byte(svg), size, c)
}

// dot is a filled circle image, size points wide.
func dot(app *gowt.App, c gowt.RGB, size int) *gowt.Image {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8" fill="currentColor"/></svg>`
	return icons.FromSVG(app, []byte(svg), size, c)
}

// avatar is the round avatar of p: the photo asset when there is one, else initials.
func avatar(app *gowt.App, p person, size int) gowt.Option {
	return gowt.Label("", gowt.Size(size, size), gowt.Init(func(l *gowt.LabelView) {
		if img := photo(app, p, size); img != nil {
			l.SetImage(img)
			return
		}
		l.SetImage(app.AvatarImage(p.initials, p.color, size))
	}))
}

// fillRound fills a rounded rect of radius r with the current fill colour.
func fillRound(g *gowt.GC, x, y, w, h, r int) {
	g.FillOval(x, y, 2*r, 2*r)
	g.FillOval(x+w-2*r, y, 2*r, 2*r)
	g.FillOval(x, y+h-2*r, 2*r, 2*r)
	g.FillOval(x+w-2*r, y+h-2*r, 2*r, 2*r)
	g.FillRect(x+r, y, w-2*r, h)
	g.FillRect(x, y+r, r, h-2*r)
	g.FillRect(x+w-r, y+r, r, h-2*r)
}

var theApp *gowt.App

// avatarLabel is the feed row avatar; it uses the app of the running demo.
func avatarLabel(p person, size int) gowt.Option {
	return avatar(theApp, p, size)
}

// pill is a rounded reaction badge: a symbol icon and its count.
func pill(r reaction) gowt.Option {
	img := symbol(theApp, r.icon, 13, r.color)
	if r.color == (gowt.RGB{}) {
		img = symbol(theApp, r.icon, 13, gowt.RGB{R: 90, G: 90, B: 100})
	}
	cw := len(r.count)*8 + 34
	return gowt.Canvas(func(g *gowt.GC) {
		w, h := g.Size()
		g.SetFill(gowt.RGB{R: 238, G: 238, B: 242})
		g.FillOval(0, 0, h, h)
		g.FillOval(w-h, 0, h, h)
		g.FillRect(h/2, 0, w-h, h)
		_, ih := img.Size()
		g.Image(img, 9, (h-ih)/2)
		g.SetFont(gowt.Font{Scale: 0.8})
		g.SetColor(gowt.RGB{R: 80, G: 80, B: 90})
		_, th := g.TextSize(r.count)
		g.Text(r.count, 26, (h-th)/2-1)
	}, gowt.Size(cw, 24), gowt.Typeface(gowt.Font{Scale: 0.8}))
}
