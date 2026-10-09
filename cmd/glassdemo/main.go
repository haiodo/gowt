// Command glassdemo shows the macOS 26 look options: window backdrop, glass panel, glass buttons,
// full-size content. GOWT_CLASSIC=1 asks for the pre-26 look; GOWT_BACKDROP=none|translucent|glass
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
		w := app.Window("glassdemo")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 24, Spacing: 10})
		look.SetFullSizeContent(w, true)
		look.SetBackdrop(w, map[string]look.Backdrop{
			"none": look.BackdropNone, "translucent": look.BackdropTranslucent, "": look.BackdropGlass, "glass": look.BackdropGlass,
		}[os.Getenv("GOWT_BACKDROP")])
		w.Label("Window backdrop, full-size content", gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 320}))
		p := w.Panel()
		p.SetLayout(gowt.Grid{Columns: 2, Margin: 4, Spacing: 8})
		p.Label("Nested panel:")
		p.Text()
		g := w.Group("Glass group")
		g.SetLayout(gowt.Grid{Columns: 2, Margin: 12, Spacing: 8})
		look.SetGlass(g, true)
		g.Button("Glass", nil, look.GlassButton())
		g.Button("Plain", nil)
		g.Button("Check, no glass bezel", nil, gowt.Check(), look.GlassButton())
		gp := w.Panel()
		gp.SetLayout(gowt.Grid{Columns: 1, Margin: 12})
		look.SetGlass(gp, true)
		gp.Label("Glass panel")
		w.Button("Glass button", nil, look.GlassButton())
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
