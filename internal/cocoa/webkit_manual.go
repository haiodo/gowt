//go:build darwin

// WKWebView binding for the webview package, hand-written over purego. One runtime ObjC class
// (gowtWKHandler) is the script-message handler, URL-scheme handler, navigation delegate and title
// observer of every view; its callbacks find the Go side by the instance address. All calls and
// callbacks run on the UI (main) thread.
package cocoa

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/ebitengine/purego"
)

// WKConfig is what WKWebViewConfiguration can no longer take once the view exists.
type WKConfig struct {
	Parent      uintptr // NSView the WKWebView is added to
	Frame       NSRect
	Inspectable bool
	Scripts     []string // run at document start, before page scripts
	Schemes     []string // custom URL schemes served through WKView.Scheme
	// Config is the WKWebViewConfiguration WebKit passed to the create delegate for a window the page
	// opens; the view must be made with it. It gets a user content controller of its own, so Scripts
	// apply, and keeps the URL scheme handlers of the opener (Schemes is ignored).
	Config uintptr
}

// WKFeatures are the window.open features; X, Y, Width and Height are 0 when the page gave none.
type WKFeatures struct {
	X, Y, Width, Height         int
	MenuBar, StatusBar, ToolBar bool
}

type WKView struct {
	View uintptr // the WKWebView

	Message      func(body string)
	NavStarted   func(url string)
	NavFinished  func(url string)
	NavFailed    func(url, err string)
	TitleChanged func(title string)
	// Scheme answers a request on one of WKConfig.Schemes; Content-Type goes in headers.
	Scheme func(url, method string) (status int, headers map[string]string, body []byte)
	// Decide is asked before every navigation; false cancels it. Nil allows all.
	Decide func(url string, mainFrame bool) bool
	// Call answers window.gowt.call(msg) from the page synchronously: the page blocks in window.prompt
	// until the reply. Without it the call returns null.
	Call func(msg string) string
	// NewWindow is asked when the page opens a window (window.open, target=_blank): it makes the view
	// with WKConfig.Config = config and returns it; nil refuses the window.
	NewWindow func(url string, config uintptr, f WKFeatures) *WKView
	// Close fires when the page asks to close its window (window.close).
	Close func()
	// Status gets the link under the pointer, "" when it leaves it (EnableStatus).
	Status func(text string)
	// Features of the window.open that made this view; zero for any other.
	Features WKFeatures

	// The last navigation started: WebKit commits about:blank for a refused connection and reports no
	// failure, so a commit of about:blank for another URL is the failure (see didCommit).
	navPtr    uintptr
	navURL    string
	navFailed bool

	config, ucc, obj uintptr
	disposed         bool
	scriptOff        bool
	statusOn         bool
}

var wkTrace = os.Getenv("GOWT_WK_TRACE") != ""

var (
	wkOnce  sync.Once
	wkClass uintptr
	wkViews = map[uintptr]*WKView{}
	wkInit  func(self, sel uintptr, frame NSRect, config uintptr) uintptr
)

// The page-side half of window.gowt.postMessage.
const wkBridge = `window.gowt={postMessage:function(m){window.webkit.messageHandlers.gowt.postMessage(typeof m==="string"?m:JSON.stringify(m))}};`

// wkCallShim is window.gowt.call, injected in the main frame only; a subframe has no call.
const wkCallShim = `window.gowt.call=function(m){return window.prompt(typeof m==="string"?m:JSON.stringify(m),"\u0001gowt")};`

// wkCallMark is the prompt's default text that tells window.gowt.call from a page's own prompt().
const wkCallMark = "\x01gowt"

