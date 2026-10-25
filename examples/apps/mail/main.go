// Command mail is a three-pane mail client: mailboxes in a sidebar, a message list with unread dots
// and dates, a reader and a toolbar. Static data, fixed 1280x800 window (docs/look-check.md).
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

type mail struct {
	from, subject, preview, date string
	unread                       bool
}

var inbox = []mail{
	{"Priya Raman", "Quarterly planning notes", "Here are the notes from Tuesday, please add your items before Friday", "09:41", true},
	{"Design Team", "New icon set ready for review", "The final set is in the shared folder. Two glyphs still need a pass", "08:15", true},
	{"Build Service", "Nightly build 2481 passed", "All 312 tests passed on three platforms in 14 minutes", "07:02", false},
	{"Tomas Weber", "Re: Release checklist", "Thanks, I moved the signing step before the upload as discussed", "Yesterday", false},
	{"Hannah Cole", "Lunch on Thursday?", "A few of us are going to the new place near the station", "Yesterday", true},
	{"Accounts", "Your invoice for September", "The invoice is attached. The total is due within thirty days", "Mon", false},
	{"Mei Tanaka", "Photos from the offsite", "I uploaded about sixty photos, the best ones are in the first album", "Mon", false},
	{"Library", "Items due next week", "You have two items due on Monday. Renew them online to avoid a fee", "Sun", false},
	{"Ola Nordmann", "Draft: onboarding guide", "Could you read the second chapter and tell me if the steps are clear", "Sat", false},
	{"Security", "New sign-in from a laptop", "We noticed a sign-in from a new device. If this was you, no action is needed", "Fri", false},
}

const body = `Hi team,

Here are the notes from Tuesday's planning session. The main points:

  1. The release moves to the second week of the month.
  2. The new onboarding flow needs one more round of review.
  3. We will drop the legacy importer after the next release.

Please add your items to the shared document before Friday so we can close the plan in the next meeting.

Thanks,
Priya`

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		w := app.Window("Inbox")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(900, 500)

		var sidebar, content *gowt.Panel
		w.Add(sidebarPane(app, &sidebar), contentPane(app, &content))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Title: "Compose", Symbol: "square.and.pencil", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Archive", Symbol: "archivebox", OnClick: func() {}},
			look.ToolbarItem{Title: "Delete", Symbol: "trash", OnClick: func() {}},
			look.ToolbarItem{Title: "Reply", Symbol: "arrowshape.turn.up.left", OnClick: func() {}},
			look.ToolbarItem{Title: "Search", OnSearch: func(string) {}})
		look.SetSidebar(w, sidebar, content, gowt.Spacing.SidebarWidth)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func sidebarPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
			t.Column("", gowt.Scaled(200)) // Table has no column autosize
			t.ShowHeader(false)
			t.ShowLines(false)
			boxes := []struct {
				name, count string
				icon        *gowt.Image
			}{
				{"Inbox", "3", kit.Symbol(app, kit.PathInbox, 18, kit.Blue)},
				{"Drafts", "", icons.Get(app, "edit", icons.Size(18))},
				{"Sent", "", kit.Symbol(app, kit.PathSend, 18, kit.Gray)},
				{"Archive", "", kit.Symbol(app, kit.PathArchive, 18, kit.Gray)},
				{"Junk", "", icons.Get(app, "warning", icons.Size(18))},
				{"Trash", "", icons.Get(app, "delete", icons.Size(18))},
			}
			for _, b := range boxes {
				r := t.Row(b.name)
				r.SetImageAt(0, b.icon)
			}
			t.Select(0)
		})))
}

func contentPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.Split(gowt.Grow(),
			gowt.VBox(gowt.Padding(0), gowt.Gap(0), list(app)),
			reader()))
}

func list(app *gowt.App) gowt.Option {
	return gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
		t.Column("", gowt.Scaled(28)) // Table has no column autosize
		t.Column("From", gowt.Scaled(150))
		t.Column("Subject", gowt.Scaled(332))
		t.Column("Date", gowt.Scaled(80), gowt.Right())
		t.ShowLines(false)
		dot := kit.Dot(app, kit.Blue, 12)
		for _, m := range inbox {
			r := t.Row("", m.from, m.subject+" - "+m.preview, m.date)
			if m.unread {
				r.SetImageAt(0, dot)
			}
		}
		t.Select(0)
	}))
}

func reader() gowt.Option {
	return gowt.VBox(gowt.Padding(gowt.Scaled(20)), gowt.Gap(gowt.Spacing.MD),
		gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(),
			kit.Avatar("PR", kit.Purple, gowt.Scaled(44)),
			gowt.VBox(gowt.Gap(gowt.Spacing.XS), gowt.GrowX(),
				gowt.Label("Priya Raman", gowt.Typeface(gowt.Type.BodyStrong)),
				gowt.Label("To: Team", kit.Faint(), gowt.Typeface(gowt.Type.Caption))),
			gowt.Label("Today at 09:41", kit.Faint(), gowt.Typeface(gowt.Type.Caption))),
		gowt.Label("Quarterly planning notes", gowt.Typeface(gowt.Type.Subtitle)),
		gowt.Separator(gowt.GrowX()),
		gowt.Label(body, gowt.Wrap(), gowt.GrowX()),
		gowt.Link(`<a href="https://example.com/plan">Open the shared document</a>`),
		gowt.Spacer())
}
