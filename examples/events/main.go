// Command events shows callbacks, the UI thread rule and background work reporting back.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		echo := rx.New("")
		clock := rx.New(time.Now().Format("15:04:05"))
		idle := rx.New(true)
		done := rx.New(0)
		w := app.Window("Events",
			gowt.Grid(2, gowt.GrowX(),
				// Callbacks are plain funcs; calling On* again adds one more listener.
				gowt.Text(gowt.GrowX(),
					gowt.OnChange(func(s string) { echo.Set("typed: " + s) }),
					gowt.OnActivate(func() { echo.Set("Enter pressed") })),
				gowt.Label("", gowt.GrowX(), gowt.Size(160, 0), gowt.BindText(echo)),

				gowt.Progress(100, gowt.GrowX(), gowt.Span(2), gowt.Bind(done)),
				// Slow work goes to a goroutine. Widgets are touched only on the UI thread,
				// so the result comes back through app.Async.
				gowt.Button("Start work", gowt.OnClick(func() {
					idle.Set(false)
					go func() {
						for i := 1; i <= 100; i++ {
							time.Sleep(30 * time.Millisecond) // stands for slow work
							app.Async(func() { done.Set(i) })
						}
						app.Async(func() {
							idle.Set(true)
							echo.Set("work done")
						})
					}()
				}), gowt.BindEnabled(idle)),
				gowt.Label("", gowt.GrowX(), gowt.BindText(clock))))

		// Every repeats on the UI thread until the stop func it returns is called; After runs once.
		app.Every(time.Second, func() { clock.Set(time.Now().Format("15:04:05")) })

		// OnClose can veto closing: return false to keep the window.
		closes := 0
		w.OnClose(func() bool {
			closes++
			echo.Set(fmt.Sprint("close attempt ", closes))
			return closes > 1
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
