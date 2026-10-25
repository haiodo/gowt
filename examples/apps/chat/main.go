// Command chat is a team chat window: a sidebar with sections and avatars, a message feed with
// reactions and links, a thread pane on the right and a glass composer capsule with attach and send.
// Static data, fixed 1280x800 window, one channel selected. It is one of the look-check apps (docs/look-check.md).
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

type person struct {
	name, initials string
	color          gowt.RGB
}

var (
	maria  = person{"Maria Lesova", "ML", kit.Purple}
	ksenia = person{"Ksenia Starkova", "KS", kit.Orange}
	me     = person{"Alex Morgan", "AM", kit.Blue}
	andrei = person{"Andrei Zaytsev", "AZ", kit.Teal}
)

type message struct {
	who       person
	time      string
	lines     []string
	reaction  string
	replies   string
	repliedBy []person
}

var feed = []message{
	{who: maria, time: "15:14", lines: []string{"@all Colleagues, Alexey is about to update staging, anyone against?"}, reaction: "+1  1", replies: "7 replies", repliedBy: []person{me, andrei, maria}},
	{who: ksenia, time: "15:43", lines: []string{"Hello everyone!", "@Alex Morgan Does anyone know this one?", "Alex, could you roll the landing out to staging from this branch again?", "https://example.com/team/platform-landing/tree/PLAT-1475"}, replies: "10 replies", repliedBy: []person{me, ksenia}},
	{who: me, time: "15:44", lines: []string{"after 17", "wait, maybe I can do it in a minute"}},
	{who: andrei, time: "15:53", lines: []string{"@Alex Morgan Shall I rebuild it?"}},
	{who: me, time: "15:53", lines: []string{"I will do it, you do not have push access anyway"}, reaction: "ok  1", replies: "2 replies", repliedBy: []person{andrei, me}},
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		w := app.Window("develop")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(900, 500)

		var sidebar, content, thread *gowt.Panel
		w.Add(sidebarPane(app, &sidebar), contentPane(app, &content), threadPane(app, &thread))
		look.SetToolbar(w, look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Search", OnSearch: func(string) {}},
			look.ToolbarItem{Title: "Hide Thread", Symbol: "sidebar.right", OnClick: func() {}})
		look.SetSidebar(w, sidebar, content, gowt.Scaled(260))
		look.SetInspector(w, content, thread, gowt.Scaled(340))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func sidebarPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	nav := func(h int, sel int, rows ...[2]any) gowt.Option {
		return gowt.Table(gowt.NoBorder(), gowt.GrowX(), gowt.Size(0, gowt.Scaled(h)), gowt.Init(func(t *gowt.TableView) {
			t.Column("", gowt.Scaled(220)) // Table has no column autosize
			t.ShowHeader(false)
			t.ShowLines(false)
			for _, r := range rows {
				t.Row(r[0].(string)).SetImageAt(0, r[1].(*gowt.Image))
			}
			if sel >= 0 {
				t.Select(sel)
			}
		}))
	}
	ic := func(n string) *gowt.Image { return icons.Get(app, n, icons.Size(18)) }
	sym := func(p string) *gowt.Image { return kit.Symbol(app, p, 18, kit.Gray) }
	dm := func(c gowt.RGB) *gowt.Image { return kit.Dot(app, c, 22) }
	return gowt.VBox(gowt.Ref(ref), gowt.Gap(gowt.Spacing.SM),
		gowt.Label("Chats", gowt.Typeface(gowt.Type.BodyStrong), gowt.Place(gowt.AlignCenter, gowt.AlignStart)),
		gowt.Text(gowt.Search(), gowt.Hint("Find channels..."), gowt.GrowX()),
		nav(96, -1, [2]any{"Inbox", sym(kit.PathInbox)}, [2]any{"Discussions", sym(kit.PathMail)}, [2]any{"Saved", ic("star")}),
		gowt.Label("Starred", kit.Faint()),
		nav(112, 0, [2]any{"develop", sym(kit.PathHash)}, [2]any{"fusion-private", ic("lock")}, [2]any{"ios & android builds", sym(kit.PathHash)}),
		gowt.Label("Direct messages", kit.Faint()),
		nav(192, -1, [2]any{"Kristina Fefelova", dm(kit.Green)}, [2]any{"Alexey Ivanov", dm(kit.Blue)}, [2]any{"Anastasia Borisova", dm(kit.Purple)},
			[2]any{"Anton Novgorodtsev", dm(kit.Teal)}, [2]any{"Maxim Denisov", dm(kit.Orange)}),
		gowt.Spacer())
}

func contentPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	rows := make([]gowt.Option, 0, len(feed))
	for _, m := range feed {
		rows = append(rows, messageRow(m))
	}
	return gowt.VBox(gowt.Ref(ref), gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.MD),
		gowt.Scroll(gowt.Grow(), gowt.VBox(append([]gowt.Option{gowt.Gap(gowt.Spacing.LG), gowt.GrowX()}, rows...)...)),
		composer(app, "Message..."))
}

