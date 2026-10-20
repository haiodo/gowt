package gowt

// SetBadge shows text on the application icon, such as an unread count; "" removes it.
// macOS: the badge of the app's Dock tile. Windows: the overlay of the taskbar button of every window,
// windows created later by App.Window too. Linux: the count on the launcher icon in Ubuntu Dock,
// dash-to-dock and KDE Plasma, which show a whole number only, so other text hides the count there.
// The launcher is the one of the desktop file named after the app name ("myapp.desktop" for
// SetAppName("myapp"), the executable name by default); see docs/packaging.md. UI thread only.
func (a *App) SetBadge(text string) {
	a.badge = text
	a.showBadge()
}
