//go:build darwin

// WKWebView binding for the webview package, hand-written over purego. One runtime ObjC class
// (gowtWKHandler) is the script-message handler, URL-scheme handler, navigation delegate and title
// observer of every view; its callbacks find the Go side by the instance address. All calls and
// callbacks run on the UI (main) thread.
package cocoa

import (
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

	config, ucc, obj uintptr
	disposed         bool
}

var (
	wkOnce  sync.Once
	wkClass uintptr
	wkViews = map[uintptr]*WKView{}
	wkInit  func(self, sel uintptr, frame NSRect, config uintptr) uintptr
)

// The page-side half of window.gowt.postMessage.
const wkBridge = `window.gowt={postMessage:function(m){window.webkit.messageHandlers.gowt.postMessage(typeof m==="string"?m:JSON.stringify(m))},call:function(m){return window.prompt(typeof m==="string"?m:JSON.stringify(m),"\u0001gowt")}};`

// wkCallMark is the prompt's default text that tells window.gowt.call from a page's own prompt().
const wkCallMark = "\x01gowt"

func wkSetup() {
	wkOnce.Do(func() {
		objcInit()
		if _, err := purego.Dlopen("/System/Library/Frameworks/WebKit.framework/WebKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
			panic("gowt/internal/cocoa: dlopen WebKit: " + err.Error())
		}
		purego.RegisterFunc(&wkInit, objcMsgSend)
		view := func(self uintptr) *WKView { return wkViews[self] }
		urlOf := func(wv uintptr) string { return goString(msg(msg(wv, "URL"), "absoluteString")) }
		wkClass = newClass("gowtWKHandler", []string{"WKScriptMessageHandler", "WKURLSchemeHandler", "WKNavigationDelegate", "WKUIDelegate"}, map[string]objcMethod{
			"userContentController:didReceiveScriptMessage:": {purego.NewCallback(func(self, _, _, m uintptr) {
				if v := view(self); v != nil && v.Message != nil {
					v.Message(goString(msg(m, "body")))
				}
			}), "v@:@@"},
			"webView:didStartProvisionalNavigation:": {purego.NewCallback(func(self, _, wv, _ uintptr) {
				if v := view(self); v != nil && v.NavStarted != nil {
					v.NavStarted(urlOf(wv))
				}
			}), "v@:@@"},
			"webView:didFinishNavigation:": {purego.NewCallback(func(self, _, wv, _ uintptr) {
				if v := view(self); v != nil && v.NavFinished != nil {
					v.NavFinished(urlOf(wv))
				}
			}), "v@:@@"},
			"webView:didFailNavigation:withError:": {purego.NewCallback(func(self, _, wv, _, e uintptr) {
				if v := view(self); v != nil && v.NavFailed != nil {
					v.NavFailed(urlOf(wv), goString(msg(e, "localizedDescription")))
				}
			}), "v@:@@@"},
			"webView:didFailProvisionalNavigation:withError:": {purego.NewCallback(func(self, _, wv, _, e uintptr) {
				if v := view(self); v != nil && v.NavFailed != nil {
					v.NavFailed(urlOf(wv), goString(msg(e, "localizedDescription")))
				}
			}), "v@:@@@"},
			"webView:decidePolicyForNavigationAction:decisionHandler:": {purego.NewCallback(func(self, _, _, action, handler uintptr) {
				allow := true
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
				policy := uintptr(0) // WKNavigationActionPolicyCancel
				if allow {
					policy = 1 // WKNavigationActionPolicyAllow
				}
				callBlock(handler, policy)
			}), "v@:@@@?"},
			"webView:runJavaScriptTextInputPanelWithPrompt:defaultText:initiatedByFrame:completionHandler:": {purego.NewCallback(func(self, _, _, prompt, def, frame, handler uintptr) {
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
	v.config = msg(msg(class("WKWebViewConfiguration"), "alloc"), "init")
	v.ucc = msg(v.config, "userContentController")
	msg(v.ucc, "addScriptMessageHandler:name:", v.obj, nsString("gowt"))
	v.AddScript(wkBridge)
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
func (v *WKView) AddScript(js string) {
	us := msg(msg(class("WKUserScript"), "alloc"), "initWithSource:injectionTime:forMainFrameOnly:", nsString(js), 0, 0)
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
	msg(v.View, "loadRequest:", req)
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
	msg(v.View, "setNavigationDelegate:", 0)
	msg(v.View, "setUIDelegate:", 0)
	msg(v.View, "removeFromSuperview")
	delete(wkViews, v.obj)
	msg(v.View, "release")
	msg(v.config, "release")
	msg(v.obj, "release")
}
