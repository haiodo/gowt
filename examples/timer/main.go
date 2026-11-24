// Command timer is the 7GUIs timer task: a progress bar follows a timer set in seconds.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func timer(app *gowt.App) gowt.Option {
	total := rx.New(10)  // seconds
	elapsed := rx.New(0) // milliseconds
	stop := app.Every(10*time.Millisecond, func() {
		limit := total.Get() * 1000
		if elapsed.Get() >= limit {
			return
		}
		elapsed.Set(min(elapsed.Get()+10, limit))
	})
	progress := rx.Compute(func() int {
		limit := total.Get() * 1000
		if limit == 0 {
			return 0
		}
		return elapsed.Get() * 1000 / limit
	})
	readout := rx.Compute(func() string {
		return fmt.Sprintf("%.1f s / %d s", float64(elapsed.Get())/1000, total.Get())
	})
	return gowt.VBox(
		gowt.Progress(1000, gowt.GrowX(), gowt.BindValue(progress)),
		gowt.Label("", gowt.BindText(readout)),
		gowt.HBox(gowt.GrowX(),
			gowt.Label("Seconds:"),
			gowt.Scale(0, 30, gowt.Value(10), gowt.GrowX(), gowt.OnChange(func(v int) {
				total.Set(v)
				if elapsed.Get() > v*1000 {
					elapsed.Set(v * 1000)
				}
			})),
			gowt.Button("Reset", gowt.OnClick(func() { elapsed.Set(0) }))),
		gowt.Init(func(p *gowt.Panel) { p.OnDispose(stop) }),
	)
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Timer", timer(app))
		w.SetSize(420, 140)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
