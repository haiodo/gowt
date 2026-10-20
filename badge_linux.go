package gowt

import (
	"strconv"
	"strings"

	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// showBadge sends the badge as the launcher count: SWT has no task bar on GTK.
func (a *App) showBadge() {
	n, ok := launcherCount(a.badge)
	gtk.LauncherEntryUpdate("application://"+swt.DisplayGetAppName()+".desktop", n, ok)
}

// badgeWindow does nothing: the launcher count belongs to the application, not to a window.
func (a *App) badgeWindow(*swt.Shell) {}

// launcherCount reads a badge as the whole number Linux docks draw; ok is false for any other text.
func launcherCount(text string) (n int64, ok bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	return n, err == nil
}
