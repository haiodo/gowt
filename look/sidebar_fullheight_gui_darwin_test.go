package look_test

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

func TestSidebarFullHeightScrollUnderToolbar(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errors []error
	var runErr error
	var unavailable bool
	onMain(func() {
		runErr = gowt.Run(func(app *gowt.App) {
			app.OnError(func(err error) { errors = append(errors, err) })
			w := app.Window("full-height sidebar")
			var sidebar, content *gowt.Panel
			var table *gowt.TableView
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.Table(gowt.Ref(&table), gowt.Grow(),
				gowt.Init(func(table *gowt.TableView) {
					for i := 0; i < 40; i++ {
						table.Row(fmt.Sprintf("Row %02d", i))
					}
				}))), gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebarFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 240)
			look.SetToolbar(w, look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up"})
			w.SetSize(640, 420)
			w.Show()
			app.After(300*time.Millisecond, func() {
				defer w.Close()
				if !cocoa.GlassAvailable() {
					unavailable = true
					return
				}
				app.Flush()
				view := table.Unwrap().View
				scrollID := cocoa.Msg(view.Id, "enclosingScrollView")
				scroll := cocoa.NewNSViewOverload1(scrollID)
				clip := cocoa.NewNSViewOverload1(cocoa.Msg(scrollID, "contentView"))
				frame := scroll.ConvertRect_toView_(scroll.Bounds(), nil)
				windowContent := cocoa.NewNSViewOverload1(cocoa.Msg(cocoa.Msg(view.Id, "window"), "contentView"))
				if math.Abs(frame.Y+frame.Height-windowContent.Frame().Height) > 1 {
					t.Errorf("scroll top %.1f, window top %.1f", frame.Y+frame.Height, windowContent.Frame().Height)
				}
				rowZero := cocoa.NewNSTableViewOverload1(view.Id).RectOfRow(0)
				rowWindow := view.ConvertRect_toView_(rowZero, nil)
				var contentInsets cocoa.NSRect
				cocoa.OSObjc_msgSend_stretOverload9(&contentInsets, scrollID, cocoa.OSSel_registerName("contentInsets"))
				topInset := cocoa.ContentTopInset(view)
				t.Logf("initial clip=%+v document=%+v row0=%+v insets=%+v topInset=%d", clip.Bounds(), view.Frame(), rowWindow, contentInsets, topInset)
				toolbarBottom := windowContent.Frame().Height - float64(topInset)
				if rowWindow.Y+rowWindow.Height > toolbarBottom+1 {
					t.Errorf("first row top %.1f overlaps toolbar ending at %.1f", rowWindow.Y+rowWindow.Height, toolbarBottom)
				}
				if clip.Bounds().Y >= 0 {
					t.Errorf("first row has no toolbar inset: clip=%+v", clip.Bounds())
				}
				if table.Unwrap().GetClientArea().Height <= 0 {
					t.Error("table has no client height")
				}
				checkMaterial := func(want bool) {
					subs := cocoa.Msg(scrollID, "subviews")
					var material int64
					for i, n := int64(0), cocoa.Msg(subs, "count"); i < n; i++ {
						child := cocoa.Msg(subs, "objectAtIndex:", i)
						if cocoa.OSClass_getName(cocoa.OSObject_getClass(child)) == "GowtSidebarScrollMaterial" {
							material = child
						}
					}
					if !want {
						if material != 0 {
							t.Error("material retained after disabling")
						}
						return
					}
					if material == 0 {
						t.Error("scroll-under material missing")
						return
					}
					edge := cocoa.NewNSViewOverload1(material).Frame()
					mask := cocoa.Msg(cocoa.Msg(material, "layer"), "mask")
					var maskFrame cocoa.NSRect
					cocoa.OSObjc_msgSend_stretOverload9(&maskFrame, mask, cocoa.OSSel_registerName("frame"))
					if math.Abs(edge.Width-scroll.Bounds().Width) > 1 || edge.Width != maskFrame.Width || edge.Height != maskFrame.Height {
						t.Errorf("viewport=%+v edge=%+v mask=%+v", scroll.Bounds(), edge, maskFrame)
					}
					if cocoa.Msg(cocoa.Msg(mask, "animationKeys"), "count") != 0 {
						t.Error("gradient resize has animations")
					}
				}
				checkMaterial(true)
				view.ScrollPoint(cocoa.NSPoint{Y: 160})
				if clip.Bounds().Y <= 0 {
					t.Error("table did not scroll beneath toolbar")
				}
				look.SetSidebarWidth(w, 300)
				app.Flush()
				checkMaterial(true)
				closeX := func() float64 {
					win := cocoa.Msg(view.Id, "window")
					btn := cocoa.Msg(win, "standardWindowButton:", 0)
					bv := cocoa.NewNSViewOverload1(btn)
					return bv.ConvertRect_toView_(bv.Bounds(), nil).X
				}
				fullX := closeX()
				look.SetSidebarFullHeight(w, false)
				app.Flush()
				checkMaterial(false)
				if cocoa.Msg(scrollID, "automaticallyAdjustsContentInsets")&255 != 0 {
					t.Error("default inset policy not restored")
				}
				plainX := closeX()
				look.SetSidebarFullHeight(w, true)
				app.Flush()
				checkMaterial(true)
				if got := closeX() - plainX; got < 6 || got > 8 {
					t.Errorf("full-height traffic lights shifted %.1f from AppKit's layout, want the 7-point glass inset", got)
				}
				if got := closeX(); math.Abs(got-fullX) > 0.5 {
					t.Errorf("re-enabled traffic lights at %.1f, want the initial %.1f", got, fullX)
				}
				look.SetSidebarFullHeight(w, false)
				app.Flush()
				if got := closeX(); math.Abs(got-plainX) > 0.5 {
					t.Errorf("traffic lights after disabling full-height at %.1f, want AppKit's %.1f back", got, plainX)
				}
			})
		})
	})
	if unavailable {
		t.Skip("full-height sidebar requires macOS 26")
	}
	for _, err := range append(errors, runErr) {
		if err != nil {
			t.Error(err)
		}
	}
}
