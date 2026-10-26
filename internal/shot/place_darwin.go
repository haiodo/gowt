package shot

import (
	"fmt"
	"os"
	"structs"

	"github.com/ebitengine/purego/objc"
)

type nsPoint struct {
	_    structs.HostLayout
	X, Y float64
}

// PlaceOnRetina moves the window to the first Retina screen unless it is on one, so a SNAP_SCREEN capture
// is not a 1x display stretched, and reports whether it moved it. The window needs a few loop turns there.
func PlaceOnRetina(view int64) bool {
	if os.Getenv("GOWT_SNAP_SCREEN") != "1" {
		return false
	}
	sel := objc.RegisterName
	win := objc.ID(view).Send(sel("window"))
	if win == 0 {
		return false
	}
	scale := func(screen objc.ID) float64 { return objc.Send[float64](screen, sel("backingScaleFactor")) }
	if cur := win.Send(sel("screen")); cur != 0 && scale(cur) >= 2 {
		return false
	}
	screens := objc.ID(objc.GetClass("NSScreen")).Send(sel("screens"))
	for i, n := uint64(0), objc.Send[uint64](screens, sel("count")); i < n; i++ {
		s := screens.Send(sel("objectAtIndex:"), i)
		if scale(s) < 2 {
			continue
		}
		vf := objc.Send[nsRect](s, sel("visibleFrame"))
		win.Send(sel("setFrameTopLeftPoint:"), nsPoint{X: vf.X + 40, Y: vf.Y + vf.H - 40})
		fmt.Fprintln(os.Stderr, "gowt: snap moved the window to a Retina screen")
		return true
	}
	return false
}
