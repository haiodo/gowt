package layout

import "github.com/haiodo/gowt/gfx"

// Apply moves the scene nodes of a laid out tree to their resolved positions inside tx, so a frame
// shows the whole layout or none of it. target maps a layout node (usually through its Tag) to the
// scene node it stands for, or nil for a node that has none; the children of such a node are still
// visited. Positions are relative to the parent, like the scene's own transforms. Sizes (RW, RH) are
// not part of a scene node: read them from the layout tree for clips and pictures.
func Apply(tx *gfx.Tx, root *Node, target func(*Node) *gfx.Node) {
	if g := target(root); g != nil {
		tx.Move(g, root.RX, root.RY)
	}
	for _, k := range root.Kids {
		Apply(tx, k, target)
	}
}
