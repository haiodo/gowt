package swt

// FollowSystemTheme is opt-in for plain swt users (gowt.Run calls it). It keeps the title bar of every shell in step with the system app theme (registry
// AppsUseLightTheme, the same source as DisplayIsSystemDarkTheme): at show and on every SWT.Settings.
// WM_SETTINGCHANGE already fires SWT.Settings; the content of the window is not themed (needs undocumented uxtheme).
func (this *Display) FollowSystemTheme() {
	apply := &ListenerFunc{fn: func(*Event) {
		dark := DisplayIsSystemDarkTheme()
		for _, shell := range this.GetShells() {
			if !shell.IsDisposed() {
				shell.SetTitleColoring(dark)
			}
		}
	}}
	this.AddFilter(Show, apply)
	this.AddListener(Settings, apply)
	apply.HandleEvent(nil)
}
