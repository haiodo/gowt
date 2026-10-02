package swt

import "github.com/haiodo/gowt/internal/cocoa"

// FollowSystemTheme is opt-in for plain swt users (gowt.Run calls it). It fires SWT.Settings when NSApp.effectiveAppearance changes (Dark/Light/Auto switch
// or a Display.SetDarkThemePreferred), refreshing the cached system colors through RunSettings.
func (this *Display) FollowSystemTheme() {
	cocoa.ObserveEffectiveAppearance(func() { this.runSettings = true })
}
