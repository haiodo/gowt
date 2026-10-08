// Command backdropdemo switches the window material (none / translucent = Acrylic / glass = Mica) and
// the rounded corners. Windows 11 22H2+ shows the material; elsewhere the calls are no-ops.
// GOWT_DARK_CONTENT=1 turns on the opt-in dark content theme (Windows).
// GOWT_BACKDROP_SECS=n exits after n seconds.
package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		if os.Getenv("GOWT_DARK_CONTENT") == "1" {
			app.SetDarkContent(true)
		}
		w := app.Window("backdropdemo")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 20, Spacing: 8})
		fill := gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 280})
		// Pure black text over the material turns transparent; near-black stays visible.
		w.Label("Material behind the window", fill).Unwrap().SetForeground(swt.NewColorRedGreenBlue(24, 24, 24))
		w.Button("None", func() { w.SetBackdrop(gowt.BackdropNone) }, fill)
		w.Button("Translucent (Acrylic)", func() { w.SetBackdrop(gowt.BackdropTranslucent) }, fill)
		w.Button("Glass (Mica)", func() { w.SetBackdrop(gowt.BackdropGlass) }, fill)
		w.Button("Square corners", func() { w.SetRoundedCorners(false) }, fill)
		w.Button("Round corners", func() { w.SetRoundedCorners(true) }, fill)
		w.Text(fill)
		if s, _ := strconv.Atoi(os.Getenv("GOWT_BACKDROP_SECS")); s > 0 {
			app.After(time.Duration(s)*time.Second, app.Quit)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
