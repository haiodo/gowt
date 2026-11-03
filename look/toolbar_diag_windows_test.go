//go:build windows

package look_test

import (
	"fmt"

	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

// shellDiag is the awaitSize geometry snapshot's Windows part: DPI, autoscaling zoom, the shell's
// minimum and maximum, and the maximized state, so a CI where a size is never reached can tell a
// display limit (DPI/monitor/maximum) from a layout problem.
func shellDiag(s *swt.Shell) string {
	return fmt.Sprintf("dpi=%d zoom=%d min=%v max=%v maximized=%v", win32.OSGetDpiForWindow(s.Handle),
		s.GetShellZoom(), s.GetMinimumSize(), s.GetMaximumSize(), s.GetMaximized())
}