func wkSetup() {
	wkOnce.Do(func() {
		objcInit()
		if _, err := purego.Dlopen("/System/Library/Frameworks/WebKit.framework/WebKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
			panic("gowt/internal/cocoa: dlopen WebKit: " + err.Error())
		}
		purego.RegisterFunc(&wkInit, objcMsgSend)
		trace := wkTrace
		view := func(self uintptr) *WKView { return wkViews[self] }
		urlOf := func(wv uintptr) string { return goString(msg(msg(wv, "URL"), "absoluteString")) }
		// A failed provisional load leaves wv.URL at the previous page; the error carries the requested one.
		failedURL := func(wv, e uintptr) string {
			if u := goString(msg(msg(e, "userInfo"), "objectForKey:", nsString("NSErrorFailingURLStringKey"))); u != "" {
				return u
			}
			return urlOf(wv)
		}
		wkClass = newClass("gowtWKHandler", []string{"WKScriptMessageHandler", "WKURLSchemeHandler", "WKNavigationDelegate", "WKUIDelegate"}, map[string]objcMethod{
			"userContentController:didReceiveScriptMessage:": {purego.NewCallback(func(self, _, _, m uintptr) {
				defer enterCallback()()
				v := view(self)
				if v == nil {
					return
				}
				if goString(msg(m, "name")) == "gowtstatus" {
					if v.Status != nil {
						v.Status(goString(msg(m, "body")))
					}
				} else if v.Message != nil {
					v.Message(goString(msg(m, "body")))
				}
			}), "v@:@@"},
			"webView:didStartProvisionalNavigation:": {purego.NewCallback(func(self, _, wv, nav uintptr) {
				defer enterCallback()()
				if trace {
					fmt.Fprintf(os.Stderr, "wk: didStart %s nav=%#x\n", urlOf(wv), nav)
				}
				if v := view(self); v != nil {
					v.navPtr, v.navURL, v.navFailed = nav, urlOf(wv), false
					if v.NavStarted != nil {
						v.NavStarted(v.navURL)
					}
				}
			}), "v@:@@"},
			"webView:didFinishNavigation:": {purego.NewCallback(func(self, _, wv, nav uintptr) {
				defer enterCallback()()
				if trace {
					fmt.Fprintln(os.Stderr, "wk: didFinish", urlOf(wv))
				}
				if v := view(self); v != nil && v.NavFinished != nil && !(nav == v.navPtr && v.navFailed) {
					v.NavFinished(urlOf(wv))
				}
			}), "v@:@@"},
			"webView:didCommitNavigation:": {purego.NewCallback(func(self, _, wv, nav uintptr) {
				defer enterCallback()()
				if trace {
					fmt.Fprintln(os.Stderr, "wk: didCommit", urlOf(wv))
				}
				v := view(self)
				if v != nil && nav == v.navPtr && v.navURL != "about:blank" && urlOf(wv) == "about:blank" {
					v.navFailed = true
					if v.NavFailed != nil {
						v.NavFailed(v.navURL, "load failed: the server could not be reached")
					}
				}
			}), "v@:@@"},
			"webView:didFailNavigation:withError:": {purego.NewCallback(func(self, _, wv, _, e uintptr) {
				defer enterCallback()()
				if trace {
					fmt.Fprintln(os.Stderr, "wk: didFail", failedURL(wv, e), goString(msg(e, "description")))
				}
				if v := view(self); v != nil && v.NavFailed != nil {
					v.NavFailed(failedURL(wv, e), goString(msg(e, "localizedDescription")))
				}
			}), "v@:@@@"},
			"webView:didFailProvisionalNavigation:withError:": {purego.NewCallback(func(self, _, wv, _, e uintptr) {
				defer enterCallback()()
				if trace {
					fmt.Fprintln(os.Stderr, "wk: didFailProvisional", failedURL(wv, e), goString(msg(e, "description")))
				}
				if v := view(self); v != nil && v.NavFailed != nil {
					v.NavFailed(failedURL(wv, e), goString(msg(e, "localizedDescription")))
				}
			}), "v@:@@@"},
			// The variant with preferences: allowsContentJavaScript is set per navigation, which is how
			// setScriptEnabled reaches the pages loaded after it.
			"webView:decidePolicyForNavigationAction:preferences:decisionHandler:": {purego.NewCallback(func(self, _, _, action, prefs, handler uintptr) {
				defer enterCallback()()
				allow := true
				if trace {
					fmt.Fprintln(os.Stderr, "wk: decidePolicy", goString(msg(msg(msg(action, "request"), "URL"), "absoluteString")))
				}
				// A panic in a listener must not skip the handler: WebKit waits for it exactly once.
				func() {
					defer func() { _ = recover() }()
					if v := view(self); v != nil && v.Decide != nil {
						url := goString(msg(msg(msg(action, "request"), "URL"), "absoluteString"))
						// targetFrame is nil for a link that opens a new window.
						frame := msg(action, "targetFrame")
						allow = v.Decide(url, frame == 0 || msg(frame, "isMainFrame")&0xff != 0)
					}
				}()
				if v := view(self); v != nil && msg(prefs, "respondsToSelector:", sel("setAllowsContentJavaScript:"))&0xff != 0 {
					on := uintptr(1)
					if v.scriptOff {
						on = 0
					}
					msg(prefs, "setAllowsContentJavaScript:", on)
				}
				policy := uintptr(0) // WKNavigationActionPolicyCancel
				if allow {
					policy = 1 // WKNavigationActionPolicyAllow
				}
				callBlock(handler, policy, prefs)
			}), "v@:@@@@?"},
			"webView:createWebViewWithConfiguration:forNavigationAction:windowFeatures:": {purego.NewCallback(func(self, _, _, config, action, features uintptr) uintptr {
				defer enterCallback()()
				v := view(self)
				if v == nil || v.NewWindow == nil {
					return 0
				}
				var nv *WKView
				// A panic in a listener must not unwind through WebKit: the window is refused.
				func() {
					defer func() { _ = recover() }()
					f := wkFeatures(features)
					nv = v.NewWindow(goString(msg(msg(msg(action, "request"), "URL"), "absoluteString")), config, f)
					if nv != nil {
						nv.Features = f
					}
				}()
				if nv == nil {
					return 0
				}
				return nv.View
			}), "@@:@@@@"},
			"webViewDidClose:": {purego.NewCallback(func(self, _, _ uintptr) {
				defer enterCallback()()
				if v := view(self); v != nil && v.Close != nil {
					v.Close()
				}
			}), "v@:@"},
			"webView:runJavaScriptTextInputPanelWithPrompt:defaultText:initiatedByFrame:completionHandler:": {purego.NewCallback(func(self, _, _, prompt, def, frame, handler uintptr) {
				defer enterCallback()()
				reply := uintptr(0)
				// Only the main frame may call Go: a subframe (possibly cross-origin) has its own window.gowt.
				if v := view(self); v != nil && v.Call != nil && goString(def) == wkCallMark && msg(frame, "isMainFrame")&0xff != 0 {
					func() {
						defer func() { _ = recover() }()
						reply = nsString(v.Call(goString(prompt)))
					}()
				}
				callBlock(handler, reply)
			}), "v@:@@@@@?"},
			"observeValueForKeyPath:ofObject:change:context:": {purego.NewCallback(func(self, _, _, wv, _, _ uintptr) {
				defer enterCallback()()
				if v := view(self); v != nil && v.TitleChanged != nil {
					v.TitleChanged(goString(msg(wv, "title")))
				}
			}), "v@:@@@^v"},
			"webView:startURLSchemeTask:": {purego.NewCallback(func(self, _, _, task uintptr) {
				if v := view(self); v != nil {
					v.serve(task)
				}
			}), "v@:@@"},
			// Requests are answered synchronously in start, so there is nothing to cancel.
			"webView:stopURLSchemeTask:": {purego.NewCallback(func(self, _, _, _ uintptr) {}), "v@:@@"},
		})
	})
}

