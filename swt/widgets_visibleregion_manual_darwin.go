package swt

// visibleRgnCount counts Controls whose visibleRgn is non-zero (any Display). Main thread only, like the rest of the toolkit.
var visibleRgnCount int

// visibleRegionsIdle reports that resetting visible regions would change nothing: no Control caches a region (Canvas and
// ProgressBar visiblePath are only built after GetVisibleRegion, so they imply one) and no GC is open.
func visibleRegionsIdle(d *Display) bool {
	return visibleRgnCount == 0 && d.contexts == nil
}
