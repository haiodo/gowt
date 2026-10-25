// Command files is a file browser: a folder tree in the sidebar, a table with icons, sizes and
// dates, a path bar, sorting and a search field. Static data, fixed 1280x800 window (docs/look-check.md).
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

type entry struct {
	name, modified, size, kind string
	dir                        bool
}

var entries = []entry{
	{"Archive", "12 Sep 2026, 18:20", "--", "Folder", true},
	{"Designs", "21 Oct 2026, 11:05", "--", "Folder", true},
	{"Invoices", "30 Sep 2026, 09:12", "--", "Folder", true},
	{"Photos", "18 Oct 2026, 20:44", "--", "Folder", true},
	{"budget-2027.xlsx", "22 Oct 2026, 16:31", "84 KB", "Spreadsheet", false},
	{"contract-draft.pdf", "19 Oct 2026, 10:02", "1.2 MB", "PDF document", false},
	{"meeting-notes.txt", "23 Oct 2026, 08:50", "4 KB", "Plain text", false},
	{"presentation.key", "20 Oct 2026, 14:18", "36.4 MB", "Presentation", false},
	{"roadmap.md", "24 Oct 2026, 09:15", "12 KB", "Markdown", false},
	{"screenshot.png", "17 Oct 2026, 13:27", "812 KB", "PNG image", false},
	{"team-photo.jpg", "10 Oct 2026, 12:00", "3.4 MB", "JPEG image", false},
	{"timeline.csv", "02 Oct 2026, 17:45", "220 KB", "CSV document", false},
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		w := app.Window("Documents")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(800, 500)

		var sidebar, content *gowt.Panel
		w.Add(sidebarPane(app, &sidebar), contentPane(app, &content))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Title: "Back/Forward", Nav: true, Group: []look.ToolbarItem{
				{Title: "Back", Symbol: "chevron.backward", OnClick: func() {}},
				{Title: "Forward", Symbol: "chevron.forward", OnClick: func() {}}}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "New Folder", Symbol: "folder.badge.plus", OnClick: func() {}},
			look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up", OnClick: func() {}},
			look.ToolbarItem{Title: "Search", OnSearch: func(string) {}})
		look.SetSidebar(w, sidebar, content, gowt.Scaled(240))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func sidebarPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.Tree(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TreeView) {
			t.Column("", gowt.Scaled(220))
			t.ShowHeader(false)
			folder := icons.Get(app, "folder", icons.Size(18))
			home := t.Node("Home")
			home.SetImage(icons.Get(app, "home", icons.Size(18)))
			docs := home.Node("Documents")
			docs.SetImage(folder)
			for _, n := range []string{"Archive", "Designs", "Invoices", "Photos"} {
				docs.Node(n).SetImage(folder)
			}
			home.Node("Downloads").SetImage(icons.Get(app, "download", icons.Size(18)))
			home.Node("Music").SetImage(folder)
			home.Node("Pictures").SetImage(folder)
			home.SetExpanded(true)
			docs.SetExpanded(true)
			t.Node("Desktop").SetImage(folder)
			t.Node("Shared").SetImage(folder)
			t.Select(docs)
		})))
}

func contentPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(), gowt.Padding(gowt.Spacing.SM),
			gowt.Link(`<a href="home">Home</a>`), gowt.Label(">", kit.Faint()),
			gowt.Label("Documents", gowt.Typeface(gowt.Type.BodyStrong)),
			gowt.Spacer(),
			gowt.Text(gowt.Search(), gowt.Hint("Search in Documents"), gowt.Size(gowt.Scaled(220), 0))),
		gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
			name := t.Column("Name", gowt.Scaled(360), gowt.Fit())
			t.Column("Date Modified", gowt.Scaled(200), gowt.Fit())
			t.Column("Size", gowt.Scaled(100), gowt.Right(), gowt.Fit())
			t.Column("Kind", gowt.Scaled(160), gowt.Fit())
			folder := icons.Get(app, "folder", icons.Size(18))
			file := icons.Get(app, "file", icons.Size(18))
			for _, e := range entries {
				r := t.Row(e.name, e.modified, e.size, e.kind)
				if e.dir {
					r.SetImageAt(0, folder)
				} else {
					r.SetImageAt(0, file)
				}
			}
			t.SetSort(name, gowt.SortAscending)
			t.Select(5)
		})),
		gowt.HBox(gowt.GrowX(), gowt.Padding(gowt.Spacing.SM),
			gowt.Label("12 items, 48.2 GB available", kit.Faint(), gowt.Typeface(gowt.Type.Caption))))
}
