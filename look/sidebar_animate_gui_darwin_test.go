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
	"github.com/haiodo/gowt/swt"
)

// The animated hide and show move the content through several intermediate widths and finish on
// the stored sidebar width.
func TestSidebarVisibleAnimated(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var hide, show []int
	var hideVisible, showVisible bool
	var hideWidth, showWidth int
	var unavailable bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("animated sidebar")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 200)
			var recording []int
			var changes int
			look.OnSidebarResize(w, func(visible bool, width int) {
				recording = append(recording, int(content.Unwrap().GetBounds().X))
			})
			look.OnSidebarChange(w, func(visible bool, width int) { changes++ })
			w.Show()
			app.After(150*time.Millisecond, func() {
				if start := content.Unwrap().GetBounds().X; start != 200 {
					unavailable = true
					w.Close()
					return
				}
				recording, changes = nil, 0
				wait := func(visible bool, x int, next func()) {
					finished := false
					var stop func()
					done := func(timedOut bool) {
						if finished {
							return
						}
						finished = true
						stop()
						if timedOut {
							t.Errorf("sidebar animation timed out: visible %v, content x %d, changes %d", look.SidebarVisible(w), content.Unwrap().GetBounds().X, changes)
						}
						next()
					}
					stop = app.Every(20*time.Millisecond, func() {
						if changes == 1 && look.SidebarVisible(w) == visible && int(content.Unwrap().GetBounds().X) == x {
							done(false)
						}
					})
					app.After(5*time.Second, func() { done(true) })
				}
				look.SetSidebarVisibleAnimated(w, false)
				wait(false, 0, func() {
					hide, hideVisible = append([]int(nil), recording...), look.SidebarVisible(w)
					hideWidth = look.SidebarWidth(w)
					if changes != 1 {
						t.Errorf("hide sent %d change callbacks, want 1", changes)
					}
					recording, changes = nil, 0
					look.SetSidebarVisibleAnimated(w, true)
					wait(true, 200, func() {
						show, showVisible = append([]int(nil), recording...), look.SidebarVisible(w)
						showWidth = look.SidebarWidth(w)
						if changes != 1 {
							t.Errorf("show sent %d change callbacks, want 1", changes)
						}
						w.Close()
					})
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if unavailable {
		t.Skip("full-height sidebar requires macOS 26")
	}
	if hideVisible {
		t.Error("the sidebar stayed visible after the hide animation")
	}
	if hideWidth != 200 || showWidth != 200 {
		t.Errorf("stored width after hide %d and show %d, want 200", hideWidth, showWidth)
	}
	if show == nil || show[len(show)-1] != 200 {
		t.Errorf("shown: visible %v, last content x %v", showVisible, tail(show))
	}
	if !showVisible {
		t.Error("the sidebar stayed hidden after the show animation")
	}
	checkAnimatedSamples(t, "hide", hide, 200, 0, false)
	checkAnimatedSamples(t, "show", show, 0, 200, true)
}

func tail(v []int) int {
	if len(v) == 0 {
		return -1
	}
	return v[len(v)-1]
}

func checkAnimatedSamples(t *testing.T, name string, samples []int, from, to int, rising bool) {
	t.Helper()
	if len(samples) == 0 || samples[0] != from || samples[len(samples)-1] != to {
		t.Fatalf("%s samples %v, want %d ... %d", name, samples, from, to)
	}
	intermediate, moves := 0, 0
	for i, x := range samples {
		if x != from && x != to {
			intermediate++
		}
		if i > 0 && x != samples[i-1] && (x > samples[i-1]) == rising {
			moves++
		}
	}
	if intermediate < 2 || moves < 2 {
		t.Fatalf("%s samples %v: %d intermediate, %d steps; want at least 2 of each", name, samples, intermediate, moves)
	}
}

// sidebarSash is the vertical sash on the sidebar's right edge.
func sidebarSash(w *gowt.Window) *swt.Sash {
	for _, c := range w.Unwrap().GetChildren() {
		if s, ok := c.Impl().(*swt.Sash); ok && s.GetStyle()&swt.VERTICAL != 0 {
			return s
		}
	}
	return nil
}

// A drag sends OnSidebarResize for every step and OnSidebarChange once, on the mouse up.
func TestSidebarResizeCallback(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var resizes, changes []string
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar drag")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebar(w, sidebar, content, 200)
			look.SetToolbar(w, look.ToolbarItem{Title: "Sidebar", Symbol: "sidebar.left", Sidebar: true})
			w.SetSize(900, 600)
			look.OnSidebarResize(w, func(visible bool, width int) {
				resizes = append(resizes, fmt.Sprint(visible, " ", width))
			})
			look.OnSidebarChange(w, func(visible bool, width int) {
				changes = append(changes, fmt.Sprint(visible, " ", width))
			})
			w.Show()
			app.After(200*time.Millisecond, func() {
				sash := sidebarSash(w)
				if sash == nil {
					t.Error("no sidebar sash")
					w.Close()
					return
				}
				send := func(typ int32, x int32) {
					e := swt.NewEvent()
					e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
					sash.NotifyListeners(typ, e)
				}
				x0 := sash.GetBounds().X
				send(swt.MouseDown, 0)
				send(swt.Selection, x0+20)
				send(swt.Selection, x0+40)
				if len(changes) != 0 {
					t.Errorf("changes during the drag: %v", changes)
				}
				send(swt.MouseUp, 0)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "[true 220 true 240]"; fmt.Sprint(resizes) != want {
		t.Errorf("resizes %v, want %s", resizes, want)
	}
	if want := "[true 240]"; fmt.Sprint(changes) != want {
		t.Errorf("changes %v, want %s", changes, want)
	}
}

// trafficLights is the x of the close, miniaturize and zoom buttons in the window.
func trafficLights(w *gowt.Window) [3]float64 {
	win := cocoa.Msg(w.Unwrap().View.Id, "window")
	var out [3]float64
	for i := range out {
		b := cocoa.Msg(win, "standardWindowButton:", int64(i))
		if b == 0 {
			out[i] = -1
			continue
		}
		out[i] = cocoa.NewNSViewOverload1(b).Frame().X
	}
	return out
}

func sameLights(a, b [3]float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 0.01 {
			return false
		}
	}
	return true
}

// A toolbar rebuilt during the hide animation does not move the traffic lights.
func TestSidebarAnimationKeepsTrafficLights(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("traffic lights")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 220)
			toggle := look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true,
				SidebarLead: true, Plain: true, Small: true}
			more := look.ToolbarItem{Title: "More", Symbol: "ellipsis", Sidebar: true, Plain: true, Small: true}
			look.SetToolbar(w, toggle, more)
			w.Show()
			app.After(300*time.Millisecond, func() {
				base := trafficLights(w)
				samples := [][3]float64{base}
				stop := app.Every(16*time.Millisecond, func() {
					samples = append(samples, trafficLights(w))
				})
				look.SetSidebarVisibleAnimated(w, false)
				app.After(60*time.Millisecond, func() {
					look.SetToolbar(w, toggle)
				})
				app.After(120*time.Millisecond, func() {
					look.SetToolbar(w, toggle, more)
				})
				app.After(350*time.Millisecond, func() {
					stop()
					samples = append(samples, trafficLights(w))
					for i, s := range samples {
						if !sameLights(base, s) {
							t.Errorf("traffic lights moved from %v to %v at sample %d", base, s, i)
							break
						}
					}
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The full-height sash sits on the card edge eight points inside the content column, and a drag keeps
// its inverse width math.
func TestSidebarSashAtCardEdge(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar sash")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetContentFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 200)
			w.SetSize(900, 600)
			w.Show()
			app.After(250*time.Millisecond, func() {
				sash := sidebarSash(w)
				if sash == nil {
					t.Error("no sidebar sash")
					w.Close()
					return
				}
				check := func() {
					b := sash.GetBounds()
					center := float64(b.X + b.Width/2)
					card := float64(content.Unwrap().GetBounds().X + 8)
					if math.Abs(center-card) > 3 {
						t.Errorf("sash centre %.1f, card edge %.1f", center, card)
					}
				}
				check()
				x0 := sash.GetBounds().X
				e := swt.NewEvent()
				e.X, e.Y, e.Button = x0+20, sash.GetBounds().Y, 1
				sash.NotifyListeners(swt.Selection, e)
				if got := look.SidebarWidth(w); got != 220 {
					t.Errorf("drag gave width %d, want 220", got)
				}
				check()
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A drag lays the toolbar items out in the same frame: More stays inside the sidebar edge.
func TestSidebarToolbarInsideDrag(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar toolbar drag")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 300)
			ok := look.SetToolbar(w,
				look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true,
					SidebarLead: true, Plain: true, Small: true, ID: "toggle", OnClick: func() {}},
				look.ToolbarItem{Title: "More", Symbol: "ellipsis", Sidebar: true,
					Plain: true, Small: true, ID: "more", OnClick: func() {}})
			if !ok {
				t.Error("the native toolbar was not installed")
			}
			w.SetSize(900, 600)
			w.Show()
			app.After(250*time.Millisecond, func() {
				sash := sidebarSash(w)
				if sash == nil {
					t.Error("no sidebar sash")
					w.Close()
					return
				}
				x0 := sash.GetBounds().X
				send := func(x int32) {
					e := swt.NewEvent()
					e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
					sash.NotifyListeners(swt.Selection, e)
				}
				e := swt.NewEvent()
				e.X, e.Y, e.Button = x0, sash.GetBounds().Y, 1
				sash.NotifyListeners(swt.MouseDown, e)
				for _, width := range []int{260, 220} {
					send(x0 + int32(width-300))
					if got := look.SidebarWidth(w); got != width {
						t.Errorf("drag gave width %d, want %d", got, width)
					}
					right := 0
					buttons := cocoa.ToolbarButtonsOf(w.Unwrap().View).List()
					for _, b := range buttons {
						if b.ID == "more" {
							right = b.Bounds[0] + b.Bounds[2]
						}
					}
					if right == 0 || right > width+2 {
						t.Errorf("width %d: More right %d, want it at or before the edge", width, right)
					}
				}
				e = swt.NewEvent()
				e.X, e.Y, e.Button = x0+int32(220-300), sash.GetBounds().Y, 1
				sash.NotifyListeners(swt.MouseUp, e)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A custom minimum stops a drag, SetSidebarWidth and the window minimum at 200 points.
func TestSidebarCustomMinWidth(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar minimum")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 300)
			look.SetSidebarMinWidth(w, 200)
			w.SetSize(900, 600)
			w.Show()
			app.After(250*time.Millisecond, func() {
				sash := sidebarSash(w)
				if sash == nil {
					t.Error("no sidebar sash")
					w.Close()
					return
				}
				x0 := sash.GetBounds().X
				for _, typ := range []int32{swt.MouseDown, swt.Selection, swt.MouseUp} {
					e := swt.NewEvent()
					e.X, e.Y, e.Button = x0-100, sash.GetBounds().Y, 1
					sash.NotifyListeners(typ, e)
				}
				if got := look.SidebarWidth(w); got != 200 {
					t.Errorf("drag gave width %d, want the 200-point minimum", got)
				}
				look.SetSidebarWidth(w, 50)
				if got := look.SidebarWidth(w); got != 200 {
					t.Errorf("SetSidebarWidth gave %d, want the 200-point minimum", got)
				}
				if got := w.Unwrap().GetMinimumSize().X; got < 400 {
					t.Errorf("window minimum width %d, want at least the sidebar and 200 points of content", got)
				}
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A panes-overlay window keeps the width when the divider is pressed without moving: there the sash
// is not shifted for the card edge.
func TestSidebarOverlayDragAtRest(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var got int
	var unavailable bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("overlay drag")
			var sidebar, content *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")))
			look.SetSidebarFullHeight(w, true)
			look.SetPanesOverlay(w, true)
			look.SetContentFullHeight(w, true)
			look.SetSidebar(w, sidebar, content, 200)
			w.SetSize(900, 600)
			w.Show()
			app.After(250*time.Millisecond, func() {
				if x := content.Unwrap().GetBounds().X; x != 0 {
					unavailable = true
					w.Close()
					return
				}
				sash := sidebarSash(w)
				if sash == nil {
					t.Error("no sidebar sash")
					w.Close()
					return
				}
				send := func(typ int32) {
					e := swt.NewEvent()
					e.X, e.Y, e.Button = sash.GetBounds().X, sash.GetBounds().Y, 1
					sash.NotifyListeners(typ, e)
				}
				send(swt.MouseDown)
				send(swt.Selection)
				got = look.SidebarWidth(w)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if unavailable {
		t.Skip("panes overlay requires macOS 26")
	}
	if got != 200 {
		t.Errorf("a still drag gave width %d, want 200", got)
	}
}

// SetSidebarMinWidth given before SetSidebar holds when a SetInspector layout came first.
func TestSidebarMinWidthBeforeSidebar(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var got int
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar minimum order")
			var sidebar, content, insp *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Label("side")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("body")),
				gowt.VBox(gowt.Ref(&insp), gowt.Label("insp")))
			look.SetInspector(w, content, insp, 200)
			look.SetSidebarMinWidth(w, 180)
			look.SetSidebar(w, sidebar, content, 300)
			w.SetSize(900, 600)
			w.Show()
			app.After(250*time.Millisecond, func() {
				look.SetSidebarWidth(w, 50)
				got = look.SidebarWidth(w)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 180 {
		t.Errorf("SetSidebarWidth gave %d, want the 180-point minimum", got)
	}
}
