// Command glassdemo shows the macOS 26 look options: window backdrop, glass panel, glass buttons,
// full-size content. GOWT_TOOLBAR=1 adds a native unified toolbar, a glass group and a background
// extension (the toolbar falls back to a gowt.ToolBar where it does not exist). GOWT_CLASSIC=1 asks for the pre-26 look; GOWT_BACKDROP=none|translucent|glass
// picks the window material (default glass); GOWT_GLASSDEMO_SECS=n exits after n seconds;
// GOWT_GLASSDEMO_DUMP=path writes the window view tree (frame view down, 4 levels) and exits.
package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
)

func main() {
	if os.Getenv("GOWT_CLASSIC") == "1" {
		look.Classic()
	}
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("glassdemo", gowt.Padding(24), gowt.Gap(10))
		look.SetFullSizeContent(w, true)
		look.SetBackdrop(w, map[string]look.Backdrop{
			"none": look.BackdropNone, "translucent": look.BackdropTranslucent, "": look.BackdropGlass, "glass": look.BackdropGlass,
		}[os.Getenv("GOWT_BACKDROP")])
		toolbar := os.Getenv("GOWT_TOOLBAR") == "1"
		if toolbar {
			note := func() { log.Print("toolbar item clicked") }
			if !look.SetToolbar(w, look.ToolbarItem{Title: "Add", Symbol: "plus", OnClick: note},
				look.ToolbarItem{Space: true},
				look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up", OnClick: note}) {
				w.Add(gowt.ToolBar(gowt.Init(func(tb *gowt.ToolBarView) { tb.Item("Add", note) })))
			}
			look.SetGlassGroup(w, 4)
		}
		w.Add(
			gowt.Label("Window backdrop, full-size content", gowt.GrowX(), gowt.Size(320, 0)),
			gowt.Grid(2, gowt.Padding(4),
				gowt.Label("Nested panel:"),
				gowt.Text()),
			gowt.Group("Glass group", gowt.Init(func(g *gowt.GroupView) { look.SetGlass(g, true) }),
				gowt.Grid(2,
					gowt.Button("Glass", look.GlassButton()),
					gowt.Button("Plain"),
					gowt.Button("Check, no glass bezel", gowt.Check(), look.GlassButton()))),
			gowt.VBox(gowt.Padding(12), gowt.Init(func(p *gowt.Panel) { look.SetGlass(p, true) }), gowt.Label("Glass panel")),
			gowt.Button("Glass button", look.GlassButton()))
		if s, _ := strconv.Atoi(os.Getenv("GOWT_GLASSDEMO_SECS")); s > 0 {
			app.After(time.Duration(s)*time.Second, app.Quit)
		}
		w.Show()
		if path := os.Getenv("GOWT_GLASSDEMO_DUMP"); path != "" {
			app.After(700*time.Millisecond, func() {
				if err := os.WriteFile(path, []byte(dumpViews(w)), 0o644); err != nil {
					log.Print(err)
				}
				app.Quit()
			})
		}
	})
	if err != nil {
		log.Fatal(err)
	}
}
