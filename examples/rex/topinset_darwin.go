//go:build darwin

package main

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
)

// titleInset is the height of the title bar and toolbar strip at the top of the full-size window.
func titleInset(w *gowt.Window) int {
	return cocoa.ContentTopInset(w.Unwrap().View)
}
