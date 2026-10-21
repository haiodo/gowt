package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// GlassButton's bezel was put back to the push bezel by SWT's layout, and AppKit draws the glass
// only for a button that left SWTButton and its cell. Circle gives it the circle shape in either order.
func TestGlassButtonKeepsBezel(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var plain, a, b *gowt.ButtonView
	var clicks, plainClicks int
	var bezel, shape [3]int
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("glass button")
			w.SetSize(300, 200)
			w.Add(gowt.HBox(
				gowt.Button("Plain", gowt.Ref(&plain), gowt.OnClick(func() { plainClicks++ })),
				gowt.Button("", look.GlassButton(), gowt.Circle(), gowt.Size(40, 40), gowt.Ref(&a), gowt.OnClick(func() { clicks++ })),
				gowt.Button("Glass", gowt.Circle(), look.GlassButton(), gowt.Size(40, 40), gowt.Ref(&b))))
			w.Show()
			app.After(150*time.Millisecond, func() {
				for i, btn := range []*gowt.ButtonView{plain, a, b} {
					bezel[i] = cocoa.ButtonBezelStyle(btn.Unwrap().View)
					shape[i] = cocoa.ButtonBorderShape(btn.Unwrap().View)
				}
				cocoa.Msg(int64(plain.Unwrap().View.Id), "performClick:", 0)
				cocoa.Msg(int64(a.Unwrap().View.Id), "performClick:", 0)
				app.After(300*time.Millisecond, w.Close)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cocoa.GlassAvailable() {
		if bezel[1] == 16 {
			t.Errorf("glass bezel before macOS 26")
		}
		return
	}
	for i, name := range []string{"", "GlassButton, Circle", "Circle, GlassButton"} {
		if i > 0 && (bezel[i] != 16 || shape[i] != 3) {
			t.Errorf("%s: bezel %d, shape %d, want 16 and 3", name, bezel[i], shape[i])
		}
	}
	if bezel[0] == 16 {
		t.Errorf("a button without GlassButton has the glass bezel")
	}
	if plainClicks != 1 {
		t.Fatalf("performClick on a plain button reached OnClick %d times, want 1: the check is unreliable", plainClicks)
	}
	if clicks != 1 {
		t.Errorf("click on the glass button reached OnClick %d times, want 1", clicks)
	}
}
