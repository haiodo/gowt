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

// sidebarItemWidth is the actual AppKit item viewer width: 36 points on some macOS versions, 44 on others.
func sidebarItemWidth(w *gowt.Window) int {
	for _, b := range cocoa.ToolbarButtonsOf(w.Unwrap().View).List() {
		if b.ID == "toggle-sidebar" && b.Bounds[2] > 0 {
			return b.Bounds[2]
		}
	}
	return 36
}
