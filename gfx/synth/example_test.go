package synth_test

import (
	"fmt"

	"github.com/haiodo/gowt/gfx/synth"
)

// A document of boxes on a grid for benchmarks and demos; Link adds a connector between two boxes.
func ExampleNew() {
	doc := synth.New(100, 1)
	n := len(doc.Conns) // New already links neighbours
	doc.Link(doc.Boxes[0], doc.Boxes[5], false)
	doc.Scene.Commit()
	fmt.Println(len(doc.Boxes), len(doc.Conns)-n)
	// Output: 100 1
}
