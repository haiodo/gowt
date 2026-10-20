//go:build darwin

// Hand-written: the placeholder of a multi-line text view.
package cocoa

// SetTextViewPlaceholder sets NSTextView's private placeholderString: AppKit draws it in the
// view's font and placeholder colour at the first line's inset while the view is empty.
func SetTextViewPlaceholder(view *NSView, s string) {
	v := uintptr(view.Id)
	if !respondsTo(v, "setPlaceholderString:") {
		return
	}
	msg(v, "setPlaceholderString:", nsString(s))
	msg(v, "setNeedsDisplay:", 1)
}

// TextViewPlaceholder reads placeholderString back, "" if the view lacks it.
func TextViewPlaceholder(view *NSView) string {
	v := uintptr(view.Id)
	if !respondsTo(v, "placeholderString") {
		return ""
	}
	return goString(msg(v, "placeholderString"))
}
