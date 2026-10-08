package gowt

// SetDarkContent on Windows turns on the dark theme of the window content (scroll bars, tables, trees,
// buttons) through undocumented uxtheme exports. Call before creating windows; it does not switch live.
func (a *App) SetDarkContent(on bool) {
	a.display.SetData("org.eclipse.swt.internal.win32.useDarkModeExplorerTheme", on)
}
