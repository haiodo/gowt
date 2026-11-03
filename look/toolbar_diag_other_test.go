//go:build !windows

package look_test

import "github.com/haiodo/gowt/swt"

// shellDiag is the awaitSize geometry snapshot's platform part; Windows logs DPI and size limits.
func shellDiag(*swt.Shell) string { return "" }
