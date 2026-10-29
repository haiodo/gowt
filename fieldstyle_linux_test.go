package gowt

import (
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestFieldStyleRegistersDisposal(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		for _, c := range []struct {
			name  string
			style func(*swt.Text)
		}{
			{"fill", func(t *swt.Text) { setFieldFill(t, 6) }},
			{"stroke", func(t *swt.Text) { setFieldStroke(t, 1, 2, 3, 255) }},
			{"inset", func(t *swt.Text) { setFieldInset(t, 8) }},
		} {
			field := put[*TextView](w, Text()).Unwrap()
			c.style(field)
			if field.GetDataKey(entryLookKey) == nil {
				t.Errorf("%s: entry look disposal not registered", c.name)
			}
			field.Dispose()
		}
	})
}
