//go:build darwin

package main

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
)

// contentTopInset is the height of the strip the title bar and toolbar take at the top
// of a full-size-content window; 0 without full-size content.
func contentTopInset(w *gowt.Window) int {
	return cocoa.ContentTopInset(w.Unwrap().View)
}
