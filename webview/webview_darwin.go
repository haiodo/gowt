package webview

import (
	"errors"
	"strings"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

type disposeFunc func()

func (f disposeFunc) WidgetDisposed(*swt.DisposeEvent) { f() }

type wkEngine struct {
	w       *WebView
	host    *swt.Composite
	opts    Options
	schemes map[string]SchemeHandler
	scripts []string
	v       *cocoa.WKView
	policy  func(url string, mainFrame bool) bool
	call    func(msg string) string
}

func newEngine(w *WebView, parent *swt.Composite, opts Options) (engine, error) {
	host := opts.host
	if host == nil {
		host = swt.NewCompositeParentStyle(parent, swt.NONE)
	}
	e := &wkEngine{w: w, host: host, opts: opts, schemes: map[string]SchemeHandler{}}
	e.host.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) {
		if e.v != nil {
			e.v.SetFrame(e.bounds())
		}
	}))
	e.host.AddDisposeListener(disposeFunc(e.release))
	return e, nil
}

func (e *wkEngine) bounds() cocoa.NSRect {
	b := e.host.View.Bounds()
	return cocoa.NSRect{Width: b.Width, Height: b.Height}
}

// start creates the native view; the scheme list and scripts are frozen from here on.
func (e *wkEngine) start() *cocoa.WKView {
	if e.v != nil {
		return e.v
	}
	cfg := cocoa.WKConfig{Parent: uintptr(e.host.View.Id), Frame: e.bounds(), Inspectable: e.opts.Inspectable, Scripts: e.scripts}
	for s := range e.schemes {
		cfg.Schemes = append(cfg.Schemes, s)
	}
	v := cocoa.NewWKView(cfg)
	w := e.w
	v.Message = func(m string) {
		if w.OnMessage != nil {
			w.OnMessage(m)
		}
	}
	v.NavStarted = func(u string) {
		if w.OnNavigationStarted != nil {
			w.OnNavigationStarted(u)
		}
	}
	v.NavFinished = func(u string) {
		if w.OnNavigationFinished != nil {
			w.OnNavigationFinished(u)
		}
	}
	v.NavFailed = func(u, msg string) {
		if w.OnNavigationFailed != nil {
			w.OnNavigationFailed(u, errors.New(msg))
		}
	}
	v.TitleChanged = func(t string) {
		if w.OnTitleChanged != nil {
			w.OnTitleChanged(t)
		}
	}
	v.Scheme = e.serve
	v.Decide = e.policy
	v.Call = e.call
	e.v = v
	return v
}

func (e *wkEngine) serve(url, method string) (int, map[string]string, []byte) {
	scheme, _, _ := strings.Cut(url, ":")
	h := e.schemes[scheme]
	if h == nil {
		return 404, nil, nil
	}
	r := h(Request{URL: url, Method: method})
	if r == nil {
		return 404, nil, nil
	}
	headers := map[string]string{}
	for k, v := range r.Headers {
		headers[k] = v
	}
	if r.MimeType != "" {
		headers["Content-Type"] = r.MimeType
	}
	if r.Status == 0 {
		r.Status = 200
	}
	return r.Status, headers, r.Body
}

func (e *wkEngine) control() *swt.Composite { return e.host }
func (e *wkEngine) navigate(url string)     { e.start().LoadURL(url) }
func (e *wkEngine) setHTML(html, base string) {
	e.start().LoadHTML(html, base)
}

func (e *wkEngine) eval(js string, done func(string, error)) {
	e.start().Eval(js, func(res, msg string) {
		if msg != "" {
			done("", errors.New(msg))
			return
		}
		done(res, nil)
	})
}

func (e *wkEngine) addScript(js string) {
	if e.v != nil {
		e.v.AddScript(js)
		return
	}
	e.scripts = append(e.scripts, js)
}

func (e *wkEngine) handleScheme(scheme string, h SchemeHandler) error {
	if e.v != nil {
		return errors.New("webview: HandleScheme after the first load")
	}
	e.schemes[scheme] = h
	return nil
}

func (e *wkEngine) release() {
	if e.v != nil {
		e.v.Dispose()
	}
}

func (e *wkEngine) dispose() { e.host.Dispose() }

func (e *wkEngine) goBack() {
	if e.v != nil {
		e.v.GoBack()
	}
}

func (e *wkEngine) goForward() {
	if e.v != nil {
		e.v.GoForward()
	}
}

func (e *wkEngine) reload() {
	if e.v != nil {
		e.v.Reload()
	}
}

func (e *wkEngine) url() string {
	if e.v == nil {
		return ""
	}
	return e.v.URL()
}

func (e *wkEngine) canGoBack() bool    { return e.v != nil && e.v.CanGoBack() }
func (e *wkEngine) canGoForward() bool { return e.v != nil && e.v.CanGoForward() }

func (e *wkEngine) stop() {
	if e.v != nil {
		e.v.Stop()
	}
}

func (e *wkEngine) setNavigationPolicy(f func(url string, mainFrame bool) bool) {
	e.policy = f
	if e.v != nil {
		e.v.Decide = f
	}
}

func (e *wkEngine) setCallHandler(f func(msg string) string) {
	e.call = f
	if e.v != nil {
		e.v.Call = f
	}
}
