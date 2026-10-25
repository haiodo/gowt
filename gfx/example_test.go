package gfx_test

import (
	"fmt"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
)

// A scene holds nodes; a node draws a Picture recorded in its own coordinates. Changes go through
// Update and take effect at Commit, which a window does every frame.
func Example() {
	rec := gfx.NewRecorder()
	rec.FillRRect(gfx.R(0, 0, 120, 60), 8, gfx.RGB(30, 120, 220))
	rec.Text("Hello", 12, 36, gfx.Font{Size: 14}, gfx.White)

	card := gfx.NewNode(1)
	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, card)
		tx.SetPicture(card, rec.Finish())
		tx.Move(card, 100, 50)
	})
	scene.Commit() // without a window nobody else does

	fmt.Println(scene.HitTest(gfx.Point{X: 110, Y: 60}, 0).ID(), scene.HitTest(gfx.Point{X: 0, Y: 0}, 0) == nil)
	fmt.Println(card.Bounds().Min, scene.Node(1) == card)
	// Output:
	// 1 true
	// {100 50} true
}

// The Recorder has a helper for each common shape; Text takes the baseline origin.
func ExampleRecorder() {
	rec := gfx.NewRecorder()
	rec.FillRect(gfx.R(0, 0, 100, 40), gfx.RGB(240, 240, 245))
	rec.StrokeRRect(gfx.R(0, 0, 100, 40), 6, gfx.StrokeStyle{Width: 2}, gfx.RGB(26, 128, 255))
	rec.FillEllipse(gfx.R(80, 20, 30, 30), gfx.RGBA(230, 51, 51, 0.8))
	pic := rec.Finish()
	fmt.Println(pic.Bounds.W() > 100, pic.Bounds.H() > 40)
	// Output: true true
}

// A node can move, fade and be clipped without touching its picture.
func ExampleTx() {
	scene := gfx.NewScene()
	frame, item := gfx.NewNode(1), gfx.NewNode(2)
	rec := gfx.NewRecorder()
	rec.FillRect(gfx.R(0, 0, 50, 50), gfx.Black)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, frame)
		tx.Add(frame, item) // children move with their parent
		tx.SetPicture(item, rec.Finish())
		tx.ClipRRect(frame, gfx.R(0, 0, 30, 30), 4) // clips the frame and its subtree
		tx.Move(frame, 10, 10)
	})
	scene.Commit()
	fmt.Println(item.Bounds())
	// Output: {{10 10} {40 40}}
}

// RenderSubtree draws a node and its subtree into a Raster; offscreen.PNG does the same in one call.
func ExampleRenderSubtree() {
	rec := gfx.NewRecorder()
	rec.FillRRect(gfx.R(0, 0, 160, 80), 10, gfx.RGB(30, 120, 220))
	n := gfx.NewNode(1)
	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})

	r, err := offscreen.New(offscreen.Options{})
	if err != nil {
		fmt.Println(err)
		return
	}
	// the zero Rect is the bounds of the subtree; 2 is device pixels per scene unit
	raster, err := gfx.RenderSubtree(r, scene, n, gfx.Rect{}, 2, gfx.White)
	if err != nil {
		fmt.Println(err)
		return
	}
	png, _ := raster.PNG()
	fmt.Println(len(png) > 0)
}

// Animations run on the frame clock of the compositor; set a goal once from any goroutine.
func ExampleSprite_FlyAlong() {
	var comp *gfx.Compositor // from view.View.Compositor()
	rec := gfx.NewRecorder()
	rec.FillEllipse(gfx.R(0, 0, 10, 10), gfx.RGB(230, 51, 51))
	cursor := comp.NewSprite(rec.Finish(), gfx.SpacePinned, gfx.Point{})
	path := &gfx.Path{}
	path.MoveTo(0, 0)
	path.CubicTo(60, 0, 100, 80, 200, 120)
	cursor.FlyAlong(path, 600*time.Millisecond, gfx.EaseInOut, false, nil)
}
