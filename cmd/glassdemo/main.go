// Command glassdemo shows the macOS 26 look options: window backdrop, glass panel, glass buttons,
// full-size content. GOWT_CLASSIC=1 asks for the pre-26 look; GOWT_BACKDROP=none|translucent|glass
// picks the window material (default glass); GOWT_GLASSDEMO_SECS=n exits after n seconds.
package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt"
)

func main() {
	if os.Getenv("GOWT_CLASSIC") == "1" {
		gowt.ClassicLook()
	}
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("glassdemo")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 24, Spacing: 10})
		w.SetFullSizeContent(true)
		w.SetBackdrop(map[string]gowt.Backdrop{
			"none": gowt.BackdropNone, "translucent": gowt.BackdropTranslucent, "": gowt.BackdropGlass, "glass": gowt.BackdropGlass,
		}[os.Getenv("GOWT_BACKDROP")])
		w.Label("Window backdrop, full-size content", gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 320}))
		g := w.Group("Glass panel")
		g.SetLayout(gowt.Grid{Columns: 2, Margin: 12, Spacing: 8})
		g.SetGlass(true)
		g.Button("Glass", nil, gowt.GlassButton())
		g.Button("Plain", nil)
		w.Button("Glass button", nil, gowt.GlassButton())
		if s, _ := strconv.Atoi(os.Getenv("GOWT_GLASSDEMO_SECS")); s > 0 {
			app.After(time.Duration(s)*time.Second, app.Quit)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
