// The WebBrowser behind SWT's Browser: package webview is the engine, standing in for SWT's own
// WebKit/Edge/WebKitGTK classes (manual.txt, port.sh). The Java overrides it replaces are declared in
// tooling/j2go/stubs/org/eclipse/swt/browser/WebViewBrowser.java; here they are the unexported
// name_ methods of WebBrowserImpl.
//
// Known gaps against SWT's Browser: the Authentication listener never fires; the trusted flag of
// setText/evaluate is ignored; BrowserFunctions are defined in the main frame only. An OpenWindow
// listener that leaves event.browser unset refuses the window (SWT would make one), and while it runs
// the opener's page is blocked, so evaluate on the opener there only queues the script and returns null.
// A window opened by the page shares the opener's scripts and session (see webview.NewWindow).
package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
	"github.com/haiodo/gowt/webview"
)

// How long a synchronous call waits for the page before it gives up.
const evalTimeout = 5 * time.Second

type webViewBrowser struct {
	*WebBrowser
	wv *webview.WebView
	// policy: the engine asks decide before a navigation. Without it, changing fires once the
	// navigation has started and cannot cancel it.
	policy bool
	// html is what setText loaded: getText returns the source, as SWT's WebKit does, not the DOM.
	html string
	// inCall counts handleCall frames in progress: the page is blocked in window.gowt.call then.
	inCall int
	// inOpen counts OpenWindow listener runs: the page that called window.open is blocked then.
	inOpen int
	// popup: this Browser takes a window opened by a page; shown: its VisibilityWindow Show has fired.
	popup, shown bool
	// navigated: a load was asked for; blank: 1 while ensurePage loads the page evaluate needs, 2 when it is done.
	navigated bool
	blank     int
	// scriptsOff: the page shown was loaded with JavaScript switched off (Browser.setJavascriptEnabled).
	scriptsOff bool
}

// pendingPopup is the window request the OpenWindow listeners are answering: the first Browser created
// meanwhile (webViewBrowser.open) is the one that takes the window.
var pendingPopup *webview.NewWindow

// BrowserFactory hands Browser its engine (SWT's per-OS class of this name).
type BrowserFactory struct{}

func (BrowserFactory) createWebBrowser(style int32) *WebBrowser {
	w := &webViewBrowser{WebBrowser: newWebBrowser()}
	w.WebBrowser.impl = w
	return w.WebBrowser
}

// CreateWebBrowser is Browser.createWebBrowser (manual.txt): there is no Edge/IE fallback dialog.
func (this *Browser) CreateWebBrowser(parent *swt.Composite, style int32) bool {
	wb := BrowserFactory{}.createWebBrowser(style)
	wb.SetBrowser(this)
	this.webBrowser = wb
	return wb.impl.(*webViewBrowser).open() == nil
}

func (w *webViewBrowser) open() error {
	var wv *webview.WebView
	var err error
	if pendingPopup != nil {
		if wv, err = pendingPopup.NewOn(w.browser.Composite, webview.Options{}); err == nil {
			w.popup = true
		}
	}
	if wv == nil {
		if wv, err = webview.NewOn(w.browser.Composite, webview.Options{}); err != nil {
			return err
		}
	}
	w.wv = wv
	wv.SetNewWindowHandler(w.newWindow)
	wv.SetShowHandler(w.windowShown)
	wv.SetCloseHandler(w.windowClosed)
	wv.SetStatusHandler(w.statusChanged)
	wv.OnNavigationStarted = w.navigationStarted
	wv.OnNavigationFinished = w.navigationFinished
	wv.OnTitleChanged = w.titleChanged
	w.policy = wv.SetNavigationPolicy(w.decide)
	wv.SetCallHandler(w.handleCall)
	wv.AddScript(preamble)
	return nil
}

func (w *webViewBrowser) create_(parent *swt.Composite, style int32) {
	if err := w.open(); err != nil {
		panic(swt.NewSWTErrorCodeMessage(swt.ERROR_NO_HANDLES, err.Error()))
	}
}

func (w *webViewBrowser) getBrowserType_() string {
	if runtime.GOOS == "windows" {
		return "edge"
	}
	return "webkit"
}

