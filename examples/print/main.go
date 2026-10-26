// Command print prints a small gfx scene through the system print dialog and exports it as PDF.
package main

import (
	"log"
	"os"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
	"github.com/haiodo/gowt/gfx/printing"
)

// scene is a titled box with a line under it.
func scene() *gfx.Scene {
	r := gfx.NewRecorder()
	r.FillRRect(gfx.R(0, 0, 400, 240), 16, gfx.RGB(235, 242, 255))
	r.StrokeRRect(gfx.R(0, 0, 400, 240), 16, gfx.StrokeStyle{Width: 2}, gfx.RGB(40, 90, 200))
	r.Text("Printed and exported", 24, 60, gfx.Font{Size: 28, Weight: 600}, gfx.Black)
	p := &gfx.Path{}
	p.MoveTo(24, 90)
	p.LineTo(376, 90)
	r.StrokePath(p, gfx.StrokeStyle{Width: 1, Dash: []float64{6, 3}}, gfx.RGB(40, 90, 200))
	s := gfx.NewScene()
	n := gfx.NewNode(1)
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, r.Finish())
	})
	return s
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		s := scene()
		w := app.Window("Print")
		var status *gowt.LabelView
		w.Add(
			gowt.Button("Print...", gowt.OnClick(func() {
				job, ok := w.PrintDialog(gowt.PrintOptions{Title: "Scene", Margin: 36})
				if !ok {
					return
				}
				err := job.Print(1, func(p *gowt.PrintPage) {
					if err := printing.Scene(p, s, nil, gfx.Rect{}, printing.Options{}); err != nil {
						status.SetText(err.Error())
					}
				})
				if err != nil {
					status.SetText(err.Error())
				}
			})),
			gowt.Button("Export PDF...", gowt.OnClick(func() {
				paths := w.FileDialog(gowt.FileDialog{Title: "Export PDF", Save: true, Name: "scene.pdf"})
				if len(paths) == 0 {
					return
				}
				f, err := os.Create(paths[0])
				if err == nil {
					// A4, 36 point margin: the scene is fitted and centred on the page.
					err = offscreen.PDF(f, s, nil, gfx.PDFOptions{PageW: 595, PageH: 842, Margin: 36})
					f.Close()
				}
				if err != nil {
					status.SetText(err.Error())
					return
				}
				status.SetText("Saved " + paths[0])
			})),
			gowt.Label("", gowt.GrowX(), gowt.Ref(&status)))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