// NewWKView creates the web view, adds it to cfg.Parent and returns it with no handlers set.
func NewWKView(cfg WKConfig) *WKView {
	wkSetup()
	v := &WKView{}
	v.obj = msg(msg(wkClass, "alloc"), "init")
	wkViews[v.obj] = v
	if cfg.Config != 0 {
		v.config = msg(cfg.Config, "retain")
		v.ucc = msg(msg(class("WKUserContentController"), "alloc"), "init")
		msg(v.config, "setUserContentController:", v.ucc)
		msg(v.ucc, "release") // the configuration holds it
		cfg.Schemes = nil
	} else {
		v.config = msg(msg(class("WKWebViewConfiguration"), "alloc"), "init")
		v.ucc = msg(v.config, "userContentController")
	}
	// Without it window.open outside a user gesture is dropped.
	msg(msg(v.config, "preferences"), "setJavaScriptCanOpenWindowsAutomatically:", 1)
	msg(v.ucc, "addScriptMessageHandler:name:", v.obj, nsString("gowt"))
	v.AddScript(wkBridge)
	v.addScript(wkCallShim, true)
	for _, s := range cfg.Scripts {
		v.AddScript(s)
	}
	for _, s := range cfg.Schemes {
		msg(v.config, "setURLSchemeHandler:forURLScheme:", v.obj, nsString(s))
	}
	v.View = wkInit(msg(class("WKWebView"), "alloc"), sel("initWithFrame:configuration:"), cfg.Frame, v.config)
	msg(v.View, "setNavigationDelegate:", v.obj)
	msg(v.View, "setUIDelegate:", v.obj)
	msg(v.View, "addObserver:forKeyPath:options:context:", v.obj, nsString("title"), 0, 0)
	if cfg.Inspectable && msg(v.View, "respondsToSelector:", sel("setInspectable:"))&0xff != 0 {
		msg(v.View, "setInspectable:", 1)
	}
	msg(cfg.Parent, "addSubview:", v.View)
	return v
}