// ---------------------------------------------------------------- navigation

func (w *webViewBrowser) setUrl_(url string, postData string, headers []string) bool {
	w.html = ""
	w.navigated = true
	w.applyScripts()
	u := escapeURL(normalizeURL(url))
	hd := parseHeaders(headers)
	if postData == "" && len(hd) == 0 {
		w.wv.Navigate(u)
		return true
	}
	r := webview.LoadRequest{URL: u, Headers: hd}
	if postData != "" {
		r.Method, r.Body = "POST", []byte(postData)
	}
	if !w.wv.Load(r) {
		w.wv.Navigate(u)
	}
	return true
}

// parseHeaders reads SWT's "Name: value" strings; one without a colon is dropped, as SWT does.
func parseHeaders(headers []string) map[string]string {
	var out map[string]string
	for _, h := range headers {
		name, value, ok := strings.Cut(h, ":")
		if name = strings.TrimSpace(name); !ok || name == "" {
			continue
		}
		if out == nil {
			out = map[string]string{}
		}
		out[name] = strings.TrimSpace(value)
	}
	return out
}

// applyScripts hands the engine the JavaScript switch of Browser.setJavascriptEnabled for the page about to load.
func (w *webViewBrowser) applyScripts() {
	on := w.browser.webBrowser.jsEnabledOnNextPage
	if on == !w.scriptsOff {
		return
	}
	if w.wv.SetScriptEnabled(on) {
		w.scriptsOff = !on
	}
}

func (w *webViewBrowser) setText_(html string, trusted bool) bool {
	w.html = html
	w.navigated = true
	w.applyScripts()
	w.wv.SetHTML(html, "")
	return true
}

func (w *webViewBrowser) getUrl_() string {
	if u := w.wv.URL(); u != "" {
		return u
	}
	return "about:blank"
}

func (w *webViewBrowser) getText_() string {
	if w.html != "" {
		return w.html
	}
	if w.wv.URL() == "" {
		return ""
	}
	res, err, ok := w.evalSync(`(document.doctype ? new XMLSerializer().serializeToString(document.doctype) : "") + document.documentElement.outerHTML`)
	if !ok || err != nil {
		return ""
	}
	var s string
	if json.Unmarshal([]byte(res), &s) != nil {
		return ""
	}
	return s
}

func (w *webViewBrowser) back_() bool {
	if can, known := w.wv.CanGoBack(); known && !can {
		return false
	}
	w.wv.GoBack()
	return true
}

func (w *webViewBrowser) forward_() bool {
	if can, known := w.wv.CanGoForward(); known && !can {
		return false
	}
	w.wv.GoForward()
	return true
}

func (w *webViewBrowser) isBackEnabled_() bool {
	can, known := w.wv.CanGoBack()
	return can || !known
}

func (w *webViewBrowser) isForwardEnabled_() bool {
	can, known := w.wv.CanGoForward()
	return can || !known
}

func (w *webViewBrowser) refresh_() { w.wv.Reload() }
func (w *webViewBrowser) stop_()    { w.wv.Stop() }

func (w *webViewBrowser) isFocusControl_() bool { return false }

// SWT's WebKit class: a path becomes a file URL, a bare host or host:port an http one.
func normalizeURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return "file://" + url
	}
	scheme, rest, ok := strings.Cut(url, ":")
	if !ok || hostPort(scheme, rest) {
		return "http://" + url
	}
	return url
}

// hostPort tells "localhost:8080/x" from "mailto:a@b": a port is digits up to a path, query, fragment or the end.
func hostPort(scheme, rest string) bool {
	switch strings.ToLower(scheme) {
	case "javascript", "data", "about", "tel", "sms", "mailto":
		return false
	}
	i := strings.IndexAny(rest, "/?#")
	if i < 0 {
		i = len(rest)
	}
	_, err := strconv.ParseUint(rest[:i], 10, 16)
	return i > 0 && err == nil
}

