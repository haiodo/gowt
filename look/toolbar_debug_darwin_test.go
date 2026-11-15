package look_test

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
)

// toolbarDump is the toolbar state the zone check prints when it fails, in window points.
func toolbarDump(w *gowt.Window) string { return cocoa.DescribeSidebarToolbar(w.Unwrap().View) }
