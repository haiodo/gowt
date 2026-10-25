package gowt

import "testing"

func TestButtonKinds(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		plain := put[*ButtonView](w, Button("Cancel"))
		save := put[*ButtonView](w, Button("Save", Primary()))
		del := put[*ButtonView](w, Button("Delete", Destructive()))
		if len(*errs) != 0 {
			t.Fatalf("valid use reported %q", *errs)
		}
		if got := w.Unwrap().GetDefaultButton(); got != save.Unwrap() {
			t.Errorf("default button is %v, want the Primary one", got)
		}
		checkButtonKinds(t, plain, save, del)
		put[*ButtonView](w, Button("c", Check(), Primary()))
		put[*LabelView](w, Label("l", Destructive()))
		if len(*errs) != 2 {
			t.Errorf("misuse reported %q", *errs)
		}
	})
}
