// Command app is the fixture of qatest's end-to-end test: controls with IDs and the QA handlers.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		clicks := rx.New(0)
		off := rx.New(0)
		name := rx.New("")
		agree := rx.New(false)
		pick := rx.New(-1)
		report := rx.New("")
		text := rx.Compute(func() string { return fmt.Sprintf("clicks: %d", clicks.Get()) })
		w := app.Window("QA fixture",
			gowt.Label("", gowt.BindText(text), gowt.ID("count")),
			gowt.Button("Click", gowt.ID("click"), gowt.OnClick(func() { clicks.Set(clicks.Get() + 1) })),
			gowt.Button("Off", gowt.ID("off"), gowt.Disabled(), gowt.OnClick(func() { off.Set(off.Get() + 1) })),
			gowt.Text(gowt.ID("name"), gowt.OnChange(func(s string) { name.Set(s) })),
			gowt.Button("Agree", gowt.Check(), gowt.ID("agree"), gowt.OnClick(func() { agree.Set(!agree.Get()) })),
			gowt.List([]string{"a", "b", "c"}, gowt.ID("list"), gowt.OnSelect(func(i int) { pick.Set(i) })),
			gowt.Button("Load", gowt.ID("load"), gowt.OnClick(func() {
				app.Job("Download report", func(ctx context.Context, p *gowt.JobProgress) error {
					p.Total(3)
					for range 3 {
						time.Sleep(500 * time.Millisecond)
						p.Step(1)
					}
					return nil
				}).Then(func(error) { report.Set("report ready") })
			})),
			gowt.Label("", gowt.BindText(report), gowt.ID("report")),
			gowt.Button("Twin"), gowt.Button("Twin"))
		w.Show()
		// Perpetual work: listed by Jobs, never waited for by settle.
		app.Job("poll", func(ctx context.Context, p *gowt.JobProgress) error {
			<-ctx.Done()
			return nil
		}, gowt.BackgroundJob())
		qa.Handle("reset", func(qa.Args) error {
			clicks.Set(0)
			off.Set(0)
			report.Set("")
			return nil
		})
		qa.Handle("add", func(a qa.Args) error { clicks.Set(clicks.Get() + a.Int("n")); return nil })
		qa.HandleValue("get", func(qa.Args) (any, error) { return clicks.Get(), nil })
		qa.HandleValue("form", func(qa.Args) (any, error) {
			return map[string]any{"off": off.Get(), "name": name.Get(), "agree": agree.Get(), "pick": pick.Get()}, nil
		})
		qa.Handle("fail", func(qa.Args) error { return fmt.Errorf("asked to fail") })
		qa.Handle("panic", func(qa.Args) error { panic("asked to panic") })
	})
	if err != nil {
		log.Fatal(err)
	}
}
