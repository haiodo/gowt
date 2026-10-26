package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// TestPopoverGUI checks focus, placement and the ways to close a popover shown over a window.
func TestPopoverGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var failed []string
	fail := func(s string) { failed = append(failed, s) }
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			var btn *ButtonView
			w := app.Window("popover", Button("Anchor", Ref(&btn)))
			var field *TextView
			p := w.Popover(Gap(4))
			p.Add(Text(Focused(), Ref(&field)), Label("below"))
			closed := 0
			p.OnClose(func() { closed++ })
			w.Show()
			wait := func(d time.Duration, f func()) { app.After(d, f) }

			var step func(n int)
			step = func(n int) {
				switch n {
				case 0:
					p.ShowAt(btn, PopoverBelow)
				case 1:
					if !p.Shown() {
						fail("not shown after ShowAt")
						break
					}
					if p.native.onScreen(p) && !p.native.focused(p, &field.t.Control) {
						fail("the search field has no focus")
					}
					a := btn.ctl.ToDisplay(0, 0)
					b := p.native.bounds(p)
					t.Logf("popover %d,%d %dx%d", b.X, b.Y, b.Width, b.Height)
					if b.Y < a.Y+btn.ctl.GetSize().Y {
						fail("the popover overlaps its anchor")
					}
					area := btn.ctl.GetMonitor().GetClientArea()
					if b.X < area.X || b.Y < area.Y || b.X+b.Width > area.X+area.Width || b.Y+b.Height > area.Y+area.Height {
						fail("the popover leaves the screen")
					}
					escapes := 0
					p.OnEscape(func() bool { escapes++; return escapes == 1 })
					f := p.filters
					f.HandleEvent(&swt.Event{Type: swt.KeyDown, KeyCode: int32(swt.ESC), Widget: &field.t.Widget, Doit: true})
					if !p.Shown() {
						fail("OnEscape returning true did not keep the popover open")
					}
					f.HandleEvent(&swt.Event{Type: swt.KeyDown, KeyCode: int32(swt.ESC), Widget: &field.t.Widget, Doit: true})
					if p.Shown() || closed != 1 {
						fail("Esc did not close the popover once")
					}
					p.ShowAtPoint(btn, 5, 5, PopoverAbove)
				case 2:
					if !p.Shown() {
						fail("not shown again at a point")
						break
					}
					p.filters.HandleEvent(&swt.Event{Type: swt.MouseDown, Widget: &btn.b.Widget, Doit: true})
					if p.Shown() || closed != 2 {
						fail("a press outside did not close the popover")
					}
					p.ShowAt(btn, PopoverRight)
				case 3:
					p.Close()
					p.Close()
					if p.Shown() || closed != 3 {
						fail("Close did not close the popover once")
					}
					p.ShowAt(btn, PopoverBelow)
				case 4:
					w.shell.ForceActive()
				case 5:
					if p.Shown() || closed != 4 {
						fail("activating the window did not close the popover")
					}
					w.Close()
					return
				}
				wait(300*time.Millisecond, func() { step(n + 1) })
			}
			step(0)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range failed {
		t.Error(s)
	}
}
