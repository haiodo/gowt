// Command gfxeditor is the core of a mind-map editor on gfx: auto-layout of a topic tree, hit-testing,
// a selection outline and a remote cursor on the overlay, animations, and dragging a topic.
package main

import (
	"log"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/layout"
	"github.com/haiodo/gowt/gfx/view"
)

type topic struct {
	title string
	kids  []*topic
	node  *gfx.Node
	box   gfx.Rect // the topic's own rectangle, in node coordinates
}

var font = gfx.Font{Size: 14}

// build adds the scene nodes of t and its children and returns their layout: the topic on the left,
// its children in a column on the right.
func build(tx *gfx.Tx, t *topic, nextID *int) *layout.Node {
	w := gfx.DefaultMeasurer.MeasureText(t.title, font).Width + 24
	t.box = gfx.R(0, 0, w, 32)
	rec := gfx.NewRecorder()
	rec.FillRRect(t.box, 6, gfx.RGB(242, 242, 247))
	rec.StrokeRRect(t.box.Expand(-0.5), 6, gfx.StrokeStyle{Width: 1}, gfx.RGB(153, 153, 179))
	rec.Text(t.title, 12, 21, font, gfx.Black)
	*nextID++
	t.node = gfx.NewNode(*nextID)
	tx.Add(nil, t.node)
	tx.SetPicture(t.node, rec.Finish())

	label := &layout.Node{W: layout.Fixed(w), H: layout.Fixed(32), Tag: t.node}
	if len(t.kids) == 0 {
		return label
	}
	col := &layout.Node{Dir: layout.Column, Gap: 12}
	for _, k := range t.kids {
		col.Kids = append(col.Kids, build(tx, k, nextID))
	}
	return &layout.Node{Gap: 40, Align: layout.Center, Pad: layout.All(8), Kids: []*layout.Node{label, col}}
}

func find(t *topic, n *gfx.Node) *topic {
	if t.node == n {
		return t
	}
	for _, k := range t.kids {
		if f := find(k, n); f != nil {
			return f
		}
	}
	return nil
}

func main() {
	root := &topic{title: "Plan", kids: []*topic{
		{title: "Design"},
		{title: "Build", kids: []*topic{{title: "Backend"}, {title: "UI"}}},
		{title: "Ship"},
	}}
	scene := gfx.NewScene()
	id := 0
	scene.Update(func(tx *gfx.Tx) {
		tree := build(tx, root, &id)
		tree.Layout()
		layout.Apply(tx, tree, nil)
	})

	err := g.Run(func(app *g.App) {
		w := app.Window("Mind map", view.New(scene, func(v *view.View) {
			comp := v.Compositor()
			comp.SetCamera(gfx.Camera{Origin: gfx.Point{X: -40, Y: -40}, Zoom: 1.5})

			cursor := gfx.NewRecorder()
			cursor.FillPath(gfx.PolygonPath(gfx.Point{}, gfx.Point{Y: 16}, gfx.Point{X: 4, Y: 12}, gfx.Point{X: 10, Y: 12}), gfx.RGB(230, 51, 51))
			remote := comp.NewSprite(cursor.Finish(), gfx.SpacePinned, gfx.Point{})

			var outline *gfx.Sprite
			var base gfx.Point // where the outline stood when the drag began
			v.OnPointer(func(p *view.Pointer) {
				if p.Kind == view.PointerMove && p.Drag && outline != nil {
					outline.SetPos(gfx.Point{X: base.X + p.Scene.X - p.Origin.X, Y: base.Y + p.Scene.Y - p.Origin.Y})
				}
				if p.Kind != view.PointerDown {
					return
				}
				t := find(root, p.Node)
				if t == nil {
					if outline != nil {
						outline.FadeTo(0, 120*time.Millisecond, gfx.EaseOut, nil)
					}
					return
				}
				rec := gfx.NewRecorder()
				rec.StrokeRRect(t.box.Expand(2), 8, gfx.StrokeStyle{Width: 2}, gfx.RGB(26, 128, 255))
				if outline == nil {
					outline = comp.NewSprite(rec.Finish(), gfx.SpaceScene, gfx.Point{})
				} else {
					outline.SetPicture(rec.Finish())
				}
				base = t.node.Bounds().Min
				outline.SetPos(base)
				outline.SetOpacity(0)
				outline.FadeTo(1, 120*time.Millisecond, gfx.EaseOut, nil)

				// the remote cursor flies to the topic along a curve
				to := t.node.Bounds().Center()
				path := &gfx.Path{}
				path.MoveTo(remote.Pos().X, remote.Pos().Y)
				path.CubicTo(to.X-120, to.Y-80, to.X-40, to.Y+80, to.X, to.Y)
				remote.FlyAlong(path, 600*time.Millisecond, gfx.EaseInOut, false, nil)

				v.Drag(p, t.node)
			})
		}, g.Grow()))
		w.SetSize(800, 600)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
