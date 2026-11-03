//go:build !windows

package look_test

import "github.com/haiodo/gowt/swt"

// Other platforms do not clamp the requested test width.
func maxWindowWidth() int32 { return 1 << 30 }

// shellDiag is the awaitSize geometry snapshot's platform part; Windows logs DPI and size limits.
func shellDiag(*swt.Shell) string { return "" }
