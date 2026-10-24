package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// textScrollOnOverflow: SWT maps V_SCROLL to a bar that is always there, Multiline wants one on overflow only.
func textScrollOnOverflow(t *swt.Text) {
	if t.GetStyle()&swt.V_SCROLL == 0 {
		return
	}
	if s := gtk.Parent(t.Handle); s != 0 && gtk.GTKGTK_IS_SCROLLED_WINDOW(s) {
		gtk.SetScrollPolicyAutomatic(s)
	}
}
