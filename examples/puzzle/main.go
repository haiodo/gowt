// Command puzzle is a 4x4 slide puzzle drawn on one Canvas.
package main

import (
	"cmp"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/haiodo/gowt"
)

const (
	animTick = 16 * time.Millisecond
	animTime = 170 * time.Millisecond
	autoTick = 300 * time.Millisecond
	boardPad = 12 // points between the board and the canvas edge
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		b := newBoard()
		moves := 0

		var canvas *gowt.CanvasView
		var movesLabel, leftLabel *gowt.LabelView
		var auto *gowt.SwitchView

		var animFrom [boardSize * boardSize][2]int // cells each tile starts away from its cell
		var animStart time.Time
		var animOn bool
		var animStop, autoStop func()
		var tick func()

		stopAnim := func() {
			if animStop != nil {
				animStop()
				animStop = nil
			}
			animOn = false
		}
		stopAuto := func() {
			if autoStop != nil {
				autoStop()
				autoStop = nil
			}
			auto.SetChecked(false)
		}
		status := func() {
			movesLabel.SetText(fmt.Sprintf("Moves: %d", moves))
			leftLabel.SetText(fmt.Sprintf("Tiles left: %d", b.tilesLeft()))
		}

		tick = func() {
			if !animOn {
				stopAnim()
				return
			}
			if time.Since(animStart) >= animTime {
				stopAnim()
			}
			canvas.Redraw()
		}

		startSlide := func(hole, cell int) {
			animFrom = [boardSize * boardSize][2]int{}
			dx := cmp.Compare(hole%boardSize, cell%boardSize)
			dy := cmp.Compare(hole/boardSize, cell/boardSize)
			for c := hole; c != cell; c -= dx + dy*boardSize {
				animFrom[c] = [2]int{-dx, -dy}
			}
			animStart = time.Now()
			animOn = true
			if animStop == nil {
				animStop = app.Every(animTick, tick)
			}
		}

		try := func(cell int) {
			if animOn {
				return
			}
			hole := b.hole()
			if !b.move(cell) {
				return
			}
			moves++
			startSlide(hole, cell)
			status()
			canvas.Redraw()
		}

		autoMove := func() {
			if b.solved() {
				stopAuto()
				return
			}
			if animOn {
				return
			}
			h := b.hole()
			next := make([]int, 0, 4)
			if h/boardSize > 0 {
				next = append(next, h-boardSize)
			}
			if h/boardSize < boardSize-1 {
				next = append(next, h+boardSize)
			}
			if h%boardSize > 0 {
				next = append(next, h-1)
			}
			if h%boardSize < boardSize-1 {
				next = append(next, h+1)
			}
			try(next[rand.IntN(len(next))])
		}

		reset := func() {
			stopAnim()
			b.shuffle()
			moves = 0
			status()
			canvas.Redraw()
		}

		paint := func(g *gowt.GC) {
			w, h := g.Size()
			g.SetFill(gowt.RGB{R: 255, G: 255, B: 255})
			g.FillRect(0, 0, w, h)
			x0, y0, cell := geometry(w, h)
			if cell == 0 {
				return
			}

			t := 1.0
			if animOn {
				t = min(1, float64(time.Since(animStart))/float64(animTime))
			}
			ease := 1 - (1-t)*(1-t)

			g.SetColor(gowt.RGB{})
			g.SetFont(gowt.Font{Size: max(1, cell/3)})
			for i, v := range b.cells {
				if v == 0 {
					continue
				}
				x := x0 + i%boardSize*cell + int(float64(animFrom[i][0]*cell)*(1-ease))
				y := y0 + i/boardSize*cell + int(float64(animFrom[i][1]*cell)*(1-ease))
				g.SetFill(gowt.RGB{R: 173, G: 216, B: 230})
				g.FillRect(x, y, cell, cell)
				g.SetLineWidth(2)
				g.Rect(x, y, cell, cell)
				label := fmt.Sprint(v)
				tw, th := g.TextSize(label)
				g.Text(label, x+(cell-tw)/2, y+(cell-th)/2)
			}

			if b.solved() && !animOn {
				hole := b.hole()
				x := x0 + hole%boardSize*cell
				y := y0 + hole/boardSize*cell
				g.SetFill(gowt.RGB{R: 255, G: 255, B: 255})
				g.SetColor(gowt.RGB{R: 20, G: 130, B: 70})
				g.SetFont(gowt.Font{Size: max(1, cell/5)})
				tw, th := g.TextSize("Solved")
				g.Text("Solved", x+(cell-tw)/2, y+(cell-th)/2)
			}
		}

		w := app.Window("Puzzle")
		w.Add(
			gowt.Canvas(paint, gowt.Grow(), gowt.Ref(&canvas)),
			gowt.HBox(gowt.GrowX(),
				gowt.Label("", gowt.Ref(&movesLabel)),
				gowt.Label("", gowt.Ref(&leftLabel)),
				gowt.Spacer(),
				gowt.Button("Reset", gowt.OnClick(reset)),
				gowt.Switch("Auto play", gowt.OnChange(func(on bool) {
					switch {
					case on && autoStop == nil:
						autoStop = app.Every(autoTick, autoMove)
					case !on:
						stopAuto()
					}
				}), gowt.Ref(&auto))))
		status()

		canvas.OnMouseDown(func(m gowt.Mouse) {
			if m.Button != 1 {
				return
			}
			w, h := canvas.Size()
			cell := cellAt(w, h, m.X, m.Y)
			if cell >= 0 {
				try(cell)
			}
		})
		canvas.OnDispose(func() {
			stopAnim()
			if autoStop != nil {
				autoStop()
			}
		})

		w.SetSize(480, 480)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// geometry puts the largest square that fits in the canvas in its centre and returns its top left
// point and the side of one cell.
func geometry(w, h int) (x0, y0, cell int) {
	side := min(w, h) - 2*boardPad
	if side <= 0 {
		return 0, 0, 0
	}
	cell = side / boardSize
	x0 = (w - cell*boardSize) / 2
	y0 = (h - cell*boardSize) / 2
	return x0, y0, cell
}

func cellAt(w, h, x, y int) int {
	x0, y0, cell := geometry(w, h)
	if cell == 0 || x < x0 || y < y0 || x >= x0+cell*boardSize || y >= y0+cell*boardSize {
		return -1
	}
	return (y-y0)/cell*boardSize + (x-x0)/cell
}
