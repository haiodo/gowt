package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// alphaAt draws v without the views behind it and returns the alpha at x, y from its top left.
func alphaAt(v *cocoa.NSView, x, y float64) float64 {
	b := v.Bounds()
	rep := v.BitmapImageRepForCachingDisplayInRect(b)
	v.CacheDisplayInRect(b, rep)
	return cocoa.NewNSColorOverload1(cocoa.Msg(rep.Id, "colorAtX:y:", int64(x*float64(rep.PixelsWide())/b.Width), int64(y*float64(rep.PixelsHigh())/b.Height))).AlphaComponent()
}

// Nothing in a SetGlass panel paints a background: Scroll, StyledText, a Label added later. Painting the
// StyledText and adding a Split pane (a new sash) after SetGlass must not hit a disposed background Color.
func TestGlassContentClear(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var scroll *gowt.ScrollView
	var split *gowt.SplitView
	var errs []error
	var text *gowt.StyledTextView
	var late *gowt.LabelView
	var labelAlpha, lateAlpha, textBgAlpha int32
	var scrollPix, textPix float64
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("glass")
			w.SetSize(400, 300)
			var glass *gowt.Panel
			var label *gowt.LabelView
			w.Add(gowt.VBox(gowt.Ref(&glass), gowt.Grow(), gowt.Padding(12),
				gowt.Scroll(gowt.Grow(), gowt.Ref(&scroll),
					gowt.Label("label", gowt.Ref(&label)),
					gowt.StyledText(gowt.ReadOnly(), gowt.Wrap(), gowt.GrowX(), gowt.Size(0, 80), gowt.Ref(&text),
						gowt.Init(func(t *gowt.StyledTextView) { t.SetText("styled") }))),
				gowt.Split(gowt.GrowX(), gowt.Size(0, 40), gowt.Ref(&split), gowt.Label("a"), gowt.Label("b"))))
			look.SetGlass(glass, true)
			w.Show()
			app.After(100*time.Millisecond, func() {
				scroll.Add(gowt.Label("late", gowt.Ref(&late)))
				split.Add(gowt.Label("c"))
				app.After(200*time.Millisecond, func() {
					labelAlpha = label.Unwrap().GetBackground().GetAlpha()
					lateAlpha = late.Unwrap().GetBackground().GetAlpha()
					textBgAlpha = text.Unwrap().GetBackground().GetAlpha()
					sv := scroll.Unwrap().View
					scrollPix = alphaAt(sv, sv.Bounds().Width-4, sv.Bounds().Height-4)
					tv := text.Unwrap().View
					textPix = alphaAt(tv, tv.Bounds().Width-4, tv.Bounds().Height-4)
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	for name, a := range map[string]int32{"label": labelAlpha, "label added later": lateAlpha, "StyledText": textBgAlpha} {
		if a != 0 {
			t.Errorf("%s background alpha %d, want 0", name, a)
		}
	}
	if scrollPix > 0.01 {
		t.Errorf("Scroll paints alpha %.2f below its content, want 0", scrollPix)
	}
	if textPix > 0.01 {
		t.Errorf("StyledText paints alpha %.2f past its text, want 0", textPix)
	}
}
