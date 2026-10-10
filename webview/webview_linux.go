package webview

import (
	"bytes"
	"context"
	"errors"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/webkit"
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
	v       *webkit.View
	policy  func(url string, mainFrame bool) bool
	call    func(msg string) string

	// related is the opener of a view made by NewWindow.NewOn.
	related    *wkEngine
	newWin     func(n *NewWindow) *WebView
	show       func(WindowFeatures)
	closeH     func()
	status     func(text string)
	lastStatus string
	script     *bool
	// cancelPost stops the request of a POST in flight; postGen tells its answer it is too late.
	cancelPost context.CancelFunc
	postGen    int
}

func newEngine(w *WebView, parent *swt.Composite, opts Options) (engine, error) {
	if err := webkit.Load(); err != nil {
		return nil, err
	}
	host := opts.host
	if host == nil {
		host = swt.NewCompositeParentStyle(parent, swt.NONE)
	}
	e := &wkEngine{w: w, host: host, opts: opts, schemes: map[string]SchemeHandler{}}
	if opts.popup != nil {
		opener, ok := opts.popup.opener.(*wkEngine)
		if !ok || opener.v == nil {
			return nil, errors.New("webview: the opener of the new window is gone")
		}
		e.related = opener
	}
	e.host.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { e.resize() }))
	e.host.AddDisposeListener(disposeFunc(e.release))
	if e.related != nil {
		// The native view must exist before WebKit's create signal returns.
		e.start()
	}
	return e, nil
}

func (e *wkEngine) resize() {
	if e.v != nil {
		s := e.host.GetSize()
		e.v.SetSize(int(s.X), int(s.Y))
	}
}

// start creates the native view; the scheme list and scripts are frozen from here on.
func (e *wkEngine) start() *webkit.View {
	if e.v != nil {
		return e.v
	}
	cfg := webkit.Config{Inspectable: e.opts.Inspectable, Scripts: e.scripts}
	for s := range e.schemes {
		cfg.Schemes = append(cfg.Schemes, s)
	}
	// Load succeeded in newEngine, so New cannot fail here.
	var v *webkit.View
	if e.related != nil {
		v, _ = webkit.NewRelated(e.related.v, cfg)
	} else {
		v, _ = webkit.New(cfg)
	}
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
	if e.newWin != nil {
		v.NewWindow = e.createWindow
	}
	v.Show = func(f webkit.WindowFeatures) {
		if e.show != nil {
			e.show(WindowFeatures(f))
		}
	}
	v.Close = func() {
		if e.closeH != nil {
			e.closeH()
		}
	}
	v.Status = func(t string) {
		// WebKit reports every change of target; only the link under the pointer matters.
		if e.status != nil && t != e.lastStatus {
			e.lastStatus = t
			e.status(t)
		}
	}
	if e.script != nil {
		v.SetScriptEnabled(*e.script)
	}
	e.v = v
	v.Attach(uintptr(e.host.Handle))
	e.resize()
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
func (e *wkEngine) navigate(url string) {
	e.abortPost()
	e.start().LoadURL(url)
}

func (e *wkEngine) setHTML(html, base string) {
	e.abortPost()
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
	e.abortPost()
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

func (e *wkEngine) setNewWindowHandler(f func(n *NewWindow) *WebView) {
	e.newWin = f
	if e.v != nil {
		e.v.NewWindow = e.createWindow
	}
}

func (e *wkEngine) createWindow(url string) *webkit.View {
	n := &NewWindow{URL: url, opener: e}
	wv := e.newWin(n)
	if wv == nil {
		return nil
	}
	if pe, ok := wv.e.(*wkEngine); ok {
		return pe.v
	}
	return nil
}

func (e *wkEngine) setShowHandler(f func(WindowFeatures)) { e.show = f }
func (e *wkEngine) setCloseHandler(f func())              { e.closeH = f }
func (e *wkEngine) setStatusHandler(f func(string))       { e.status = f }

func (e *wkEngine) setScriptEnabled(on bool) {
	e.script = &on
	if e.v != nil {
		e.v.SetScriptEnabled(on)
	}
}

func (e *wkEngine) abortPost() {
	e.postGen++
	if e.cancelPost != nil {
		e.cancelPost()
		e.cancelPost = nil
	}
}

func (e *wkEngine) loadRequest(r LoadRequest) {
	e.abortPost()
	v := e.start()
	if r.Method == "" || strings.EqualFold(r.Method, "GET") {
		v.LoadRequest(r.URL, r.Headers)
		return
	}
	e.post(v, r)
}

// post sends the request with net/http, as WebKitGTK cannot, and shows the answer with the address of
// the final URL as its base. The answer comes back on the UI thread.
func (e *wkEngine) post(v *webkit.View, r LoadRequest) {
	u, err := url.Parse(r.URL)
	switch {
	case err != nil || u.Scheme == "":
		e.postFailed(v, r.URL, "URL is invalid")
		return
	case u.Scheme != "http" && u.Scheme != "https":
		e.postFailed(v, r.URL, "Unsupported connection type")
		return
	case u.Host == "":
		e.postFailed(v, r.URL, "URL is invalid")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	e.cancelPost = cancel
	display, gen := e.host.GetDisplay(), e.postGen
	back := func(f func()) {
		display.AsyncExec(jrt.NewRunnable(func() {
			if gen != e.postGen || e.host.IsDisposed() {
				return
			}
			f()
		}))
	}
	go func() {
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewReader(r.Body))
		if err != nil {
			back(func() { e.postFailed(v, r.URL, "URL is invalid") })
			return
		}
		for k, val := range r.Headers {
			req.Header.Set(k, val)
		}
		if req.Header.Get("Content-Type") == "" && len(r.Body) > 0 {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			msg := err.Error()
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				msg = "Unexpected end of file from server"
			}
			back(func() { e.postFailed(v, r.URL, msg) })
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			back(func() { e.postFailed(v, r.URL, err.Error()) })
			return
		}
		mt, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if mt == "" {
			mt, params, _ = mime.ParseMediaType(http.DetectContentType(body))
		}
		final := resp.Request.URL.String()
		back(func() { v.LoadBytes(body, mt, params["charset"], final) })
	}()
}

func (e *wkEngine) postFailed(v *webkit.View, url, msg string) {
	if e.w.OnNavigationFailed != nil {
		e.w.OnNavigationFailed(url, errors.New(msg))
	}
	v.LoadHTML("<html><body>"+html.EscapeString(msg)+"</body></html>", "")
}