// escapeURL percent-escapes what a URL may not hold, keeping existing %XX and '#'.
func escapeURL(u string) string {
	var sb strings.Builder
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c <= ' ' || c >= 0x7f || strings.IndexByte(`"<>\^`+"`"+`{|}`, c) >= 0 {
			fmt.Fprintf(&sb, "%%%02X", c)
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------- events

func (w *webViewBrowser) newLocationEvent(location string, top bool) *LocationEvent {
	e := NewLocationEvent(w.browser)
	e.Display = w.browser.GetDisplay()
	e.Widget = w.browser.AsWidget()
	e.Location = location
	e.Top = top
	e.Doit = true
	return e
}

// decide is the engine's navigation policy: LocationListener.changing, where doit=false cancels.
func (w *webViewBrowser) decide(url string, mainFrame bool) bool {
	if w.browser.IsDisposed() || w.blank == 1 {
		return true
	}
	e := w.newLocationEvent(url, mainFrame)
	for _, l := range w.locationListeners {
		l.Changing(e)
		if w.browser.IsDisposed() {
			return true
		}
	}
	if e.Doit && mainFrame && url != "about:blank" {
		w.html = ""
	}
	return e.Doit
}

func (w *webViewBrowser) navigationStarted(url string) {
	if w.blank == 1 {
		return
	}
	w.ensureShown()
	if !w.policy {
		w.decide(url, true)
	}
	w.fireProgress(false, 0)
}

func (w *webViewBrowser) navigationFinished(url string) {
	if w.browser.IsDisposed() {
		return
	}
	if w.blank == 1 {
		w.blank = 2
		return
	}
	w.ensureShown()
	e := w.newLocationEvent(url, true)
	for _, l := range w.locationListeners {
		l.Changed(e)
		if w.browser.IsDisposed() {
			return
		}
	}
	w.fireProgress(true, 100)
}

func (w *webViewBrowser) fireProgress(completed bool, current int32) {
	if w.browser.IsDisposed() {
		return
	}
	for _, l := range w.progressListeners {
		e := NewProgressEvent(w.browser)
		e.Display = w.browser.GetDisplay()
		e.Widget = w.browser.AsWidget()
		e.Current = current
		e.Total = 100
		if completed {
			l.Completed(e)
		} else {
			l.Changed(e)
		}
		if w.browser.IsDisposed() {
			return
		}
	}
}

func (w *webViewBrowser) titleChanged(title string) {
	if title == "" || w.browser.IsDisposed() || w.blank == 1 {
		return
	}
	for _, l := range w.titleListeners {
		e := NewTitleEvent(w.browser)
		e.Display = w.browser.GetDisplay()
		e.Widget = w.browser.AsWidget()
		e.Title = title
		l.Changed(e)
		if w.browser.IsDisposed() {
			return
		}
	}
}

// ---------------------------------------------------------------- windows

func (w *webViewBrowser) newWindowEvent() *WindowEvent {
	e := NewWindowEvent(w.browser)
	e.Display = w.browser.GetDisplay()
	e.Widget = w.browser.AsWidget()
	return e
}

// newWindow answers window.open: the Browser the OpenWindow listeners set in event.browser takes the page.
func (w *webViewBrowser) newWindow(n *webview.NewWindow) *webview.WebView {
	if w.browser.IsDisposed() || len(w.openWindowListeners) == 0 {
		return nil
	}
	e := w.newWindowEvent()
	prev := pendingPopup
	pendingPopup = n
	w.inOpen++
	defer func() { pendingPopup = prev; w.inOpen-- }()
	for _, l := range w.openWindowListeners {
		l.Open(e)
		if w.browser.IsDisposed() {
			return nil
		}
	}
	if e.Browser == nil || e.Browser.IsDisposed() {
		return nil
	}
	if child, ok := e.Browser.webBrowser.impl.(*webViewBrowser); ok && child.popup {
		return child.wv
	}
	return nil
}

// windowShown is the page's wish to see its window: VisibilityWindow Show, once, with the features of window.open.
func (w *webViewBrowser) windowShown(f webview.WindowFeatures) {
	if w.shown || w.browser.IsDisposed() {
		return
	}
	w.shown = true
	e := w.newWindowEvent()
	if f.X != 0 || f.Y != 0 {
		e.Location = swt.NewPoint(int32(f.X), int32(f.Y))
	}
	if f.Width != 0 || f.Height != 0 {
		e.Size = swt.NewPoint(int32(f.Width), int32(f.Height))
	}
	e.AddressBar, e.MenuBar, e.StatusBar, e.ToolBar = f.LocationBar, f.MenuBar, f.StatusBar, f.ToolBar
	for _, l := range w.visibilityWindowListeners {
		l.Show(e)
		if w.browser.IsDisposed() {
			return
		}
	}
}

// ensureShown puts Show before the first location or progress event of a window opened by a page,
// for an engine that has not said the window is ready by then.
func (w *webViewBrowser) ensureShown() {
	if w.popup && !w.shown {
		w.windowShown(webview.WindowFeatures{})
	}
}

func (w *webViewBrowser) windowClosed() {
	if w.browser.IsDisposed() {
		return
	}
	e := w.newWindowEvent()
	for _, l := range w.closeWindowListeners {
		l.Close(e)
		if w.browser.IsDisposed() {
			return
		}
	}
	// SWT closes the Browser once its listeners have seen the request.
	w.browser.Dispose()
}

func (w *webViewBrowser) statusChanged(text string) {
	if w.browser.IsDisposed() {
		return
	}
	for _, l := range w.statusTextListeners {
		e := NewStatusTextEvent(w.browser)
		e.Display = w.browser.GetDisplay()
		e.Widget = w.browser.AsWidget()
		e.Text = text
		l.Changed(e)
		if w.browser.IsDisposed() {
			return
		}
	}
}

// ---------------------------------------------------------------- cookies

// The cookie statics are the process's, not a Browser's: they wait for the store on the current UI thread.
func init() {
	WebBrowserNativeGetCookie = jrt.NewRunnable(func() {
		name, url := WebBrowserCookieName, WebBrowserCookieUrl
		value, done := "", false
		webview.Cookies(url, func(cs []webview.Cookie, ok bool) {
			for _, c := range cs {
				if c.Name == name {
					value = c.Value
					break
				}
			}
			done = true
		})
		pumpCurrent(&done)
		WebBrowserCookieValue = value
	})
	WebBrowserNativeSetCookie = jrt.NewRunnable(func() {
		ok, done := false, false
		webview.SetCookie(WebBrowserCookieUrl, WebBrowserCookieValue, func(r bool) { ok, done = r, true })
		pumpCurrent(&done)
		WebBrowserCookieResult = ok
	})
	WebBrowserNativeClearSessions = jrt.NewRunnable(func() {
		done := false
		webview.ClearSessionCookies(func() { done = true })
		pumpCurrent(&done)
	})
}

// pumpCurrent runs the event loop of the current thread's Display until *done or the timeout.
func pumpCurrent(done *bool) {
	if d := swt.DisplayGetCurrent(); d != nil {
		pumpDisplay(d, func() bool { return *done }, func() bool { return false })
	}
}

// ---------------------------------------------------------------- script

// pump runs the UI event loop until done, which is how a synchronous SWT call waits for the
// engine's asynchronous answer. It reports false on timeout or when the Browser is disposed.
func (w *webViewBrowser) pump(done func() bool) bool {
	return pumpDisplay(w.browser.GetDisplay(), done, w.browser.IsDisposed)
}

func pumpDisplay(d *swt.Display, done, abort func() bool) bool {
	deadline := time.Now().Add(evalTimeout)
	// A timer ends Sleep when no event would, so the deadline is checked.
	tick := jrt.NewRunnable(func() {})
	for !done() {
		if abort() || time.Now().After(deadline) {
			return false
		}
		if !d.ReadAndDispatch() {
			d.TimerExec(20, tick)
			d.Sleep()
		}
	}
	return true
}

// evalSync evaluates js (see webview.WebView.Eval for the result) and waits for it.
func (w *webViewBrowser) evalSync(js string) (result string, err error, ok bool) {
	if w.inCall > 0 {
		// The page waits for this call, so the script would only run after it returned.
		panic(swt.NewSWTExceptionCodeMessage(swt.ERROR_FAILED_EVALUATE, "evaluate, execute and getText cannot be used inside a BrowserFunction"))
	}
	if w.inOpen > 0 {
		return "", errors.New("the page is blocked in window.open"), false
	}
	w.ensurePage()
	var finished bool
	w.wv.Eval(js, func(r string, e error) { result, err, finished = r, e, true })
	if !w.pump(func() bool { return finished }) {
		if w.browser.IsDisposed() {
			swt.Error(swt.ERROR_WIDGET_DISPOSED)
		}
		return "", errors.New("timeout"), false
	}
	return result, err, true
}

// ensurePage loads an empty page into a Browser nothing was loaded into, as SWT's engines show about:blank
// from the start: the scripts of the BrowserFunctions run at the start of a page. No listener hears it.
func (w *webViewBrowser) ensurePage() {
	if w.navigated {
		return
	}
	w.navigated = true
	w.blank = 1
	w.wv.SetHTML("", "")
	w.pump(func() bool { return w.blank == 2 })
	w.blank = 0
}

func (w *webViewBrowser) execute_(script string) bool {
	if w.scriptsOff {
		return false
	}
	if w.inOpen > 0 {
		w.wv.Eval(executeWrapper(script), func(string, error) {})
		return true
	}
	res, err, ok := w.evalSync(executeWrapper(script))
	return ok && err == nil && res == "true"
}

// executeWrapper runs script as a global script and answers true unless it threw; its value is not
// serialized, so one that JSON cannot hold (a DOM node) does not count as a failure.
func executeWrapper(script string) string {
	q, _ := json.Marshal(script)
	return `(function(){try{(0,eval)(` + string(q) + `);return true}catch(e){return false}})()`
}

// nonBlockingExecute does not wait: the engine runs scripts in the order it got them.
func (w *webViewBrowser) nonBlockingExecute_(script string) {
	if w.wv.URL() == "" {
		return // no page yet; the init script of the first one defines the functions
	}
	w.wv.Eval(script, func(string, error) {})
}

func (w *webViewBrowser) evaluate_(script string, trusted bool) any {
	return w.evaluateScript_(script)
}

func (w *webViewBrowser) evaluateScript_(script string) any {
	if w.scriptsOff {
		return nil
	}
	if w.inOpen > 0 {
		// The page that called window.open waits for the listener, so the script can only run after it.
		w.wv.Eval(evaluateWrapper(script), func(string, error) {})
		return nil
	}
	res, err, ok := w.evalSync(evaluateWrapper(script))
	if !ok {
		panic(swt.NewSWTExceptionCodeMessage(swt.ERROR_FAILED_EVALUATE, "the page did not answer"))
	}
	if err != nil {
		panic(swt.NewSWTExceptionCodeMessage(swt.ERROR_FAILED_EVALUATE, err.Error()))
	}
	v, code, msg := decodeEvaluate(res)
	if code != 0 {
		panic(swt.NewSWTExceptionCodeMessage(code, msg))
	}
	return v
}

const invalidMark = "\x01invalid"

// evaluateWrapper turns an SWT script (the body of a function) into one expression whose JSON is
// {"v":value} or {"e":message}; only null, string, number, boolean and arrays of those may return.
func evaluateWrapper(script string) string {
	q, _ := json.Marshal(script)
	return `(function(){function c(v){var t=typeof v;if(v===null||v===undefined)return null;` +
		`if(t==="string"||t==="number"||t==="boolean")return v;if(Array.isArray(v))return v.map(c);throw new Error("` + invalidMark + `")}` +
		`try{return{v:c((new Function(` + string(q) + `))())}}catch(e){return{e:String(e&&e.message!==undefined?e.message:e)}}})()`
}

// decodeEvaluate maps the JSON of evaluateWrapper to a Java-style value: Double, String, Boolean,
// Object[] or nil. A non-zero code is the SWTException to throw.
func decodeEvaluate(res string) (v any, code int32, msg string) {
	var r struct {
		V json.RawMessage `json:"v"`
		E *string         `json:"e"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		return nil, swt.ERROR_FAILED_EVALUATE, "bad result: " + res
	}
	if r.E != nil {
		if *r.E == invalidMark {
			return nil, swt.ERROR_INVALID_RETURN_VALUE, ""
		}
		return nil, swt.ERROR_FAILED_EVALUATE, *r.E
	}
	var x any
	if err := json.Unmarshal(r.V, &x); err != nil {
		return nil, swt.ERROR_FAILED_EVALUATE, "bad result: " + res
	}
	return javaValue(x), 0, ""
}

// javaValue is what SWT hands out for a JSON value: arrays become []any.
func javaValue(x any) any {
	if a, ok := x.([]any); ok {
		out := make([]any, len(a))
		for i, e := range a {
			out[i] = javaValue(e)
		}
		return out
	}
	return x
}

// ---------------------------------------------------------------- BrowserFunction

// preamble runs at the start of every page: it defines callJava and asks Go for the scripts of the
// functions registered by now, so they exist before the page's own scripts.
const preamble = `(function(){var g=window.gowt;if(!g||!g.call||window.top!==window)return;` +
	`window.callJava=function callJava(i,t,a){var r=g.call(JSON.stringify([i,t,a]));return r==null?undefined:JSON.parse(r)};` +
	`var s=g.call('["init"]');if(s)(0,eval)(s)})();`

func (w *webViewBrowser) getJavaCallDeclaration_() string {
	return "if (!window.callJava) {\n" +
		"	window.callJava = function callJava(index, token, args) {\n" +
		"		if (!window.gowt || !window.gowt.call) return undefined;\n" +
		"		var r = window.gowt.call(JSON.stringify([index, token, args]));\n" +
		"		return r == null ? undefined : JSON.parse(r);\n" +
		"	}\n" +
		"};\n"
}

// handleCall answers window.gowt.call: ["init"] for the preamble, [index, token, args] for a function.
func (w *webViewBrowser) handleCall(msg string) string {
	var req []json.RawMessage
	if json.Unmarshal([]byte(msg), &req) != nil || len(req) == 0 {
		return "null"
	}
	var first any
	_ = json.Unmarshal(req[0], &first)
	if s, ok := first.(string); ok && s == "init" {
		return w.initScript()
	}
	if len(req) != 3 {
		return "null"
	}
	var index float64
	var token string
	var args []any
	if json.Unmarshal(req[0], &index) != nil || json.Unmarshal(req[1], &token) != nil || json.Unmarshal(req[2], &args) != nil {
		return "null"
	}
	f, _ := w.functions.Get(int32(index)).(*BrowserFunction)
	if f == nil || f.token != token {
		return "null"
	}
	w.inCall++
	defer func() { w.inCall-- }()
	return callFunction(f, javaValue(args).([]any))
}

// initScript joins the registered functions' scripts, the ones createFunction runs in the live page.
func (w *webViewBrowser) initScript() string {
	var fs []*BrowserFunction
	for _, v := range w.functions.Values().ToArray() {
		if f, ok := v.(*BrowserFunction); ok && f.functionString != "" {
			fs = append(fs, f)
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].index < fs[j].index })
	var sb strings.Builder
	for _, f := range fs {
		sb.WriteString(f.functionString)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// callFunction runs f and returns its result as JSON for the page; an error, a panic or a value
// the page cannot take comes back as the error string the page-side wrapper throws.
func callFunction(f *BrowserFunction, args []any) (reply string) {
	defer func() {
		if r := recover(); r != nil {
			reply = jsonString(WebBrowserCreateErrorString(fmt.Sprint(r)))
		}
	}()
	out, ok := encodeJavaValue(f.Function(args))
	if !ok {
		return jsonString(WebBrowserCreateErrorString("Unsupported return type"))
	}
	return out
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// encodeJavaValue is the JSON of a value a BrowserFunction may return: nil, string, bool, a number
// or []any of those.
func encodeJavaValue(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "null", true
	case string, bool, int, int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64:
		b, err := json.Marshal(x)
		if err != nil { // NaN, Inf
			return "null", true
		}
		return string(b), true
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			s, ok := encodeJavaValue(e)
			if !ok {
				return "", false
			}
			parts[i] = s
		}
		return "[" + strings.Join(parts, ",") + "]", true
	}
	return "", false
}
