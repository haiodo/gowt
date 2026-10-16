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
	"github.com/haiodo/gowt/rx"
)

func label(dark bool) string {
	if dark {
		return "system theme: dark"
	}
	return "system theme: light"
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		theme := rx.New(label(app.Dark()))
		w := app.Window("themecheck", gowt.Label("", gowt.GrowX(), gowt.Size(240, 0), gowt.BindText(theme)))
		fmt.Println(label(app.Dark()))
		app.OnThemeChange(func(dark bool) {
			theme.Set(label(dark))
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
