package gowt

import "testing"

func TestIconsRender(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		w.SetLayout(Grid{Columns: 8, Margin: 5})
		for _, n := range IconNames() {
			img := app.Icon(n, IconSize(20))
			if wd, h := img.Size(); wd != 20 || h != 20 {
				t.Errorf("%s: size %dx%d", n, wd, h)
			}
			w.Button("", nil, Tooltip(n)).SetImage(img)
		}
	})
}
