package gowt

import "github.com/haiodo/gowt/swt"

// Dark reports whether the system is in dark mode (macOS appearance, Windows apps theme, Linux
// xdg-desktop-portal color-scheme). Run keeps the app following it; call from the UI thread.
func (a *App) Dark() bool { return systemDark() }

// OnThemeChange runs f on the UI thread when the system switches between light and dark.
// Cached widget colors are already refreshed when f runs.
func (a *App) OnThemeChange(f func(dark bool)) {
	last := a.Dark()
	a.display.AddListener(swt.Settings, themeListener(func() {
		if d := a.Dark(); d != last {
			last = d
			f(d)
		}
	}))
}

type themeListener func()

func (l themeListener) HandleEvent(*swt.Event) { l() }
