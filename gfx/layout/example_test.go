package layout_test

import (
	"fmt"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/layout"
)

// A toolbar of chips that wraps, with a badge pinned to its corner.
func Example() {
	chip := func(w float64) *layout.Node {
		return &layout.Node{W: layout.Fixed(w), H: layout.Fixed(20)}
	}
	badge := &layout.Node{
		Absolute: true, X: 90, Y: -4,
		W: layout.Fixed(8), H: layout.Fixed(8),
	}
	bar := &layout.Node{
		Dir: layout.Row, Wrap: true, Gap: 6, LineGap: 4, Pad: [4]float64{4, 4, 4, 4},
		W:    layout.Fixed(100),
		Kids: []*layout.Node{chip(40), chip(40), chip(30), badge},
	}
	bar.Layout()
	for i, k := range bar.Kids {
		fmt.Printf("%d: x=%v y=%v w=%v h=%v\n", i, k.RX, k.RY, k.RW, k.RH)
	}
	fmt.Printf("bar: %v x %v\n", bar.RW, bar.RH)
	// Output:
	// 0: x=4 y=4 w=40 h=20
	// 1: x=50 y=4 w=40 h=20
	// 2: x=4 y=28 w=30 h=20
	// 3: x=90 y=-4 w=8 h=8
	// bar: 100 x 52
}

// Text measures itself: LeafW is its width on one line, LeafH its height at a given width.
func ExampleNode_text() {
	const charW, lineH = 7.0, 16.0
	content := "a rather long caption"
	words := func(width float64) float64 { // greedy wrap by words
		lines, cur := 1.0, 0.0
		for _, w := range []string{"a", "rather", "long", "caption"} {
			n := float64(len(w)) * charW
			if cur > 0 && cur+charW+n > width {
				lines, cur = lines+1, 0
			}
			if cur > 0 {
				cur += charW
			}
			cur += n
		}
		return lines * lineH
	}
	text := &layout.Node{
		W:     layout.Fill,
		LeafW: func() float64 { return float64(len(content)) * charW },
		LeafH: words,
	}
	card := &layout.Node{
		Dir: layout.Column, Pad: [4]float64{8, 8, 8, 8},
		W:    layout.Fixed(96),
		Kids: []*layout.Node{text},
	}
	card.Layout()
	fmt.Printf("text %v x %v, card %v x %v\n", text.RW, text.RH, card.RW, card.RH)
	// Output: text 80 x 48, card 96 x 64
}

// Apply moves scene nodes to the resolved positions in one transaction. The layout tree here
// mirrors the app's tree; Tag links each layout node to its scene node.
func ExampleApply() {
	scene := gfx.NewScene()
	left, right := gfx.NewNode(1), gfx.NewNode(2)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, left)
		tx.Add(nil, right)
	})
	scene.Commit()

	box := func(g *gfx.Node) *layout.Node {
		return &layout.Node{
			Tag: g,
			W:   layout.Fixed(50),
			H:   layout.Fixed(30),
		}
	}
	root := &layout.Node{Gap: 10, Kids: []*layout.Node{box(left), box(right)}}
	root.Layout()

	scene.Update(func(tx *gfx.Tx) {
		layout.Apply(tx, root, func(n *layout.Node) *gfx.Node {
			g, _ := n.Tag.(*gfx.Node)
			return g
		})
	})
	scene.Commit()
	t := right.Transform()
	fmt.Println(t[4], t[5])
	// Output: 60 0
}

// Layout nodes without a scene node pass their offset on: the scene is flat here, the layout nested.
// A nil target reads Tag as the *gfx.Node.
func ExampleApply_nested() {
	scene := gfx.NewScene()
	a, b := gfx.NewNode(1), gfx.NewNode(2)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, a)
		tx.Add(nil, b)
	})
	scene.Commit()

	leaf := func(g *gfx.Node) *layout.Node {
		return &layout.Node{Tag: g, W: layout.Fixed(40), H: layout.Fixed(20)}
	}
	column := &layout.Node{Dir: layout.Column, Gap: 6, Kids: []*layout.Node{leaf(a), leaf(b)}}
	root := &layout.Node{Pad: layout.All(10), Kids: []*layout.Node{{W: layout.Fixed(30), H: layout.Fixed(30)}, column}, Gap: 5}
	root.Layout()

	scene.Update(func(tx *gfx.Tx) { layout.Apply(tx, root, nil) })
	scene.Commit()
	fmt.Println(a.Transform()[4], a.Transform()[5], b.Transform()[4], b.Transform()[5])
	// Output: 45 10 45 36
}
