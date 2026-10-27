package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// A SetMaterial pane under a canvas in a Split, on white content: the pane has the platform material
// (materialProblem), and a StyledText in it with NoBackground and NoBorder has an alpha-0 background
// and no frame, with nothing reported to OnError.
func TestMaterialPane(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	white := gowt.RGB{R: 255, G: 255, B: 255}
	var errs []error
	var problem string
	var textAlpha, border int32
	var style int32
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("material")
			w.SetSize(480, 360)
			var pane *gowt.Panel
			var text *gowt.StyledTextView
			w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(0), gowt.Background(white),
				gowt.Split(gowt.Vertical(), gowt.Grow(),
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
				problem = materialProblem(w, pane, white)
				w.Close()
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
	if textAlpha != 0 {
		t.Errorf("StyledText background alpha %d, want 0", textAlpha)
	}
	if style&swt.BORDER != 0 || border != 0 {
		t.Errorf("StyledText with NoBorder: BORDER style %v, border width %d, want none", style&swt.BORDER != 0, border)
	}
}
