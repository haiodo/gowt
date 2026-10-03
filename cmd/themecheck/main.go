// Command themecheck shows the system theme and logs every change: switch light/dark in system
// settings while it runs and the label and stdout must follow (Linux: GOWT_THEMECHECK_SECS=n exits after n seconds).
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt"
)

func label(dark bool) string {
	if dark {
		return "system theme: dark"
	}
	return "system theme: light"
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("themecheck")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 20, Spacing: 8})
		l := w.Label(label(app.Dark()), gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 240}))
		fmt.Println(label(app.Dark()))
		app.OnThemeChange(func(dark bool) {
			l.SetText(label(dark))
			fmt.Println(label(dark))
		})
		if s, _ := strconv.Atoi(os.Getenv("GOWT_THEMECHECK_SECS")); s > 0 {
			app.After(time.Duration(s)*time.Second, app.Quit)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
