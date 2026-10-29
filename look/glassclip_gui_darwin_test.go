package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// A SetGlass capsule in the content of SetSidebar clips its glass to the corner radius, with SetGlass
// before and after SetSidebar: on macOS 26.5 the glass view otherwise fills a square behind the capsule.
func TestGlassCapsuleClipInSidebarContent(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("needs macOS 26")
	}
	var reduce bool
	onMain(func() { reduce = cocoa.ReduceTransparency() })
	if reduce {
		t.Skip("Reduce transparency: the panel backdrop is the NSVisualEffectView sibling")
	}
	for _, glassFirst := range []bool{true, false} {
		var radius float64
		var masks, ok bool
		var err error
		onMain(func() {
			err = gowt.Run(func(app *gowt.App) {
				w := app.Window("glass clip")
				w.SetSize(700, 400)
				var sidebar, content, capsule *gowt.Panel
				w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.Label("sidebar")),
					gowt.VBox(gowt.Ref(&content), gowt.Padding(7),
						gowt.HBox(gowt.Ref(&capsule), gowt.GrowX(), gowt.Size(0, 40), gowt.Label("capsule"))))
				glass := func() {
					look.SetGlassRadius(capsule, 18)
					look.SetGlass(capsule, true)
				}
				if glassFirst {
					glass()
				}
				look.SetSidebar(w, sidebar, content, 200)
				if !glassFirst {
					glass()
				}
				w.Show()
				app.After(300*time.Millisecond, func() {
					radius, masks, ok = cocoa.PanelGlassClip(capsule.Unwrap().View)
					w.Close()
				})
			})
		})
		if err != nil {
			t.Fatal(err)
		}
		if !ok || radius != 18 || !masks {
			t.Errorf("glass first %v: glass layer cornerRadius %v masksToBounds %v (glass found %v), want 18 and true", glassFirst, radius, masks, ok)
		}
	}
}
