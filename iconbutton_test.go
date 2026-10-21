package gowt

import (
	"strings"
	"testing"
)

func TestIconButtonOptions(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		put[*ButtonView](w, Button("", Borderless(), Size(40, 40), AccessibleLabel("Attach"), Tooltip("Attach")))
		put[*ButtonView](w, Button("", Circle(), Borderless(), Size(40, 40)))
		put[*LabelView](w, Label("x", AccessibleLabel("Name")))
		if len(*errs) != 0 {
			t.Fatalf("valid use reported %q", *errs)
		}
		put[*ButtonView](w, Button("c", Check(), Borderless()))
		put[*ButtonView](w, Button("r", Radio(), Circle()))
		put[*LabelView](w, Label("l", Circle()))
		put[*SwitchView](w, Switch("s", Borderless()))
		want := []string{
			"gowt: Borderless applies to a push Button, not to Button",
			"gowt: Circle applies to a push Button, not to Button",
			"gowt: Circle applies to a push Button, not to Label",
			"gowt: Borderless applies to a push Button, not to Button",
		}
		if got := strings.Join(*errs, "\n"); got != strings.Join(want, "\n") {
			t.Errorf("errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
		}
	})
}
