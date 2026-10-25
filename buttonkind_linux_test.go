package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/gtk"
)

func checkButtonKinds(t *testing.T, plain, primary, destructive *ButtonView) {
	for _, c := range []struct {
		name               string
		b                  *ButtonView
		suggested, destroy bool
	}{{"plain", plain, false, false}, {"Primary", primary, true, false}, {"Destructive", destructive, false, true}} {
		h := c.b.Unwrap().Handle
		if got := gtk.HasStyleClass(h, "suggested-action"); got != c.suggested {
			t.Errorf("%s: suggested-action %v, want %v", c.name, got, c.suggested)
		}
		if got := gtk.HasStyleClass(h, "destructive-action"); got != c.destroy {
			t.Errorf("%s: destructive-action %v, want %v", c.name, got, c.destroy)
		}
	}
}
