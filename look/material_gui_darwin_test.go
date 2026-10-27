package look_test

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
)

// materialProblem wants the inspector's edge-to-edge glass in the pane on macOS 26, square and clipped
// to the pane, and the under-window-background material with behind-window blending (21, 0) before.
func materialProblem(_ *gowt.Window, pane *gowt.Panel, _ gowt.RGB) string {
	v := pane.AsComposite().View
	name, material, blending := cocoa.BackdropOf(v)
	if a := pane.AsComposite().GetBackground().GetAlpha(); a != 0 {
		return fmt.Sprintf("pane background alpha %d, want 0", a)
	}
	if !cocoa.GlassAvailable() {
		if name != "NSVisualEffectView" || material != 21 || blending != 0 {
			return fmt.Sprintf("pane backdrop %s material %d blending %d, want NSVisualEffectView 21 0", name, material, blending)
		}
		return ""
	}
	if name != "GowtGlassPanel" {
		return fmt.Sprintf("pane backdrop %q, want the panel glass GowtGlassPanel", name)
	}
	if radius, masks, ok := cocoa.PanelGlassClip(v); !ok || radius != 0 || !masks {
		return fmt.Sprintf("pane glass radius %.1f, masksToBounds %v, want 0 and true", radius, masks)
	}
	return ""
}
