//go:build !linux

package gowt

import "github.com/haiodo/gowt/swt"

// showBadge sets the TaskBar items: macOS has only the application item (the Dock tile),
// Windows only the window items (taskbar buttons).
func (a *App) showBadge() {
	bar := a.display.GetSystemTaskBar()
	if bar == nil {
		return
	}
	if it := bar.GetItemShell(nil); it != nil {
		it.SetOverlayText(a.badge)
	}
	for _, s := range a.display.GetShells() {
		if it := bar.GetItemShell(s); it != nil {
			it.SetOverlayText(a.badge)
		}
	}
}

// badgeWindow gives a new window's taskbar button the badge on Windows.
func (a *App) badgeWindow(s *swt.Shell) {
	if a.badge == "" {
		return
	}
	bar := a.display.GetSystemTaskBar()
	if bar == nil {
		return
	}
	if it := bar.GetItemShell(s); it != nil {
		it.SetOverlayText(a.badge)
	}
}
