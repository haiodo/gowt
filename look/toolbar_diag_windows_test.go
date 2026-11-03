//go:build windows

package look_test

import (
	"fmt"

	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

// Windows clamps SetWindowPos to the display's maximum tracking size; use 96-DPI
// units because the test's sizes and GetSize are logical coordinates.
func maxWindowWidth() int32 { return win32.OSGetSystemMetricsForDpi(win32.OSSM_CXMAXTRACK, 96) }

// Display metrics distinguish a Windows size clamp from a toolbar layout failure.
func shellDiag(s *swt.Shell) string {
	return fmt.Sprintf("dpi=%d zoom=%d screen=%dx%d maxTrack=%dx%d min=%v max=%v maximized=%v",
		win32.OSGetDpiForWindow(s.Handle), s.GetShellZoom(),
		win32.OSGetSystemMetrics(win32.OSSM_CXSCREEN), win32.OSGetSystemMetrics(win32.OSSM_CYSCREEN),
		win32.OSGetSystemMetrics(win32.OSSM_CXMAXTRACK), win32.OSGetSystemMetrics(win32.OSSM_CYMAXTRACK),
		s.GetMinimumSize(), s.GetMaximumSize(), s.GetMaximized())
}
