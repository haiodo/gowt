package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

const solidHorizontalGridLineMask = 2 // NSTableViewSolidHorizontalGridLineMask

// rowLinesOnly sets horizontal grid lines without alternating rows; it reports false for a control that is no table view.
func rowLinesOnly(c *swt.Control, on bool) bool {
	var tv *cocoa.NSTableView
	switch v := c.View.Impl().(type) {
	case *cocoa.SWTTableView:
		tv = &v.NSTableView
	case *cocoa.SWTOutlineView:
		tv = &v.NSTableView
	default:
		return false
	}
	tv.SetUsesAlternatingRowBackgroundColors(false)
	mask := int64(cocoa.OSNSTableViewGridNone)
	if on {
		mask = solidHorizontalGridLineMask
	}
	tv.SetGridStyleMask(mask)
	return true
}
