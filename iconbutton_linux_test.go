package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/gtk"
)

func TestIconButtonStyleClasses(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		plain := put[*ButtonView](w, Button("", Size(40, 40)))
		flat := put[*ButtonView](w, Button("", Borderless(), Size(40, 40)))
		round := put[*ButtonView](w, Button("", Circle(), Size(40, 40)))
		for _, c := range []struct {
			name         string
			b            *ButtonView
			flat, circle bool
		}{{"plain", plain, false, false}, {"Borderless", flat, true, false}, {"Circle", round, false, true}} {
			h := c.b.Unwrap().Handle
			if got := gtk.HasStyleClass(h, "flat"); got != c.flat {
				t.Errorf("%s: flat class %v, want %v", c.name, got, c.flat)
			}
			if got := gtk.HasStyleClass(h, "circular"); got != c.circle {
				t.Errorf("%s: circular class %v, want %v", c.name, got, c.circle)
			}
		}
	})
}
