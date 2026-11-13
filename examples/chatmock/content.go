package main

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

var (
	cardBg    = gowt.RGB{R: 255, G: 255, B: 255}
	feedPanel *gowt.Panel
)

// content is the white card on the right: the channel header, the message feed, the glass
// composer and the Honey status line at the bottom.
func content(app *gowt.App, header *rx.State[string]) gowt.Option {
	hint := rx.Compute(func() string { return "Message " + header.Get() })
	// Outer wrapper takes the inset from the window edge and carries the warm tone of
	// the sidebar. The card is a Section: installCard gives it the card colour and
	// roundCard clips it (and its rows) to the rounded rectangle - no underlay layers.
	return gowt.VBox(gowt.Grow(), gowt.Padding(6),
		gowt.Background(sidebarBg),
		gowt.Section(gowt.ID("card"), gowt.Grow(),
			gowt.VBox(gowt.Grow(), gowt.Gap(6), gowt.Padding(4),
				gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM), gowt.Size(0, gowt.Scaled(30)),
					gowt.Label("", gowt.ID("header"), gowt.BindText(header), gowt.Typeface(gowt.Font{Scale: 1.15, Weight: gowt.Semibold})),
					gowt.Spacer(),
					gowt.HBox(gowt.Gap(2), gowt.Place(gowt.AlignCenter, gowt.AlignCenter),
						headerButton(app, pathUsers, "Members"),
						gowt.Label("9", gowt.Typeface(gowt.Font{Scale: 0.85}), gowt.Foreground(gowt.SecondaryText))),
					headerButton(app, pathHeadphones, "Huddle"),
					headerButton(app, pathSliders, "Channel settings")),
				gowt.Scroll(gowt.ID("feed"), gowt.Grow(), gowt.VBox(gowt.Ref(&feedPanel), gowt.Gap(gowt.Spacing.MD))),
				composerBar(app, rx.Reader[string](hint)),
				gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Size(0, gowt.Scaled(18)),
					gowt.Label("", gowt.Init(func(l *gowt.LabelView) { l.SetImage(dot(app, honey.color, 9)) })),
					gowt.Label("Honey: Working", gowt.ID("honey"), gowt.Typeface(gowt.Font{Scale: 0.8}), gowt.Foreground(gowt.SecondaryText))))))
}

// headerButton is one of the three icon actions at the top right of the content card.
func headerButton(app *gowt.App, path, tip string) gowt.Option {
	return gowt.Button("", gowt.Borderless(), gowt.Size(gowt.Scaled(28), gowt.Scaled(28)), gowt.Tooltip(tip),
		gowt.Init(func(b *gowt.ButtonView) { b.SetImage(symbol(app, path, 17, grayIcon)) }),
		gowt.OnClick(func() {}))
}

// messageRow is one feed entry: avatar, name and time, paragraphs, reaction pills.
func messageRow(m message) gowt.Option {
	body := []gowt.Option{gowt.Gap(1), gowt.GrowX(),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Place(gowt.AlignStart, gowt.AlignCenter),
			gowt.Label(m.who.name, gowt.Typeface(gowt.Font{Scale: 0.95, Weight: gowt.Semibold})),
			gowt.Label(m.time, gowt.Typeface(gowt.Font{Scale: 0.78}), gowt.Foreground(gowt.SecondaryText)))}
	for _, p := range m.paras {
		body = append(body, gowt.Label(p, gowt.Wrap(), gowt.GrowX(), gowt.Typeface(gowt.Font{Scale: 0.92})))
	}
	if len(m.reactions) > 0 {
		row := []gowt.Option{gowt.Gap(4)}
		for _, r := range m.reactions {
			row = append(row, pill(r))
		}
		body = append(body, gowt.HBox(row...))
	}
	return gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(), gowt.Place(gowt.AlignStart, gowt.AlignStart),
		avatarLabel(m.who, 34),
		gowt.VBox(body...))
}

// newDivider is the NEW separator between older and newer messages: a line edge to edge
// with NEW in the middle, drawn on one canvas so both halves take the full width.
func newDivider() gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		w, h := g.Size()
		g.SetFont(gowt.Font{Scale: 0.72, Weight: gowt.Semibold})
		tw, th := g.TextSize("NEW")
		y := h / 2
		g.SetColor(gowt.RGB{R: 200, G: 200, B: 206})
		g.Line(0, y, (w-tw)/2-10, y)
		g.Line((w+tw)/2+10, y, w, y)
		g.SetColor(gowt.RGB{R: 110, G: 110, B: 118})
		g.Text("NEW", (w-tw)/2, y-th/2-1)
	}, gowt.GrowX(), gowt.Size(0, gowt.Scaled(18)))
}

// composerBar is the rounded white card, two rows as in the reference: the message
// field on top, the @/clip/refresh/AA icons and the round send button below. Send
// appends the typed text to the feed, on click and on Enter.
func composerBar(app *gowt.App, hint rx.Reader[string]) gowt.Option {
	var text *gowt.TextView
	var sendBtn *gowt.CanvasView
	send := func() {
		s := text.Text()
		if s == "" {
			return
		}
		feed = append(feed, message{who: me, time: "now", paras: []string{s}})
		render(feedPanel)
		text.SetText("")
	}
	return gowt.Section(gowt.GrowX(), gowt.Size(0, gowt.Scaled(76)),
		gowt.VBox(gowt.Grow(), gowt.Gap(6),
			gowt.Text(gowt.ID("composer"), gowt.NoBorder(), gowt.GrowX(), gowt.Ref(&text),
				gowt.Init(func(t *gowt.TextView) { gowt.Watch(t, hint, t.SetHint) }),
				gowt.OnActivate(send)),
			gowt.HBox(gowt.GrowX(), gowt.Gap(2),
				composerIcon(app, pathAt, "Mention"),
				composerIcon(app, pathClip, "Attach"),
				composerIcon(app, pathRefresh, "Remind"),
				gowt.Label("AA", gowt.Indent(3, 0), gowt.Typeface(gowt.Font{Scale: 0.78, Weight: gowt.Semibold}), gowt.Foreground(gowt.SecondaryText)),
				gowt.Spacer(),
				gowt.Canvas(func(g *gowt.GC) {
					w, h := g.Size()
					g.SetFill(gowt.RGB{R: 226, G: 226, B: 231})
					g.FillOval(0, 0, w, h)
					g.SetColor(gowt.RGB{R: 100, G: 100, B: 110})
					g.SetLineWidth(2)
					cx := w / 2
					g.Line(cx, h-9, cx, 9)
					g.Line(cx, 9, cx-5, 14)
					g.Line(cx, 9, cx+5, 14)
				}, gowt.ID("send"), gowt.Size(28, 28), gowt.Place(gowt.AlignEnd, gowt.AlignCenter),
					gowt.Ref(&sendBtn),
					gowt.Init(func(*gowt.CanvasView) { sendBtn.OnMouseDown(func(gowt.Mouse) { send() }) })))))
}

func composerIcon(app *gowt.App, path, tip string) gowt.Option {
	return gowt.Button("", gowt.Borderless(), gowt.Size(gowt.Scaled(26), gowt.Scaled(26)), gowt.Tooltip(tip),
		gowt.Init(func(b *gowt.ButtonView) { b.SetImage(symbol(app, path, 15, grayIcon)) }),
		gowt.OnClick(func() {}))
}
