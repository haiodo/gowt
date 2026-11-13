package main

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/rx"
)

var (
	selBg     = gowt.RGB{R: 224, G: 224, B: 229}
	sidebarBg = gowt.RGB{R: 232, G: 232, B: 206}
)

// groupIcons are the tiny symbols before the group headers, as in the reference.
var groupIcons = map[string]string{
	"The Hive":     pathZap,
	"Product":      pathWrench,
	"Launch Swarm": pathRocket,
}

// channelRows holds every channel row canvas so selecting one can re-paint the others.
var channelRows = map[string]*gowt.CanvasView{}

// sidebar is the translucent left pane: search, Inbox/Agents, grouped channels, direct
// messages and the current user block at the bottom. The panel is glass with a warm tint;
// sidebar is the flush full-height left pane: the material runs under the traffic
// lights, the search and rows start below the title bar strip, and only the selected
// row draws a background - a rounded pill.
func sidebar(app *gowt.App, header, sel *rx.State[string], topInset int) gowt.Option {
	rows := []gowt.Option{}
	for _, g := range groups {
		first := true
		for _, c := range channels {
			if c.group != g {
				continue
			}
			if first {
				first = false
				if g != "" {
					rows = append(rows, groupHeader(app, g))
				}
			}
			rows = append(rows, channelRow(app, c, header, sel))
		}
	}
	// SetSidebar gives the panel its flush material; the warm background keeps the
	// sidebar distinct from the white card, rows fill the same tone so they read as
	// transparent.
	return gowt.VBox(gowt.Grow(), gowt.Padding(6), gowt.Gap(1),
		gowt.Indent(0, topInset),
		gowt.Background(sidebarBg),
		// The search is a small rounded card, like the content card; the Text is
		// borderless so no white rectangle cuts the card.
		gowt.Section(gowt.GrowX(),
			gowt.Text(gowt.Small(), gowt.ID("search"), gowt.NoBorder(), gowt.Hint("Search everything"),
				gowt.GrowX(), gowt.PrefixImage(icons.Get(app, "search", icons.Size(14))))),
		gowt.Scroll(gowt.ID("rows-scroll"), gowt.Grow(), gowt.VBox(append([]gowt.Option{gowt.ID("rows"), gowt.Gap(1), gowt.GrowX()}, rows...)...)),
		userBlock(app))
}

// groupHeader is one small grey section title, with a tiny symbol for the workspace groups.
func groupHeader(app *gowt.App, g string) gowt.Option {
	label := gowt.Label(g, gowt.Typeface(gowt.Font{Scale: 0.8, Weight: gowt.Semibold}),
		gowt.Foreground(gowt.SecondaryText), gowt.Indent(0, 6))
	if path, ok := groupIcons[g]; !ok {
		return label
	} else {
		return gowt.HBox(gowt.Gap(4), gowt.Indent(2, 0),
			gowt.Label("", gowt.Init(func(l *gowt.LabelView) { l.SetImage(symbol(app, path, 11, grayIcon)) })),
			gowt.Indent(0, 6), label)
	}
}

