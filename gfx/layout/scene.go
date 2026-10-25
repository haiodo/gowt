package layout

import "github.com/haiodo/gowt/gfx"

// Apply moves the scene nodes of a laid out tree to their resolved positions inside tx, so a frame
// shows the whole layout or none of it. target maps a layout node to the scene node it stands for,
// or nil for a node that has none; a nil target reads Tag as a *gfx.Node. A layout node without a
// scene node passes its offset on, so a scene node is placed relative to its nearest ancestor that
// has one, which is how the scene's own transforms work. Sizes (RW, RH) are not part of a scene
// node: read them from the layout tree for clips and pictures.
func Apply(tx *gfx.Tx, root *Node, target func(*Node) *gfx.Node) {
	if target == nil {
		target = func(n *Node) *gfx.Node { g, _ := n.Tag.(*gfx.Node); return g }
	}
	apply(tx, root, target, 0, 0)
}

func apply(tx *gfx.Tx, n *Node, target func(*Node) *gfx.Node, ox, oy float64) {
	x, y := ox+n.RX, oy+n.RY
	if g := target(n); g != nil {
		tx.Move(g, x, y)
		x, y = 0, 0
	}
	for _, k := range n.Kids {
		apply(tx, k, target, x, y)
	}
}
