package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

const (
	dwmWindowCornerPreference = 33 // DWMWA_WINDOW_CORNER_PREFERENCE
	dwmCornerRound            = 2  // DWMWCP_ROUND
)

// fluentPopover gives the popover the Fluent layer colour and, on Windows 11, the rounded corners
// of a flyout. Older Windows ignores the attribute and keeps square corners.
func fluentPopover(s *swt.Shell, body *swt.Composite) {
	fluent.Background(&s.Composite)
	fluent.Background(body)
	win32.OSDwmSetWindowAttribute(int64(s.Handle), dwmWindowCornerPreference, []int32{dwmCornerRound}, 4)
}
