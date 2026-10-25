package swt

import "github.com/haiodo/gowt/internal/cocoa"

// FollowSystemTheme is opt-in for plain swt users (gowt.Run calls it). It fires SWT.Settings when NSApp.effectiveAppearance changes (Dark/Light/Auto switch
// or a Display.SetDarkThemePreferred), refreshing the cached system colors through RunSettings.
func (this *Display) FollowSystemTheme() {
	cocoa.ObserveEffectiveAppearance(func() { this.runSettings = true })
	cocoa.SyncCurrentAppearance()
	this.InitColors0() // colors cached in Display init were resolved before the appearance was synced
	// RunSettings marks only each shell's own view for display; in a look.SetSidebar window (glass
	// beside the views) the content composite kept its old default background after a switch.
	this.AddListener(Settings, &ListenerFunc{fn: func(*Event) {
		for _, s := range this.GetShells() {
			if s.window == nil {
				continue
			}
			if v := s.window.ContentView(); v != nil {
				cocoa.NeedsDisplayTree(v)
			}
		}
	}})
}

// ForceColorScheme makes the app dark (1) or light (2) whatever the system is, or follow it again (0).
func (this *Display) ForceColorScheme(scheme int32) {
	cocoa.SetAppearance(scheme)
	this.InitColors0()
	this.runSettings = true
}