func messageRow(m message) gowt.Option {
	body := []gowt.Option{gowt.Gap(gowt.Spacing.XS), gowt.GrowX(),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Label(m.who.name, gowt.Typeface(gowt.Type.BodyStrong)), gowt.Label(m.time, kit.Faint(), gowt.Typeface(gowt.Type.Caption)))}
	for _, l := range m.lines {
		if len(l) > 8 && l[:8] == "https://" {
			body = append(body, gowt.Link(`<a href="`+l+`">`+l+`</a>`, gowt.Wrap(), gowt.GrowX()))
		} else {
			body = append(body, gowt.Label(l, gowt.Wrap(), gowt.GrowX()))
		}
	}
	if m.reaction != "" {
		// No badge widget: the pill is a Canvas with a fixed width.
		body = append(body, gowt.HBox(kit.Pill(m.reaction, gowt.Scaled(52), gowt.Scaled(24),
			gowt.RGB{R: 232, G: 234, B: 240}, gowt.RGB{R: 40, G: 40, B: 50})))
	}
	if m.replies != "" {
		row := []gowt.Option{gowt.Gap(gowt.Spacing.XS)}
		for _, p := range m.repliedBy {
			row = append(row, kit.Avatar(p.initials, p.color, gowt.Scaled(28)))
		}
		row = append(row, gowt.Link(`<a href="thread">`+m.replies+`</a>`))
		body = append(body, gowt.HBox(row...))
	}
	return gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(), gowt.Place(gowt.AlignStart, gowt.AlignStart),
		kit.Avatar(m.who.initials, m.who.color, gowt.Scaled(40)), gowt.VBox(body...))
}

// composer is the glass capsule with the text field and send button, and a round attach button beside it.
func composer(app *gowt.App, hint string) gowt.Option {
	var cap *gowt.Panel
	return gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(),
		gowt.Button("", look.GlassButton(), gowt.Circle(), gowt.Size(gowt.Scaled(36), gowt.Scaled(36)), gowt.Tooltip("Attach"),
			gowt.Init(func(b *gowt.ButtonView) { b.SetImage(kit.Symbol(app, kit.PathClip, 18, kit.Gray)) })),
		gowt.HBox(gowt.Ref(&cap), gowt.GrowX(), gowt.Padding(gowt.Spacing.XS), gowt.Gap(gowt.Spacing.XS),
			gowt.Init(func(p *gowt.Panel) { look.SetGlass(p, true); look.SetGlassRadius(p, look.Capsule) }),
			gowt.Text(gowt.Borderless(), gowt.GrowX(), gowt.Hint(hint)),
			gowt.Button("", gowt.Flat(), gowt.Size(gowt.Scaled(32), gowt.Scaled(32)), gowt.Tooltip("Send"),
				gowt.Init(func(b *gowt.ButtonView) { b.SetImage(kit.Symbol(app, kit.PathSend, 18, kit.Blue)) }))))
}

func threadPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	code := "Hello,\n\nThank you for your resubmission. Upon further review, we identified additional issues that need your attention. See below for more information.\n\nSubmission ID: 75e80b89-9490-46d1-a748\nReview date: October 23, 2026\nVersion reviewed: 1.0 (319)"
	return gowt.VBox(gowt.Ref(ref), gowt.Gap(gowt.Spacing.MD), gowt.Padding(gowt.Spacing.MD),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Label("Thread", gowt.Typeface(gowt.Type.Subtitle)), gowt.Label("develop", kit.Faint()), gowt.Spacer(),
			gowt.Button("", gowt.Flat(), gowt.Size(gowt.Scaled(28), gowt.Scaled(28)), gowt.Init(func(b *gowt.ButtonView) { b.SetImage(icons.Get(app, "close", icons.Size(14))) }))),
		gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(), gowt.Place(gowt.AlignStart, gowt.AlignStart),
			kit.Avatar(me.initials, me.color, gowt.Scaled(36)),
			gowt.VBox(gowt.Gap(gowt.Spacing.XS), gowt.GrowX(),
				gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Label(me.name, gowt.Typeface(gowt.Type.BodyStrong)), gowt.Label("21:30", kit.Faint(), gowt.Typeface(gowt.Type.Caption))),
				gowt.Label("Apple cannot identify the build, again", gowt.Wrap(), gowt.GrowX()),
				gowt.HBox(kit.Pill("eyes  1", gowt.Scaled(52), gowt.Scaled(24), gowt.RGB{R: 232, G: 234, B: 240}, gowt.RGB{R: 40, G: 40, B: 50})))),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(), gowt.Label("3 replies", kit.Faint(), gowt.Typeface(gowt.Type.BodyStrong)), gowt.Separator(gowt.GrowX(), gowt.Place(gowt.AlignFill, gowt.AlignCenter))),
		gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.Grow(), gowt.Place(gowt.AlignStart, gowt.AlignStart),
			kit.Avatar(me.initials, me.color, gowt.Scaled(36)),
			gowt.VBox(gowt.Gap(gowt.Spacing.XS), gowt.Grow(),
				gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.Label(me.name, gowt.Typeface(gowt.Type.BodyStrong)), gowt.Label("21:30", kit.Faint(), gowt.Typeface(gowt.Type.Caption))),
				gowt.StyledText(gowt.ReadOnly(), gowt.Wrap(), gowt.Grow(), gowt.Borderless(),
					gowt.Init(func(t *gowt.StyledTextView) {
						t.SetFont(gowt.Type.Mono)
						t.SetText(code)
					})))),
		composer(app, "Reply in thread"))
}
