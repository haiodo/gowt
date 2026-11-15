//go:build !darwin

package look_test

import "github.com/haiodo/gowt"

// toolbarDump has nothing to report: the zones of the native toolbar are macOS only.
func toolbarDump(*gowt.Window) string { return "" }
