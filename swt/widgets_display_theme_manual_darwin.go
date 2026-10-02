package swt

import "github.com/haiodo/gowt/internal/cocoa"

// FollowSystemTheme fires SWT.Settings when NSApp.effectiveAppearance changes (Dark/Light/Auto switch
// or a Display.SetDarkThemePreferred), refreshing the cached system colors through RunSettings.
func (this *Display) FollowSystemTheme() {
	cocoa.ObserveEffectiveAppearance(func() { this.runSettings = true })
}
