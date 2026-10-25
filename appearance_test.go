package gowt

import "testing"

func TestSetAppearance(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		light, dark := RGB{10, 20, 30}, RGB{200, 210, 220}
		l := put[*LabelView](w, Label("x", Foreground(Adaptive{Light: light, Dark: dark})))
		var seen []bool
		app.OnThemeChange(func(d bool) { seen = append(seen, d) })
		system := app.Dark()

		app.SetAppearance(AppearanceDark)
		if !app.Dark() || app.Appearance() != AppearanceDark || fg(l.ctl) != dark {
			t.Errorf("dark: Dark %v, label %v", app.Dark(), fg(l.ctl))
		}
		app.SetAppearance(AppearanceLight)
		if app.Dark() || fg(l.ctl) != light {
			t.Errorf("light: Dark %v, label %v", app.Dark(), fg(l.ctl))
		}
		app.SetAppearance(AppearanceSystem)
		if app.Appearance() != AppearanceSystem {
			t.Error("Appearance after AppearanceSystem")
		}
		if len(seen) == 0 {
			t.Errorf("OnThemeChange did not run (system dark %v)", system)
		}
	})
}
