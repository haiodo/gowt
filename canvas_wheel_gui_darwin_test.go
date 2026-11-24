//go:build darwin

package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt/internal/cocoa"
)

func TestCanvasNativeWheel(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var create func(uintptr, uint32, uint32, int32, int32, int32) uintptr
	var release func(uintptr)
	var msg func(uintptr, uintptr, uintptr) uintptr
	purego.RegisterLibFunc(&create, purego.RTLD_DEFAULT, "CGEventCreateScrollWheelEvent2")
	purego.RegisterLibFunc(&release, purego.RTLD_DEFAULT, "CFRelease")
	purego.RegisterLibFunc(&msg, purego.RTLD_DEFAULT, "objc_msgSend")
	var calls []Wheel
	var before, after, consumedOrigin int
	consume := false
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			var c *CanvasView
			var sc *ScrollView
			w := app.Window("wheel")
			w.SetSize(240, 160)
			w.Add(Scroll(Grow(), Ref(&sc), Canvas(nil, Size(120, 80), Ref(&c), OnWheel(func(e Wheel) bool { calls = append(calls, e); return consume && !e.Horizontal })), Label("tall", Size(100, 1000))))
			w.Show()
			app.After(150*time.Millisecond, func() {
				_, before = sc.Origin()
				cg := create(0, 1, 2, -3, -2, 0)
				if cg == 0 {
					t.Error("CGEventCreateScrollWheelEvent2 returned nil")
					w.Close()
					return
				}
				defer release(cg)
				ev := msg(uintptr(cocoa.OSObjc_getClass("NSEvent")), uintptr(cocoa.OSSel_registerName("eventWithCGEvent:")), cg)
				msg(uintptr(c.Unwrap().View.Id), uintptr(cocoa.OSSel_registerName("scrollWheel:")), ev)
				app.After(150*time.Millisecond, func() {
					_, after = sc.Origin()
					consume = true
					cg := create(0, 1, 2, -3, -2, 0)
					if cg == 0 {
						t.Error("CGEventCreateScrollWheelEvent2 returned nil")
						w.Close()
						return
					}
					defer release(cg)
					ev := msg(uintptr(cocoa.OSObjc_getClass("NSEvent")), uintptr(cocoa.OSSel_registerName("eventWithCGEvent:")), cg)
					msg(uintptr(c.Unwrap().View.Id), uintptr(cocoa.OSSel_registerName("scrollWheel:")), ev)
					app.After(150*time.Millisecond, func() { _, consumedOrigin = sc.Origin(); w.Close() })
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 || calls[0].Horizontal || !calls[1].Horizontal || calls[2].Horizontal || !calls[3].Horizontal || calls[0].Delta != -3 || calls[1].Delta != -2 {
		t.Errorf("diagonal calls: %+v", calls)
	}
	if after <= before || consumedOrigin != after {
		t.Errorf("parent origin before=%d after=%d consumed=%d", before, after, consumedOrigin)
	}
}

func TestCanvasNativeWheelDispose(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var create func(uintptr, uint32, uint32, int32, int32, int32) uintptr
	var release func(uintptr)
	var msg func(uintptr, uintptr, uintptr) uintptr
	purego.RegisterLibFunc(&create, purego.RTLD_DEFAULT, "CGEventCreateScrollWheelEvent2")
	purego.RegisterLibFunc(&release, purego.RTLD_DEFAULT, "CFRelease")
	purego.RegisterLibFunc(&msg, purego.RTLD_DEFAULT, "objc_msgSend")
	for _, disposeHorizontal := range []bool{false, true} {
		var calls int
		var err error
		onMain(func() {
			err = Run(func(app *App) {
				var c *CanvasView
				w := app.Window("wheel dispose")
				w.Add(Canvas(nil, Size(120, 80), Ref(&c), OnWheel(func(e Wheel) bool {
					calls++
					if e.Horizontal == disposeHorizontal {
						c.Unwrap().Dispose()
					}
					return false
				})))
				w.Show()
				app.After(100*time.Millisecond, func() {
					cg := create(0, 1, 2, -3, -2, 0)
					if cg == 0 {
						t.Error("CGEventCreateScrollWheelEvent2 returned nil")
						w.Close()
						return
					}
					defer release(cg)
					ev := msg(uintptr(cocoa.OSObjc_getClass("NSEvent")), uintptr(cocoa.OSSel_registerName("eventWithCGEvent:")), cg)
					msg(uintptr(c.Unwrap().View.Id), uintptr(cocoa.OSSel_registerName("scrollWheel:")), ev)
					w.Close()
				})
			})
		})
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if disposeHorizontal {
			want = 2
		}
		if calls != want {
			t.Errorf("horizontal=%v: calls=%d, want %d", disposeHorizontal, calls, want)
		}
	}
}
