//go:build windows

package look

import (
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

// pillTabStop puts the segment canvas in the shell's Tab order. The Button the pill replaced carries
// WS_TABSTOP in its class style; a Canvas does not, so the bit is added here. SWT reads the style on
// every traversal, so setting it once after creation is enough.
func pillTabStop(c *swt.Canvas) {
	bits := win32.OSGetWindowLong(c.Handle, win32.OSGWL_STYLE)
	win32.OSSetWindowLong(c.Handle, win32.OSGWL_STYLE, bits|win32.OSWS_TABSTOP)
}