// channelRow is one selectable channel or direct message row drawn on a canvas: a "#" or an
// avatar icon, the name, a trailing badge. Only the selected row paints a background,
// a rounded pill. A canvas is used because a macOS button centres its image+title, which no
// option of the facade overrides.
func channelRow(app *gowt.App, c channel, header, sel *rx.State[string]) gowt.Option {
	var cv *gowt.CanvasView
	img := rowImage(app, c)
	return gowt.Canvas(func(g *gowt.GC) {
		w, h := g.Size()
		// A canvas is opaque: fill it with the sidebar tone so the row reads as transparent.
		g.SetFill(sidebarBg)
		g.FillRect(0, 0, w, h)
		if sel.Get() == c.id {
			g.SetFill(selBg)
			fillRound(g, 0, 1, w, h-2, 6)
		}
		_, ih := img.Size()
		g.Image(img, 8, (h-ih)/2)
		g.SetFont(gowt.Font{Scale: 0.92})
		_, th := g.TextSize(c.name)
		g.SetColor(gowt.RGB{R: 40, G: 40, B: 48})
		g.Text(c.name, 32, (h-th)/2-1)
		switch {
		case c.badge == "dot":
			d := dot(app, gowt.RGB{R: 120, G: 120, B: 128}, 7)
			dw, dh := d.Size()
			g.Image(d, w-dw-10, (h-dh)/2)
		case c.dm && c.badge != "":
			// Direct messages: a black circle with the unread count.
			g.SetFill(gowt.RGB{R: 30, G: 30, B: 34})
			g.FillOval(w-20, (h-14)/2, 14, 14)
			g.SetColor(gowt.RGB{R: 255, G: 255, B: 255})
			g.SetFont(gowt.Font{Scale: 0.62, Weight: gowt.Semibold})
			bw, bh := g.TextSize(c.badge)
			g.Text(c.badge, w-20+(14-bw)/2, (h-14)/2+(14-bh)/2-1)
		case c.badge != "":
			g.SetColor(gowt.RGB{R: 110, G: 110, B: 118})
			g.SetFont(gowt.Font{Scale: 0.75})
			tw, _ := g.TextSize(c.badge)
			g.Text(c.badge, w-tw-10, (h-th)/2-1)
		}
		// GrowX: the row follows the sidebar width when the splitter moves; the drawing
		// reads g.Size(), so the pill and the badges follow.
	}, gowt.ID("ch-"+c.id), gowt.GrowX(), gowt.Size(0, gowt.Scaled(24)),
		gowt.Ref(&cv),
		gowt.Init(func(*gowt.CanvasView) {
			channelRows[c.id] = cv
			cv.OnMouseDown(func(gowt.Mouse) {
				sel.Set(c.id)
				header.Set(channelTitle(c))
				for _, other := range channelRows {
					other.Redraw()
				}
			})
		}))
}

// rowImage is the leading icon of a channel row: "#" for channels, an avatar for direct
// messages, a symbol for the special top rows.
func rowImage(app *gowt.App, c channel) *gowt.Image {
	if c.dm {
		return app.AvatarImage(c.who.initials, c.who.color, 20)
	}
	switch c.id {
	case "inbox":
		return symbol(app, pathInbox, 15, grayIcon)
	case "agents":
		return symbol(app, pathBot, 15, grayIcon)
	case "welcome":
		return symbol(app, pathMail, 15, grayIcon)
	}
	// Channels show a text "#", drawn into an image so rows draw one image either way.
	return hashText(app)
}

// hashText renders the "#" of a channel row as a small image.
func hashText(app *gowt.App) *gowt.Image {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#787880" stroke-width="2" stroke-linecap="round"><line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/></svg>`
	return icons.FromSVG(app, []byte(svg), 14, grayIcon)
}

// userBlock is the current user at the bottom of the sidebar: avatar with a green status
// dot, name and workspace.
func userBlock(app *gowt.App) gowt.Option {
	return gowt.HBox(gowt.ID("user"), gowt.Gap(gowt.Spacing.SM), gowt.Padding(3),
		gowt.Canvas(func(g *gowt.GC) {
			w, h := g.Size()
			g.Image(app.AvatarImage(me.initials, me.color, 34), 0, 0)
			g.SetFill(gowt.RGB{R: 52, G: 199, B: 89})
			g.FillOval(w-11, h-11, 10, 10)
		}, gowt.Size(34, 34)),
		gowt.VBox(gowt.Gap(0),
			gowt.Label(me.name, gowt.Typeface(gowt.Font{Scale: 0.9, Weight: gowt.Semibold})),
			gowt.Label("Honeycomb Studios", gowt.Typeface(gowt.Font{Scale: 0.8}), gowt.Foreground(gowt.SecondaryText))))
}
