// Command dashboard is a metrics overview: summary cards, progress bars, a Canvas bar chart,
// a recent activity table and links. Static data, fixed 1280x800 window (docs/look-check.md).
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
)

var weekly = []int{42, 55, 48, 63, 71, 58, 80, 92, 76, 88, 95, 104}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		w := app.Window("Overview")
		kit.Size(w)
		w.SetMinSize(900, 600)
		w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(gowt.Scaled(24)), gowt.Gap(gowt.Spacing.LG),
			gowt.HBox(gowt.Gap(gowt.Spacing.MD), gowt.GrowX(),
				gowt.VBox(gowt.Gap(gowt.Spacing.XS), gowt.Label("Overview", gowt.Typeface(gowt.Type.Title)), gowt.Label("Last 12 weeks, all regions", kit.Faint())),
				gowt.Spacer(),
				gowt.Combo([]string{"Last 12 weeks", "Last 30 days", "This year"}, gowt.ReadOnly(), gowt.Selected(0)),
				gowt.Button("Export")),
			gowt.HBox(gowt.Gap(gowt.Spacing.LG), gowt.GrowX(),
				card("Revenue", "$84,200", "+12.4% from last period", kit.Green, 78),
				card("Active users", "12,480", "+3.1% from last period", kit.Green, 62),
				card("Open tickets", "37", "-8 from last period", kit.Green, 35),
				card("Error rate", "0.42%", "+0.05% from last period", kit.Red, 20)),
			gowt.HBox(gowt.Gap(gowt.Spacing.LG), gowt.Grow(),
				gowt.VBox(gowt.Gap(gowt.Spacing.SM), gowt.Grow(),
					gowt.Label("Weekly signups", gowt.Typeface(gowt.Type.Subtitle)),
					gowt.Section(gowt.Grow(), gowt.Canvas(chart, gowt.Grow(), gowt.Size(gowt.Scaled(400), gowt.Scaled(240)), gowt.Typeface(gowt.Type.Caption)))),
				gowt.VBox(gowt.Gap(gowt.Spacing.SM), gowt.Size(gowt.Scaled(440), 0),
					gowt.Label("Recent activity", gowt.Typeface(gowt.Type.Subtitle)),
					activity(),
					gowt.HBox(gowt.Gap(gowt.Spacing.MD),
						gowt.Link(`<a href="https://example.com/reports">All reports</a>`),
						gowt.Link(`<a href="https://example.com/status">System status</a>`))))))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func card(title, value, note string, noteColor gowt.RGB, pct int) gowt.Option {
	return gowt.Section(gowt.GrowX(),
		gowt.VBox(gowt.Gap(gowt.Spacing.SM), gowt.Padding(gowt.Spacing.XS),
			gowt.Label(title, kit.Faint()),
			gowt.Label(value, gowt.Typeface(gowt.Type.LargeTitle)),
			gowt.Progress(100, gowt.GrowX(), gowt.Value(pct)),
			gowt.Label(note, gowt.Typeface(gowt.Type.Caption), gowt.Foreground(noteColor))))
}

func activity() gowt.Option {
	return gowt.Table(gowt.Grow(), gowt.Init(func(t *gowt.TableView) {
		t.Column("Event", gowt.Scaled(250), gowt.Fit())
		t.Column("Who", gowt.Scaled(110), gowt.Fit())
		t.Column("When", gowt.Scaled(60), gowt.Right(), gowt.Fit())
		for _, r := range [][]string{
			{"Plan upgraded to Team", "Priya Raman", "2h"},
			{"New project created", "Tomas Weber", "3h"},
			{"Invoice 2026-09 paid", "Accounts", "5h"},
			{"Invite sent to 4 people", "Hannah Cole", "1d"},
			{"API key rotated", "Alex Morgan", "1d"},
			{"Report exported", "Mei Tanaka", "2d"},
		} {
			t.Row(r...)
		}
	}))
}

func chart(g *gowt.GC) {
	w, h := g.Size()
	labelW, labelH := g.TextSize("120")
	left, bottom, top := labelW+12, labelH+14, labelH
	maxV := 120
	plotH := h - bottom - top
	g.SetColor(gowt.RGB{R: 200, G: 200, B: 205})
	for i := 0; i <= 4; i++ {
		y := top + plotH*i/4
		g.Line(left, y, w, y)
		g.SetColor(gowt.RGB{R: 140, G: 140, B: 145})
		g.Text(fmt.Sprint(maxV-maxV*i/4), 4, y-labelH/2)
		g.SetColor(gowt.RGB{R: 200, G: 200, B: 205})
	}
	slot := (w - left) / len(weekly)
	g.SetFill(kit.Blue)
	for i, v := range weekly {
		bh := plotH * v / maxV
		g.FillRect(left+i*slot+slot/5, top+plotH-bh, slot*3/5, bh)
		g.SetColor(gowt.RGB{R: 140, G: 140, B: 145})
		g.Text(fmt.Sprint("W", i+1), left+i*slot+slot/5, h-bottom+4)
	}
}
