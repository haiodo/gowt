// Command events shows callbacks, the UI thread rule and background work reporting back.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		var field *gowt.TextView
		var echo *gowt.LabelView
		var progress *gowt.ProgressView
		var start *gowt.ButtonView
		var clock *gowt.LabelView
		w := app.Window("Events",
			gowt.Grid(2, gowt.GrowX(),
				gowt.Text(gowt.GrowX(), gowt.Ref(&field)),
				gowt.Label("", gowt.GrowX(), gowt.Size(160, 0), gowt.Ref(&echo)),

				gowt.Progress(100, gowt.GrowX(), gowt.Span(2), gowt.Ref(&progress)),
				// Slow work goes to a goroutine. Widgets are touched only on the UI thread,
				// so the result comes back through app.Async.
				gowt.Button("Start work", func() {
					start.SetEnabled(false)
					go func() {
						for i := 1; i <= 100; i++ {
							time.Sleep(30 * time.Millisecond) // stands for slow work
							app.Async(func() { progress.SetValue(i) })
						}
						app.Async(func() {
							start.SetEnabled(true)
							echo.SetText("work done")
						})
					}()
				}, gowt.Ref(&start)),
				gowt.Label(time.Now().Format("15:04:05"), gowt.GrowX(), gowt.Ref(&clock)),
			),
		)

		// Callbacks are plain funcs; calling On* again adds one more listener.
		field.OnChange(func(s string) { echo.SetText("typed: " + s) })
		field.OnActivate(func() { echo.SetText("Enter pressed") })

		// Every repeats on the UI thread until the stop func it returns is called; After runs once.
		app.Every(time.Second, func() { clock.SetText(time.Now().Format("15:04:05")) })

		// OnClose can veto closing: return false to keep the window.
		closes := 0
		w.OnClose(func() bool {
			closes++
			echo.SetText(fmt.Sprint("close attempt ", closes))
			return closes > 1
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
