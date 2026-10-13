package swt

// visibleRgnHolders holds every Control that, or whose descendant, may cache a visibleRgn. A control missing from it
// has no region in its whole subtree, so resetting that subtree is a no-op. Over-approximation is safe, under is not.
// Main thread only, like the rest of the toolkit.
var visibleRgnHolders = map[*Control]struct{}{}

// markVisibleRgn walks to the root without stopping at a marked node, so a chain broken by a skipped walk is repaired.
func markVisibleRgn(c *Control) {
	for c != nil {
		visibleRgnHolders[c] = struct{}{}
		c = upcastCompositeToControl(c.parent)
	}
}

// markVisibleRgnReparent re-marks the new parent chain of a holder moved by SetParent. The old chain stays over-marked.
func markVisibleRgnReparent(c *Control) {
	if _, ok := visibleRgnHolders[c]; ok {
		markVisibleRgn(upcastCompositeToControl(c.parent))
	}
}

func forgetVisibleRgn(c *Control) { delete(visibleRgnHolders, c) }

// clearVisibleRgn is called after c and its whole subtree were reset. An open GC context makes reset re-create regions.
func clearVisibleRgn(c *Control) {
	if c.display.contexts == nil {
		delete(visibleRgnHolders, c)
	}
}

// visibleRegionsIdle reports that resetting c's subtree changes nothing: no region cached there (Canvas and ProgressBar
// visiblePath are built after GetVisibleRegion, so they imply one) and no open GC, whose reset copies the region.
func visibleRegionsIdle(c *Control) bool {
	if c.display.contexts != nil {
		return false
	}
	_, held := visibleRgnHolders[c]
	return !held
}
