//go:build darwin

package look_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// A bare single icon hovers through the tracking-area messages AppKit delivers: mouseEntered: sets
// the quiet fill, mouseExited: clears it, and the NSWindowDidResignKeyNotification observer clears
// it when the window resigns key, so an inactive window keeps no stale hover. The messages are sent
// to the button directly - moving the real pointer is off limits in tests - so the actual delivery
// by AppKit under a physical cursor is not covered here; the tracking area behind it is checked in
// TestToolbarBareIconItem (internal/cocoa): options entered/exited, active in the key window.
func TestToolbarHoverActiveWindow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []string
	fail := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("hover")
			look.SetToolbar(w, look.ToolbarItem{Title: "Zoom In", ID: "zoom-in", Symbol: "plus.magnifyingglass", OnClick: func() {}})
			w.Unwrap().SetLocation(60, 80)
			w.SetSize(700, 400)
			w.Show()
			app.After(600*time.Millisecond, func() {
				hovered := func() bool { return cocoa.ToolbarIconHovered(w.Unwrap().View, "zoom-in") }
				if !cocoa.ToolbarIconSendHover(w.Unwrap().View, "zoom-in", true) {
					fail("no bare icon button for zoom-in")
				} else {
					if !hovered() {
						fail("mouseEntered: left no hover")
					}
					cocoa.ToolbarIconSendHover(w.Unwrap().View, "zoom-in", false)
					if hovered() {
						fail("mouseExited: kept the hover")
					}
					cocoa.ToolbarIconSendHover(w.Unwrap().View, "zoom-in", true)
					cocoa.ToolbarPostResignKey()
					if hovered() {
						fail("the resign-key notification kept the hover")
					}
				}
				for _, e := range errs {
					t.Error(e)
				}
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
