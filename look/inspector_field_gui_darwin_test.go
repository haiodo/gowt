package look_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// Small Number fields on the inspector glass, light and dark: the field editor of the focused one
// starts after its prefix, and each scrubbed field has its handle. GOWT_PIXELS_DIR keeps the PNGs.
func TestInspectorFields(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, mode := range []gowt.Appearance{gowt.AppearanceLight, gowt.AppearanceDark} {
		name := map[gowt.Appearance]string{gowt.AppearanceLight: "light", gowt.AppearanceDark: "dark"}[mode]
		t.Run(name, func(t *testing.T) {
			r := captureInspectorFields(t, mode, filepath.Join(dir, "inspector-fields-"+name+".png"))
			if !r.editing {
				t.Fatal("the Focused field is not being edited")
			}
			if r.valueStart < 10 || math.Abs(r.editorX-r.valueStart) > 1 {
				t.Errorf("field editor at x %.1f, value drawn from %.1f: the editor must start after the prefix", r.editorX, r.valueStart)
			}
			if r.handles != 4 {
				t.Errorf("%d of 4 scrubbed fields have a handle", r.handles)
			}
		})
	}
}

type fieldReport struct {
	editorX, valueStart float64
	editing             bool
	handles             int
}

// captureInspectorFields shows small fields in an inspector, X focused, and saves the window to path.
func captureInspectorFields(t *testing.T, mode gowt.Appearance, path string) (r fieldReport) {
	t.Helper()
	var errs []error
	var err, shotErr error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("inspector fields")
			var content, insp *gowt.Panel
			n := make([]*gowt.NumberView, 4)
			field := func(i int, opts ...gowt.Option) gowt.Option {
				return gowt.Number(-1000, 1000, append([]gowt.Option{gowt.Small(), gowt.GrowX(), gowt.Ref(&n[i])}, opts...)...)
			}
			w.Add(gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(16), gowt.Gap(10),
					gowt.Label("Layout", gowt.Typeface(gowt.Type.BodyStrong)),
					gowt.Grid(2, gowt.GrowX(),
						field(0, gowt.Prefix("X"), gowt.FieldFill(gowt.RGB{240, 240, 240}), gowt.Value(260), gowt.Focused()),
						field(1, gowt.Prefix("Y"), gowt.Value(-40)),
						field(2, gowt.Prefix("W"), gowt.Suffix("px"), gowt.Value(120)),
						gowt.Number(0, 100, gowt.Prefix("Gap"), gowt.Small(), gowt.GrowX(), gowt.Value(8))),
					field(3, gowt.PrefixImage(icons.Get(app, "star", icons.Size(12))), gowt.Suffix("%"), gowt.Step(0.5), gowt.Value(87.5)),
					gowt.Text(gowt.Prefix("Name"), gowt.Small(), gowt.Border(), gowt.Value("Server"), gowt.GrowX()),
					gowt.Number(0, 100, gowt.Prefix("R"), gowt.Value(12), gowt.GrowX(), gowt.Disabled())))
			look.SetInspector(w, content, insp, 260)
			w.SetSize(700, 420)
			w.Show()
			app.After(800*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				v := n[0].Unwrap().View
				r.editorX, r.editing = cocoa.FieldEditorX(v)
				r.valueStart = cocoa.FieldValueStart(v)
				for _, f := range n {
					if cocoa.FieldAffixHandle(f.Unwrap().View) != 0 {
						r.handles++
					}
				}
				shotErr = shot.WindowPNG(shotHandle(w), path)
			})
		})
	})
	for _, e := range append(errs, err, shotErr) {
		if e != nil {
			t.Fatal(e)
		}
	}
	return r
}
