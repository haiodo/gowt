package gfx

import "time"

// Group orders the layers a Host holds: tiles lie under the active layer, which lies under the overlay, which lies under the HUD.
type Group uint8

// The groups of layers a Host draws, from the bottom: static tiles, the active layer of dragged
// nodes, the overlay of sprites, the HUD in view coordinates.
const (
	GroupTiles Group = iota
	GroupActive
	GroupOverlay
	GroupHUD
)

// Layer is one retained picture a Host composites: a tile, a sprite. Within a group a layer made
// later lies above one made earlier. Coordinates are view points, y down.
type Layer interface {
	// SetImage replaces the contents. The Host keeps its own reference; the caller releases img after the call.
	SetImage(img Image)
	// Set places the untransformed frame, then applies m about the point anchor given in the frame's
	// own coordinates.
	Set(frame Rect, anchor Point, m Affine)
	SetOpacity(a float64)
	SetHidden(h bool)
	Remove()
}

// Host is the surface the compositor puts layers on: a layer-backed view. All calls come from one
// goroutine, the one that runs Frame.
type Host interface {
	NewLayer(g Group) Layer
	// Begin and Commit bracket the layer changes of one frame, so that they appear together.
	Begin()
	Commit()
}

// FrameInfo describes one display frame. Time is the frame clock, the time the frame will be shown,
// counted from an arbitrary start; only differences mean anything. Animations run on it.
type FrameInfo struct {
	Time     time.Duration
	Interval time.Duration // the display's refresh period, 0 if unknown
	Seq      uint64
}