// AddScript runs js at document start in pages loaded from now on.
func (v *WKView) AddScript(js string) { v.addScript(js, false) }

func (v *WKView) addScript(js string, mainFrameOnly bool) {
	main := uintptr(0)
	if mainFrameOnly {
		main = 1
	}
	us := msg(msg(class("WKUserScript"), "alloc"), "initWithSource:injectionTime:forMainFrameOnly:", nsString(js), 0, main)
	msg(v.ucc, "addUserScript:", us)
	msg(us, "release")
}

func (v *WKView) SetFrame(r NSRect) { msgRectOnly(v.View, sel("setFrame:"), r) }

func (v *WKView) GoBack()    { msg(v.View, "goBack") }
func (v *WKView) GoForward() { msg(v.View, "goForward") }
func (v *WKView) Reload()    { msg(v.View, "reload") }
func (v *WKView) Stop()      { msg(v.View, "stopLoading") }

func (v *WKView) CanGoBack() bool    { return msg(v.View, "canGoBack")&0xff != 0 }
func (v *WKView) CanGoForward() bool { return msg(v.View, "canGoForward")&0xff != 0 }

func (v *WKView) URL() string { return goString(msg(msg(v.View, "URL"), "absoluteString")) }

func (v *WKView) LoadURL(url string) {
	req := msg(class("NSURLRequest"), "requestWithURL:", msg(class("NSURL"), "URLWithString:", nsString(url)))
	nav := msg(v.View, "loadRequest:", req)
	if wkTrace {
		fmt.Fprintf(os.Stderr, "wk: loadRequest %s url=%#x nav=%#x\n", url, msg(req, "URL"), nav)
	}
}

// LoadRequest loads url with method ("" is GET), extra request headers and a body.
func (v *WKView) LoadRequest(url, method string, headers map[string]string, body []byte) {
	req := msg(msg(class("NSMutableURLRequest"), "alloc"), "initWithURL:", msg(class("NSURL"), "URLWithString:", nsString(url)))
	if method != "" {
		msg(req, "setHTTPMethod:", nsString(method))
	}
	if body != nil {
		msg(req, "setHTTPBody:", nsData(body))
	}
	for k, val := range headers {
		msg(req, "setValue:forHTTPHeaderField:", nsString(val), nsString(k))
	}
	msg(v.View, "loadRequest:", req)
	msg(req, "release")
}

// SetScriptEnabled switches the page scripts of the pages loaded from now on (allowsContentJavaScript).
func (v *WKView) SetScriptEnabled(on bool) { v.scriptOff = !on }

