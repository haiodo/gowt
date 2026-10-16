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
		w := app.Window("Events")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})

		// Callbacks are plain funcs; calling On* again adds one more listener.
		field := w.Text(gowt.GrowX())
		echo := w.Label("", gowt.GrowX(), gowt.Size(160, 0))
		field.OnChange(func(s string) { echo.SetText("typed: " + s) })
		field.OnActivate(func() { echo.SetText("Enter pressed") })

		// Slow work goes to a goroutine. Widgets are touched only on the UI thread,
		// so the result comes back through app.Async.
		progress := w.Progress(100, gowt.GrowX(), gowt.Span(2))
		var start *gowt.Button
		start = w.Button("Start work", func() {
			gowt.SetEnabled(start, false)
			go func() {
				for i := 1; i <= 100; i++ {
					time.Sleep(30 * time.Millisecond) // stands for slow work
					app.Async(func() { progress.SetValue(i) })
				}
				app.Async(func() {
					gowt.SetEnabled(start, true)
					echo.SetText("work done")
				})
			}()
		})

		// After runs once on the UI thread; re-arm it for a periodic tick.
		clock := w.Label("", gowt.GrowX())
		var tick func()
		tick = func() {
			clock.SetText(time.Now().Format("15:04:05"))
			app.After(time.Second, tick)
		}
		tick()

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
