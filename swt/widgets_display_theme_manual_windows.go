package swt

// forcedScheme is the ForceColorScheme value: 1 dark, 2 light, 0 the system decides.
var forcedScheme int32

// darkContentKey is the Display data key that turns on the dark theme of the controls (uxtheme).
const darkContentKey = "org.eclipse.swt.internal.win32.useDarkModeExplorerTheme"

// ForceColorScheme makes the title bars dark (1) or light (2) whatever the system is, or follow it
// again (0). The content (scroll bars, tables, buttons) takes the scheme only for windows created
// afterwards: the dark controls theme does not switch live. 0 leaves the content theme as it is.
func (this *Display) ForceColorScheme(scheme int32) {
	forcedScheme = scheme
	if scheme != 0 {
		this.SetData(darkContentKey, scheme == 1)
	}
	for _, shell := range this.GetShells() {
		if !shell.IsDisposed() {
			shell.SetTitleColoring(scheme == 1 || scheme == 0 && DisplayIsSystemDarkTheme())
		}
	}
}

// FollowSystemTheme is opt-in for plain swt users (gowt.Run calls it). It keeps the title bar of every shell in step with the system app theme (registry
// AppsUseLightTheme, the same source as DisplayIsSystemDarkTheme): at show and on every SWT.Settings.
// WM_SETTINGCHANGE already fires SWT.Settings; the content of the window is not themed (needs undocumented uxtheme).
func (this *Display) FollowSystemTheme() {
	apply := &ListenerFunc{fn: func(*Event) {
		dark := DisplayIsSystemDarkTheme()
		if forcedScheme != 0 {
			dark = forcedScheme == 1
		}
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