// wkStatusScript tells the native side the link under the pointer; WKWebView has no API for it.
const wkStatusScript = `(function(){var last="";function s(t){if(t!==last){last=t;try{window.webkit.messageHandlers.gowtstatus.postMessage(t)}catch(e){}}}` +
	`document.addEventListener("mouseover",function(e){var a=e.target&&e.target.closest&&e.target.closest("a[href]");s(a?a.href:"")},true);` +
	`document.addEventListener("mouseout",function(e){if(!e.relatedTarget)s("")},true)})();`

// EnableStatus starts Status for the pages loaded from now on, with a script run in every frame.
func (v *WKView) EnableStatus() {
	if v.statusOn {
		return
	}
	v.statusOn = true
	msg(v.ucc, "addScriptMessageHandler:name:", v.obj, nsString("gowtstatus"))
	v.AddScript(wkStatusScript)
}

func wkFeatures(f uintptr) WKFeatures {
	num := func(name string) int {
		if n := msg(f, name); n != 0 {
			return int(int64(msg(n, "integerValue")))
		}
		return 0
	}
	flag := func(name string) bool {
		n := msg(f, name)
		return n != 0 && msg(n, "boolValue")&0xff != 0
	}
	return WKFeatures{X: num("x"), Y: num("y"), Width: num("width"), Height: num("height"),
		MenuBar: flag("menuBarVisibility"), StatusBar: flag("statusBarVisibility"), ToolBar: flag("toolbarsVisibility")}
}

func (v *WKView) LoadHTML(html, baseURL string) {
	base := uintptr(0)
	if baseURL != "" {
		base = msg(class("NSURL"), "URLWithString:", nsString(baseURL))
	}
	msg(v.View, "loadHTMLString:baseURL:", nsString(html), base)
}

// Eval runs js in the page; done gets the result (non-strings through -description) or the error text.
func (v *WKView) Eval(js string, done func(result string, err string)) {
	blk := NewBlock(2, func(a []uintptr) {
		if v.disposed {
			return
		}
		if a[1] != 0 {
			done("", goString(msg(a[1], "localizedDescription")))
			return
		}
		res := a[0]
		if res != 0 && msg(res, "isKindOfClass:", class("NSString"))&0xff == 0 {
			res = msg(res, "description")
		}
		done(goString(res), "")
	})
	msg(v.View, "evaluateJavaScript:completionHandler:", nsString(js), blk)
}

func (v *WKView) serve(task uintptr) {
	req := msg(task, "request")
	url := msg(req, "URL")
	status, headers, body := 404, map[string]string(nil), []byte(nil)
	if v.Scheme != nil {
		status, headers, body = v.Scheme(goString(msg(url, "absoluteString")), goString(msg(req, "HTTPMethod")))
	}
	hd := msg(class("NSMutableDictionary"), "dictionary")
	for k, val := range headers {
		msg(hd, "setObject:forKey:", nsString(val), nsString(k))
	}
	msg(hd, "setObject:forKey:", nsString(strconv.Itoa(len(body))), nsString("Content-Length"))
	resp := msg(msg(msg(class("NSHTTPURLResponse"), "alloc"), "initWithURL:statusCode:HTTPVersion:headerFields:", url, uintptr(status), nsString("HTTP/1.1"), hd), "autorelease")
	msg(task, "didReceiveResponse:", resp)
	msg(task, "didReceiveData:", nsData(body))
	msg(task, "didFinish")
}

func (v *WKView) Dispose() {
	if v.disposed {
		return
	}
	v.disposed = true
	msg(v.View, "removeObserver:forKeyPath:", v.obj, nsString("title"))
	msg(v.ucc, "removeScriptMessageHandlerForName:", nsString("gowt"))
	if v.statusOn {
		msg(v.ucc, "removeScriptMessageHandlerForName:", nsString("gowtstatus"))
	}
	msg(v.View, "setNavigationDelegate:", 0)
	msg(v.View, "setUIDelegate:", 0)
	msg(v.View, "removeFromSuperview")
	delete(wkViews, v.obj)
	msg(v.View, "release")
	msg(v.config, "release")
	msg(v.obj, "release")
}
