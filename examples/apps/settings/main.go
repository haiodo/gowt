// Command settings is a System Settings style window: category sidebar, grouped sections of form
// rows with Text, Combo, Switch, Check, Radio, Slider and Spinner, and primary and destructive
// buttons. Static data, fixed 1280x800 window (docs/look-check.md).
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		w := app.Window("Appearance")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(800, 500)

		var sidebar, content *gowt.Panel
		w.Add(sidebarPane(app, &sidebar), contentPane(&content))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Title: "Back/Forward", Nav: true, Group: []look.ToolbarItem{
				{Title: "Back", Symbol: "chevron.backward", OnClick: func() {}},
				{Title: "Forward", Symbol: "chevron.forward", OnClick: func() {}}}},
			look.ToolbarItem{Space: true},
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
		gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
			t.Column("", gowt.Scaled(220))
			t.ShowHeader(false)
			t.ShowLines(false)
			cats := []struct {
				name string
				icon *gowt.Image
			}{
				{"Wi-Fi", kit.Symbol(app, kit.PathWifi, 18, kit.Blue)},
				{"Notifications", kit.Symbol(app, kit.PathBell, 18, kit.Red)},
				{"General", icons.Get(app, "settings", icons.Size(18))},
				{"Appearance", kit.Symbol(app, kit.PathPalette, 18, kit.Purple)},
				{"Privacy & Security", kit.Symbol(app, kit.PathShield, 18, kit.Teal)},
				{"Accounts", icons.Get(app, "user", icons.Size(18))},
				{"Keyboard", icons.Get(app, "edit", icons.Size(18))},
				{"Software Update", icons.Get(app, "refresh", icons.Size(18))},
			}
			for _, c := range cats {
				t.Row(c.name).SetImageAt(0, c.icon)
			}
			t.Select(3)
		})))
}

// A settings row: the label at the start, the control at the end. The Account form uses Fields.
func row(label string, control gowt.Option) gowt.Option {
	return gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(), gowt.Label(label, gowt.GrowX(), gowt.Place(gowt.AlignFill, gowt.AlignCenter)), gowt.VBox(gowt.Padding(0), control))
}

func contentPane(ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(gowt.Scaled(24)), gowt.Gap(gowt.Spacing.MD),
		gowt.Scroll(gowt.Grow(),
			gowt.VBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(), gowt.Padding(0),
				gowt.Label("Appearance", gowt.Typeface(gowt.Type.Title)),
				gowt.Label("Style", kit.Faint()),
				gowt.Section(gowt.GrowX(),
					row("Theme", gowt.Combo([]string{"Light", "Dark", "Automatic"}, gowt.ReadOnly(), gowt.Selected(2))),
					row("Accent colour", gowt.Combo([]string{"Blue", "Purple", "Green", "Orange"}, gowt.ReadOnly(), gowt.Selected(0))),
					row("Show scroll bars", gowt.HBox(gowt.Gap(gowt.Spacing.SM),
						gowt.Button("Automatically", gowt.Radio(), gowt.Checked(true)),
						gowt.Button("Always", gowt.Radio())))),
				gowt.Label("Display", kit.Faint()),
				gowt.Section(gowt.GrowX(),
					row("Text size", gowt.Scale(8, 20, gowt.Value(13), gowt.Size(gowt.Scaled(220), 0))),
					row("Sidebar icon size", gowt.Spinner(12, 32, gowt.Value(18))),
					row("Use large toolbar icons", gowt.Switch("", gowt.Checked(true))),
					row("Show path in the title", gowt.Button("", gowt.Check(), gowt.Checked(true)))),
				gowt.Label("Account", kit.Faint()),
				gowt.Section(gowt.GrowX(),
					gowt.Fields(gowt.GrowX(), gowt.Gap(gowt.Spacing.MD),
						gowt.Field("Display name", gowt.Text(gowt.Value("Alex Morgan"), gowt.GrowX())),
						gowt.Field("Email", gowt.Text(gowt.Hint("name@example.com"), gowt.GrowX())),
						gowt.Field("Password", gowt.Text(gowt.Password(), gowt.Value("secret-pass"), gowt.GrowX())))),
				gowt.HBox(gowt.Gap(gowt.Spacing.SM), gowt.GrowX(),
					gowt.Button("Delete Account...", gowt.Destructive()),
					gowt.Spacer(),
					gowt.Button("Cancel"),
					gowt.Button("Save Changes", gowt.Primary())),
				gowt.Link(`<a href="https://example.com/privacy">About privacy and this setting</a>`))))
}
