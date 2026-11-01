package look_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

func traceMaterial(stage string) {
	if os.Getenv("GOWT_TRACE_LIFECYCLE") == "1" {
		fmt.Fprintln(os.Stderr, "[material]", stage)
	}
}

// A SetMaterial pane under a canvas in a Split, on white content: the pane has the platform material
// (materialProblem), and a StyledText in it with NoBackground and NoBorder has an alpha-0 background
// and no frame, with nothing reported to OnError; the sash beside the pane paints the divider.
func TestMaterialPane(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	white := gowt.RGB{R: 255, G: 255, B: 255}
	var errs []error
	var problem string
	var textAlpha, border int32
	var style, sashes, painted int32
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("material")
			w.SetSize(480, 360)
			var pane *gowt.Panel
			var text *gowt.StyledTextView
			var split *gowt.SplitView
			w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(0), gowt.Background(white),
				gowt.Split(gowt.Vertical(), gowt.Grow(), gowt.Ref(&split),
					gowt.Canvas(func(g *gowt.GC) {}, gowt.Background(white)),
					gowt.VBox(gowt.Ref(&pane), gowt.Padding(0),
						gowt.StyledText(gowt.NoBackground(), gowt.NoBorder(), gowt.Grow(), gowt.Ref(&text),
							gowt.Init(func(t *gowt.StyledTextView) { t.SetText("func main() {}") }))))))
			look.SetMaterial(pane, true)
			w.Show()
			app.After(400*time.Millisecond, func() {
				textAlpha = text.Unwrap().GetBackground().GetAlpha()
				style = text.Unwrap().GetStyle()
				border = text.Unwrap().GetBorderWidth()
				traceMaterial("materialProblem before")
				problem = materialProblem(w, pane, white)
				traceMaterial("materialProblem after; enumerate before")
				for _, ch := range split.Unwrap().GetChildren() {
					if _, ok := ch.Impl().(*swt.Sash); ok {
						sashes++
						if ch.IsListening(swt.Paint) {
							painted++
						}
					}
				}
				traceMaterial("enumerate after; close before")
				w.Close()
				traceMaterial("close after")
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	if problem != "" {
		t.Error(problem)
	}
	if sashes == 0 || painted != sashes {
		t.Errorf("%d of %d sashes paint the divider, want all", painted, sashes)
	}
	if textAlpha != 0 {
		t.Errorf("StyledText background alpha %d, want 0", textAlpha)
	}
	if style&swt.BORDER != 0 || border != 0 {
		t.Errorf("StyledText with NoBorder: BORDER style %v, border width %d, want none", style&swt.BORDER != 0, border)
	}
}
