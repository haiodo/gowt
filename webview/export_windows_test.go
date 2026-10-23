package webview

import "github.com/haiodo/gowt/swt"

// SetPageDialog replaces the box a page's own alert, confirm, prompt or beforeunload would show.
func SetPageDialog(f func(kind int32, msg, def string) (bool, string)) {
	pageDialog = func(_ *swt.Shell, kind int32, msg, def string) (bool, string) { return f(kind, msg, def) }
}
