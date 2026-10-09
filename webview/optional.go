package webview

import "github.com/haiodo/gowt/swt"

// Optional engine features. An engine implements the unexported method of each one its web view can
// do (checked by type assertion, so the engine interface stays the same on every OS); the WebView
// method then reports false, or does nothing, on an engine without it.

type navigator interface {
	canGoBack() bool
	canGoForward() bool
	stop()
}

type navigationPolicy interface {
	setNavigationPolicy(f func(url string, mainFrame bool) bool)
}

type callHandler interface {
	setCallHandler(f func(msg string) string)
}

// CanGoBack and CanGoForward report whether the history has an entry to go to; known is false when
// the engine cannot tell.
func (w *WebView) CanGoBack() (can, known bool) {
	if n, ok := w.e.(navigator); ok {
		return n.canGoBack(), true
	}
	return false, false
}

func (w *WebView) CanGoForward() (can, known bool) {
	if n, ok := w.e.(navigator); ok {
		return n.canGoForward(), true
	}
	return false, false
}

// Stop cancels the page load in progress; a no-op without the feature.
func (w *WebView) Stop() {
	if n, ok := w.e.(navigator); ok {
		n.stop()
	}
}

// SetNavigationPolicy makes the engine ask f before every navigation, in the main frame or a
// subframe; false cancels it. It reports whether the engine can ask. Set it before the first load.
func (w *WebView) SetNavigationPolicy(f func(url string, mainFrame bool) bool) bool {
	n, ok := w.e.(navigationPolicy)
	if ok {
		n.setNavigationPolicy(f)
	}
	return ok
}

// SetCallHandler answers window.gowt.call(msg) in the page: the page blocks until f returns its
// reply as a string (msg is a string or the JSON of any other value). It reports whether the engine
// can; without it window.gowt.call is missing or returns null. Set it before the first load.
func (w *WebView) SetCallHandler(f func(msg string) string) bool {
	n, ok := w.e.(callHandler)
	if ok {
		n.setCallHandler(f)
	}
	return ok
}

// NewOn is New with host itself as the hosting Composite instead of a new child of it, for a widget
// that is the web view (swt Browser). An engine that cannot host in an existing Composite adds a
// child to host as New does.
func NewOn(host *swt.Composite, opts Options) (*WebView, error) {
	opts.host = host
	return New(host, opts)
}
