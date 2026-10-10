// Command graphics draws on a Canvas and shows an Image built in memory.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"time"

	"github.com/haiodo/gowt"
)

// gradient is a PNG made in memory; App.LoadImage reads the same formats from a file.
func gradient() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 160, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Graphics")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 8, Spacing: 8})

		// The caller owns an Image; the App disposes what is left when Run returns.
		tile, err := app.ImageFrom(bytes.NewReader(gradient()))
		if err != nil {
			log.Fatal(err)
		}

		angle := 0
		canvas := w.Canvas(func(g *gowt.GC) {
			// The GC lives only inside this callback; draw the whole area every time.
			g.SetFill(gowt.RGB{R: 245, G: 245, B: 245})
			g.FillRect(0, 0, 400, 200)
			g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
			g.SetLineWidth(2)
			g.Rect(10, 10, 380, 180)
			g.Line(10, 10, 390, 190)
			g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
			g.FillOval(30+angle%200, 60, 80, 80)
			g.Image(tile, 300, 20)
			g.Text(fmt.Sprint("frame ", angle/4), 20, 20)
		}, gowt.Cell(gowt.GridCell{Width: 400, Height: 200}))

		var step func()
		step = func() {
			angle += 4
			canvas.Redraw()
			app.After(40*time.Millisecond, step)
		}
		step()

		w.Label("Image from memory:")
		l := w.Label("")
		l.SetImage(tile)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
