package gowt

import (
	"runtime"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestFields(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		form := put[*Panel](w, Fields(GrowX(),
			Field("Name", Text(GrowX())),
			Field("Notes", Text(Multiline(), GrowX(), Size(0, 60)), FieldTop())))
		if len(*errs) != 0 {
			t.Fatalf("reported %q", *errs)
		}
		kids := form.Unwrap().GetChildren()
		if len(kids) != 4 {
			t.Fatalf("%d children, want 2 labels and 2 controls", len(kids))
		}
		h := int32(swt.BEGINNING)
		if runtime.GOOS == "darwin" {
			h = swt.END
		}
		for i, v := range []int32{swt.CENTER, swt.BEGINNING} {
			d := kids[2*i].GetLayoutData().(*swt.GridData)
			if d.HorizontalAlignment != h || d.VerticalAlignment != v {
				t.Errorf("label %d: align %d,%d want %d,%d", i, d.HorizontalAlignment, d.VerticalAlignment, h, v)
			}
		}
	})
}
