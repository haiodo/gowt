// Command tracker is an issue tracker: a filter bar, tabs, a table of issues with status
// dots and an inspector panel with the selected issue. Static data, fixed 1280x800 window
// (docs/look-check.md).
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/look"
)

type issue struct {
	id, title, status, assignee, priority, updated string
}

var statuses = map[string]gowt.RGB{"Todo": kit.Gray, "In Progress": kit.Orange, "In Review": kit.Purple, "Done": kit.Green}

var issues = []issue{
	{"APP-101", "Sidebar selection loses focus after resize", "In Progress", "Priya Raman", "High", "2h ago"},
	{"APP-102", "Add keyboard shortcuts to the inspector", "Todo", "Tomas Weber", "Medium", "5h ago"},
	{"APP-103", "Table header sort indicator is clipped", "In Review", "Hannah Cole", "Medium", "Yesterday"},
	{"APP-104", "Export the report as PDF", "Todo", "Mei Tanaka", "Low", "Yesterday"},
	{"APP-105", "Crash when the window is moved between displays", "In Progress", "Alex Morgan", "Urgent", "Yesterday"},
	{"APP-106", "Update the onboarding illustrations", "Done", "Ola Nordmann", "Low", "Mon"},
	{"APP-107", "Dark mode contrast in the settings page", "In Review", "Priya Raman", "High", "Mon"},
	{"APP-108", "Remember the last used filter", "Done", "Tomas Weber", "Medium", "Sun"},
	{"APP-109", "Search is slow with more than 10 000 items", "Todo", "Alex Morgan", "High", "Sat"},
	{"APP-110", "Localise the date format in the list", "Done", "Hannah Cole", "Low", "Fri"},
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		w := app.Window("Issues")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(900, 500)

		var content, insp *gowt.Panel
		w.Add(contentPane(app, &content), inspectorPane(&insp))
		look.SetToolbar(w,
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "New Issue", Symbol: "plus", OnClick: func() {}},
			look.ToolbarItem{Title: "Search", OnSearch: func(string) {}},
			look.ToolbarItem{Title: "Hide Inspector", Symbol: "sidebar.right", OnClick: func() {}})
		look.SetInspector(w, content, insp, gowt.Scaled(340))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func contentPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(gowt.Spacing.MD), gowt.Gap(gowt.Spacing.MD),
		gowt.Tabs(gowt.Grow(), gowt.Selected(0),
			gowt.Tab("All issues", filterBar(), table(app)),
			gowt.Tab("Assigned to me"),
			gowt.Tab("Recently closed")))
}

func filterBar() gowt.Option {
	return gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(),
		gowt.Combo([]string{"Any status", "Todo", "In Progress", "In Review", "Done"}, gowt.ReadOnly(), gowt.Selected(0)),
		gowt.Combo([]string{"Any assignee", "Alex Morgan", "Priya Raman"}, gowt.ReadOnly(), gowt.Selected(0)),
		gowt.Combo([]string{"Any priority", "Urgent", "High", "Medium", "Low"}, gowt.ReadOnly(), gowt.Selected(0)),
		gowt.Spacer(),
		gowt.Text(gowt.Search(), gowt.Hint("Filter issues"), gowt.Size(gowt.Scaled(220), 0)))
}

func table(app *gowt.App) gowt.Option {
	return gowt.Table(gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
		t.Column("ID", gowt.Scaled(92), gowt.Fit())
		title := t.Column("Title", gowt.Scaled(400), gowt.Fit())
		t.Column("Status", gowt.Scaled(120), gowt.Fit())
		t.Column("Assignee", gowt.Scaled(130), gowt.Fit())
		t.Column("Priority", gowt.Scaled(80), gowt.Fit())
		t.Column("Updated", gowt.Scaled(92), gowt.Right(), gowt.Fit())
		for _, i := range issues {
			r := t.Row(i.id, i.title, i.status, i.assignee, i.priority, i.updated)
			r.SetImageAt(2, kit.Dot(app, statuses[i.status], 12))
		}
		t.SetSort(title, gowt.SortNone)
		t.Select(4)
	}))
}

func inspectorPane(ref **gowt.Panel) gowt.Option {
	i := issues[4]
	field := func(name string, v gowt.Option) gowt.Option {
		return gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(), gowt.Label(name, kit.Faint(), gowt.Size(gowt.Scaled(80), 0), gowt.Place(gowt.AlignStart, gowt.AlignCenter)), v)
	}
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(gowt.Spacing.PageMargin), gowt.Gap(gowt.Spacing.MD),
		gowt.Label(i.id, kit.Faint(), gowt.Typeface(gowt.Type.Caption)),
		gowt.Label(i.title, gowt.Typeface(gowt.Type.Subtitle), gowt.Wrap(), gowt.GrowX()),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), kit.Pill("In Progress", gowt.Scaled(92), gowt.Scaled(24), kit.Tint(kit.Orange), kit.Orange), kit.Pill("Urgent", gowt.Scaled(64), gowt.Scaled(24), kit.Tint(kit.Red), kit.Red)),
		gowt.Separator(gowt.GrowX()),
		field("Status", gowt.Combo([]string{"Todo", "In Progress", "In Review", "Done"}, gowt.ReadOnly(), gowt.Selected(1), gowt.GrowX())),
		field("Assignee", gowt.Combo([]string{"Alex Morgan", "Priya Raman"}, gowt.ReadOnly(), gowt.Selected(0), gowt.GrowX())),
		field("Due", gowt.Label("30 Oct 2026")),
		field("Estimate", gowt.Spinner(0, 40, gowt.Value(5))),
		gowt.Label("Description", gowt.Typeface(gowt.Type.BodyStrong)),
		gowt.Text(gowt.Multiline(), gowt.ReadOnly(), gowt.Border(), gowt.GrowX(), gowt.Size(0, gowt.Scaled(120)),
			gowt.Value("The window crashes when it is dragged from the laptop display to an external monitor with a different scale factor.")),
		gowt.Link(`<a href="https://example.com/issues/105">Open in browser</a>`),
		gowt.Spacer(),
		gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(), gowt.Spacer(),
			gowt.Button("Close Issue", gowt.Destructive()),
			gowt.Button("Save", gowt.Primary())))
}
