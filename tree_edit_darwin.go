package gowt

import "github.com/haiodo/gowt/swt"

// AppKit adds row views above SWT children, so the editor must be raised
// after attaching it to the outline view.
func raiseTreeEditor(c *swt.Control) { c.SetZOrderSiblingAbove(nil, true) }
