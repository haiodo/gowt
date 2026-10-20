//go:build !darwin && !windows

package look

import "github.com/haiodo/gowt/swt"

func setBackdrop(*swt.Shell, Backdrop) {}

func setRoundedCorners(*swt.Shell, bool) {}

func topInset(*swt.Shell) int { return 0 }

func classic() {}

func setFullSizeContent(*swt.Shell, bool) {}

func setGlass(*swt.Composite, bool) {}

func glassButton(*swt.Control) {}

func setDarkContent(*swt.Display, bool) {}

func setToolbar(*swt.Shell, []ToolbarItem) bool { return false }

func setGlassGroup(*swt.Composite, float64) {}

func sidebarInset() int32 { return 0 }

func setSidebarChrome(*swt.Shell, *swt.Composite, *swt.Control) {}

func syncSidebar(*swt.Shell, *swt.Composite, [4]int32, bool) {}

// sidebarDivider leaves a gap for the sash: an SWT sash here paints its own background and would cover
// the content edge.
func sidebarDivider() int32 { return 4 }
