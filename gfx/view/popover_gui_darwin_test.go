package view

import (
	"os"
	"runtime"
	"structs"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

var mainq = make(chan func())

func init() { runtime.LockOSThread() }

// AppKit wants the main thread: the tests run on their own goroutines and hand work to this one.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

// macKeys are the virtual key codes of the characters postKey types.
var macKeys = map[uint16]uint16{' ': 49, 'x': 7, 'y': 16, '\r': 36}

// postKey queues a press and a release of char as typed keys for the key window. A test process that
// is not active has no key window: then they go straight to the window of the focus control.
func postKey(d *swt.Display, char uint16) {
	app := cocoa.NSApplicationSharedApplication()
	win := app.KeyWindow()
	if win == nil {
		c := d.GetFocusControl()
		if c == nil {
			return
		}
		win = c.View.Window()
	}
	sel := objc.RegisterName
	s := cocoa.NSStringStringWith(string(rune(char)))
	for _, typ := range []uint64{10, 11} { // NSEventTypeKeyDown, NSEventTypeKeyUp
		ev := objc.ID(objc.GetClass("NSEvent")).Send(sel("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"),
			typ, nsPoint{}, uint64(0), 0.0, win.WindowNumber(), objc.ID(0), objc.ID(s.Id), objc.ID(s.Id), false, macKeys[char])
		if app.KeyWindow() != nil {
			app.PostEvent(cocoa.NewNSEventOverload1(int64(ev)), false)
		} else {
			win.SendEvent(cocoa.NewNSEventOverload1(int64(ev)))
		}
	}
}

type nsPoint struct {
	_    structs.HostLayout
	X, Y float64
}

// TestPopoverAtPointGUI shows a popover above a point in the lower half of a view: it must sit next
// to the point, not at the top edge of the view, and carry its buttons.
func TestPopoverAtPointGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			var v *View
			w := app.Window("popover", New(gfx.NewScene(), func(x *View) { v = x }, g.Grow()))
			w.SetSize(500, 600)
			w.Show()
			var a, b *g.ButtonView
			p := w.Popover(g.Gap(2), g.Padding(4))
			p.Add(g.HBox(g.Button("One", g.Ref(&a)), g.Button("Two", g.Ref(&b))))
			const px, py = 250, 400
			app.After(300*time.Millisecond, func() { p.ShowAtPoint(v.Canvas(), px, py, g.PopoverAbove) })
			app.After(1500*time.Millisecond, func() {
				defer w.Close()
				if !p.Shown() {
					t.Error("the popover is not shown")
					return
				}
				sh := p.Unwrap()
				win := sh.View.Window()
				bounds := sh.View.Bounds()
				r := sh.View.ConvertRect_toView_(bounds, nil)
				top := win.ConvertBaseToScreen(cocoa.NSPoint{X: r.X, Y: r.Y + r.Height})
				topY := sh.GetDisplay().GetPrimaryFrame().Height - top.Y
				anchor := v.Canvas().Unwrap().ToDisplay(px, py)
				t.Logf("anchor %d,%d; popover content %.0f,%.0f %.0fx%.0f", anchor.X, anchor.Y, top.X, topY, bounds.Width, bounds.Height)
				if gap := float64(anchor.Y) - (topY + bounds.Height); gap < -2 || gap > 40 {
					t.Errorf("the popover bottom is %.0f points from the anchor, want 0..40", gap)
				}
				if dx := top.X + bounds.Width/2 - float64(anchor.X); dx < -60 || dx > 60 {
					t.Errorf("the popover is %.0f points off the anchor horizontally", dx)
				}
				for _, bv := range []*g.ButtonView{a, b} {
					c := bv.Unwrap()
					if c.View.Window() == nil || c.View.Window().Id != win.Id {
						t.Error("a button is not in the popover window")
					} else if sz := c.GetSize(); sz.X <= 0 || sz.Y <= 0 {
						t.Errorf("a button has the size %v", sz)
					}
				}
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
