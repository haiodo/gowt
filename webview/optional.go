package webview

import (
	"errors"

	"github.com/haiodo/gowt/swt"
)

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

type prestarter interface {
	prestart(done func())
}

type callHandler interface {
	setCallHandler(f func(msg string) string)
}

// Prestart creates the native view now instead of at the first load and calls done once it is ready or has
// failed; it reports false, without calling done, on an engine that has nothing to prepare. Like the first
// load, it ends HandleScheme and AddScript.
func (w *WebView) Prestart(done func()) bool {
	p, ok := w.e.(prestarter)
	if ok {
		p.prestart(done)
	}
	return ok
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
// Only calls made by the main frame reach f: a subframe (maybe cross-origin) must get null and f is
// not called, or any iframe could use what f hands out. f may panic; the engine then answers null.
// While f runs the page is blocked, so f cannot wait for a script evaluated in that page.
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

// WindowFeatures are what window.open asked for. A zero Width and Height mean the page gave no size.
// X and Y are in the engine's own units (points on macOS, pixels elsewhere).
type WindowFeatures struct {
	X, Y, Width, Height                      int
	MenuBar, StatusBar, ToolBar, LocationBar bool
}

// NewWindow is a page's request for a new window (window.open, a link with target=_blank). It is valid
// only while the handler given to SetNewWindowHandler runs.
type NewWindow struct {
	// URL the page asked for; "" or "about:blank" for window.open() without an address.
	URL string

	opener engine
	done   bool
	view   *WebView

	// native is the engine's own handle of the request (macOS: the WKWebViewConfiguration, Windows: the
	// event args, kept until the new view is ready); deferral is Windows' deferral of that event.
	native, deferral uintptr
	features         WindowFeatures
}

// NewOn makes the view that takes the page of the request, with host as its hosting Composite as the
// package-level NewOn does. Call it inside the handler, at most once; the handler returns the view. The new view shares the user scripts, the
// message handler and the session (cookies, storage) of the opener, as the engine's own popups do:
// scripts added with AddScript later are not added to a popup, and on Linux OnMessage of the popup
// and the opener both hear the page of either.
func (n *NewWindow) NewOn(host *swt.Composite, opts Options) (*WebView, error) {
	if n.done {
		return nil, errors.New("webview: the new window request is over")
	}
	if n.view != nil {
		return nil, errors.New("webview: the new window request has its view already")
	}
	opts.popup = n
	wv, err := NewOn(host, opts)
	if err == nil {
		n.view = wv
	}
	return wv, err
}

type windowOpener interface {
	setNewWindowHandler(f func(n *NewWindow) *WebView)
}

type windowEvents interface {
	setShowHandler(f func(f WindowFeatures))
	setCloseHandler(f func())
}

type statusEvents interface {
	setStatusHandler(f func(text string))
}

type scriptSwitch interface {
	setScriptEnabled(on bool)
}

type requestLoader interface {
	loadRequest(r LoadRequest)
}

// SetNewWindowHandler lets f decide what a page's request for a new window gets: f returns the
// WebView made with n.NewOn that shows the page, or nil to refuse the window. The new view is loaded by the engine with the requested address; it is not shown or
// placed by the package: put its Control in a window and listen to SetShowHandler for the moment the
// page wants it shown. It reports whether the engine can; set it before the first load.
// Without a handler the engine's own default applies: the window is refused on macOS and Linux, and
// WebView2 opens a browser window of its own on Windows.
// While f runs the opener's page is blocked: do not wait there for a script of the opener.
func (w *WebView) SetNewWindowHandler(f func(n *NewWindow) *WebView) bool {
	o, ok := w.e.(windowOpener)
	if ok {
		o.setNewWindowHandler(func(n *NewWindow) *WebView {
			defer func() { n.done = true }()
			r := f(n)
			if r != n.view {
				return nil // not the view made for this request
			}
			return r
		})
	}
	return ok
}

// SetShowHandler calls f once for a view made by NewOn, when its page wants to be shown, with the
// window.open features. The engines that have no such moment (macOS, Windows) call it right after the
// view is made. It reports whether the engine can.
func (w *WebView) SetShowHandler(f func(WindowFeatures)) bool {
	e, ok := w.e.(windowEvents)
	if ok {
		e.setShowHandler(f)
	}
	return ok
}

// SetCloseHandler calls f when the page asks to close its window (window.close); the view is not closed
// by the package: f disposes whatever hosts it. It reports whether the engine can.
func (w *WebView) SetCloseHandler(f func()) bool {
	e, ok := w.e.(windowEvents)
	if ok {
		e.setCloseHandler(f)
	}
	return ok
}

// SetStatusHandler calls f with the address of the link under the pointer, and with "" when the pointer
// leaves it. It reports whether the engine can; on macOS the page tells it (a script injected in every
// frame), so a page that stops mouse events or a frame the script does not reach is silent.
func (w *WebView) SetStatusHandler(f func(text string)) bool {
	e, ok := w.e.(statusEvents)
	if ok {
		e.setStatusHandler(f)
	}
	return ok
}

// SetScriptEnabled switches the scripts of the pages loaded after the call; the page on screen keeps
// what it has. On macOS and Windows Eval still runs scripts in such a page; on Linux it does not
// answer. It reports whether the engine can.
func (w *WebView) SetScriptEnabled(on bool) bool {
	s, ok := w.e.(scriptSwitch)
	if ok {
		s.setScriptEnabled(on)
	}
	return ok
}

// LoadRequest is a navigation with a method, headers and a body.
type LoadRequest struct {
	URL string
	// Method is "GET" or "POST"; "" is GET.
	Method  string
	Headers map[string]string
	Body    []byte
}

// Load navigates with r. It reports whether the engine can send r: the engines can send a GET with headers
// and a POST. On Linux the body of a POST goes through Go's net/http, not WebKit (WebKitGTK has no API
// for it), so WebKit's cookie jar and proxy settings are not used for that request, and a failure shows
// as a page with the error text.
func (w *WebView) Load(r LoadRequest) bool {
	l, ok := w.e.(requestLoader)
	if ok {
		l.loadRequest(r)
	}
	return ok
}

// Cookie is a cookie the session holds.
type Cookie struct{ Name, Value string }

// Cookies reports the cookies the session would send to url. Calls and callbacks are on the UI thread.
// ok is false when the engine has no cookie store to ask (no view exists yet on Windows).
func Cookies(url string, done func(cookies []Cookie, ok bool)) { cookies(url, done) }

// SetCookie stores header (the value of a Set-Cookie field: "name=value; Path=/; Expires=...") as if the
// page at url had sent it, and reports whether the store took it.
func SetCookie(url, header string, done func(ok bool)) { setCookie(url, header, done) }

// ClearSessionCookies deletes the cookies without an expiry date; done runs when that is over.
func ClearSessionCookies(done func()) { clearSessionCookies(done) }
