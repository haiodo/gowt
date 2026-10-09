// Package webview embeds the system web view (WKWebView on macOS) in an SWT Composite and bridges
// it to Go: custom URL schemes served from Go, page-to-Go messages, async script evaluation.
// Everything must be called on the SWT UI thread; callbacks run there too.
package webview

import (
	"encoding/json"
	"errors"

	"github.com/haiodo/gowt/swt"
)

type Options struct {
	// Inspectable lets Safari's Web Inspector attach (macOS 13.3+).
	Inspectable bool

	// host is set by NewOn: an engine that can use it as the hosting Composite does, the others put a child in it.
	host *swt.Composite
}

type Request struct{ URL, Method string }

type Response struct {
	Status   int // 0 means 200
	MimeType string
	Headers  map[string]string
	Body     []byte
}

// SchemeHandler answers requests to a custom scheme; a nil Response is a 404.
type SchemeHandler func(req Request) *Response

type WebView struct {
	// OnMessage gets what the page passes to window.gowt.postMessage (non-strings as JSON).
	OnMessage            func(msg string)
	OnNavigationStarted  func(url string)
	OnNavigationFinished func(url string)
	OnNavigationFailed   func(url string, err error)
	OnTitleChanged       func(title string)

	e engine
}

type engine interface {
	control() *swt.Composite
	navigate(url string)
	setHTML(html, baseURL string)
	eval(js string, done func(result string, err error))
	addScript(js string)
	goBack()
	goForward()
	reload()
	url() string
	handleScheme(scheme string, h SchemeHandler) error
	dispose()
}

// New adds a web view filling a new Composite inside parent. The native view is created at the first
// load, so HandleScheme and AddScript must come before it.
func New(parent *swt.Composite, opts Options) (*WebView, error) {
	w := &WebView{}
	e, err := newEngine(w, parent, opts)
	if err != nil {
		return nil, err
	}
	w.e = e
	return w, nil
}

// Control is the Composite hosting the view: set its layout data, size and visibility.
func (w *WebView) Control() *swt.Composite { return w.e.control() }

func (w *WebView) Navigate(url string)          { w.e.navigate(url) }
func (w *WebView) SetHTML(html, baseURL string) { w.e.setHTML(html, baseURL) }

// AddScript runs js at document start, before page scripts, in every page loaded afterwards.
func (w *WebView) AddScript(js string) { w.e.addScript(js) }

// HandleScheme serves scheme:// URLs from Go. It fails once the page has been loaded.
func (w *WebView) HandleScheme(scheme string, h SchemeHandler) error {
	return w.e.handleScheme(scheme, h)
}

// Eval runs js as a global script and reports its value as JSON text ("" for undefined), or the
// script error, asynchronously.
func (w *WebView) Eval(js string, done func(result string, err error)) {
	q, _ := json.Marshal(js)
	w.e.eval("JSON.stringify((0,eval)("+string(q)+"))", done)
}

func (w *WebView) GoBack()    { w.e.goBack() }
func (w *WebView) GoForward() { w.e.goForward() }
func (w *WebView) Reload()    { w.e.reload() }

// URL is the current page's URL ("" before the first load).
func (w *WebView) URL() string { return w.e.url() }

func (w *WebView) Dispose() { w.e.dispose() }

var errNotImplemented = errors.New("webview: not implemented on this OS")
