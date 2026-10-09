// WebView2 backend without cgo and without WebView2Loader.dll: the installed Evergreen runtime is found through
// the registry (wv2_util.go) and its EmbeddedBrowserWebView.dll creates the environment; the COM callbacks are
// Go objects with shared vtables (wv2_com_windows.go).

package webview

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

type disposeFunc func()

func (f disposeFunc) WidgetDisposed(*swt.DisposeEvent) { f() }

// op is work waiting for the controller; abort runs instead when creation fails.
type op struct {
	run   func()
	abort func(error)
}

type wv2Engine struct {
	w       *WebView
	host    *swt.Composite
	opts    Options
	dll     string
	schemes map[string]SchemeHandler
	scripts []string

	started, disposed, isReady bool
	err                        error
	pending                    []op

	env, ctl, view uintptr
	lastURL        string

	policy    func(url string, mainFrame bool) bool
	call      func(msg string) string
	cancelled bool
}

func newEngine(w *WebView, parent *swt.Composite, opts Options) (engine, error) {
	dll, err := findRuntimeDLL(realRuntimeEnv(), runtimeArch())
	if err != nil {
		return nil, err
	}
	host := opts.host
	if host == nil {
		host = swt.NewCompositeParentStyle(parent, swt.NONE)
	}
	e := &wv2Engine{w: w, host: host, opts: opts, dll: dll,
		schemes: map[string]SchemeHandler{}, scripts: []string{wv2Bridge}}
	e.host.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { e.resize() }))
	e.host.AddFocusListener(swt.FocusListenerFocusGainedAdapter(func(*swt.FocusEvent) {
		if e.ctl != 0 {
			vcall(e.ctl, 12, 2) // MoveFocus(PROGRAMMATIC)
		}
	}))
	if sh := e.host.GetShell(); sh != nil {
		sh.AddControlListener(swt.ControlListenerControlMovedAdapter(func(*swt.ControlEvent) {
			if e.ctl != 0 {
				vcall(e.ctl, 23) // NotifyParentWindowPositionChanged
			}
		}))
	}
	e.host.AddDisposeListener(disposeFunc(e.release))
	return e, nil
}

// start creates the environment and the controller; the scheme list is frozen from here on. Everything that
// needs the view waits in pending until the completion handlers have run.
func (e *wv2Engine) start() {
	if e.started {
		return
	}
	e.started = true
	var names []string
	for s := range e.schemes {
		names = append(names, s)
	}
	opts := newOptions(names)
	h := newHandler(func(hr, env uintptr) uintptr {
		e.envReady(hr, env)
		return sOK
	})
	exe, _ := os.Executable()
	err := createEnvironment(e.dll, userDataDir(os.Getenv("LOCALAPPDATA"), exe), opts.ptr(), h.ptr())
	release(h.ptr())
	release(opts.ptr())
	if err != nil {
		e.fail(err)
	}
}

func (e *wv2Engine) envReady(hr, env uintptr) {
	if failed(hr) || env == 0 {
		e.fail(hrErr("creating the WebView2 environment", hr))
		return
	}
	vcall(env, 1)
	e.env = env
	if e.disposed {
		return
	}
	h := newHandler(func(hr, ctl uintptr) uintptr {
		e.controllerReady(hr, ctl)
		return sOK
	})
	hr = vcall(env, 3, uintptr(e.host.Handle), h.ptr()) // CreateCoreWebView2Controller
	release(h.ptr())
	if failed(hr) {
		e.fail(hrErr("CreateCoreWebView2Controller", hr))
	}
}

func (e *wv2Engine) controllerReady(hr, ctl uintptr) {
	if failed(hr) || ctl == 0 {
		e.fail(hrErr("creating the WebView2 controller", hr))
		return
	}
	vcall(ctl, 1)
	e.ctl = ctl
	if e.disposed {
		e.closeController()
		return
	}
	out := heap[uintptr]()
	if hr := vcall(ctl, 25, addr(out)); failed(hr) { // get_CoreWebView2
		e.fail(hrErr("get_CoreWebView2", hr))
		return
	}
	e.view = *out
	if vcall(e.view, 3, addr(out)) == sOK { // get_Settings
		vcall(*out, 12, b2u(e.opts.Inspectable)) // put_AreDevToolsEnabled
		release(*out)
	}
	e.resize()
	e.hookEvents()
	for s := range e.schemes {
		u := utf16Z(s + "://*")
		vcall(e.view, 57, addr(&u[0]), 0) // AddWebResourceRequestedFilter(uri, ALL)
	}
	e.addScripts(0)
}

func b2u(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// addScripts registers the document-start scripts one after another: a navigation must not begin before all
// of them are in.
func (e *wv2Engine) addScripts(i int) {
	if i >= len(e.scripts) {
		e.ready()
		return
	}
	u := utf16Z(e.scripts[i])
	h := newHandler(func(hr, id uintptr) uintptr {
		e.addScripts(i + 1)
		return sOK
	})
	hr := vcall(e.view, 27, addr(&u[0]), h.ptr())
	release(h.ptr())
	if failed(hr) {
		e.fail(hrErr("AddScriptToExecuteOnDocumentCreated", hr))
	}
}

func (e *wv2Engine) ready() {
	if e.disposed {
		return
	}
	e.isReady = true
	p := e.pending
	e.pending = nil
	for _, o := range p {
		o.run()
	}
}

func (e *wv2Engine) fail(err error) {
	if e.err != nil {
		return
	}
	e.err = err
	p := e.pending
	e.pending = nil
	for _, o := range p {
		o.abort(err)
	}
}

// do runs f now if the view exists, else when it does; abort gets the creation error.
func (e *wv2Engine) do(f func(), abort func(error)) {
	switch {
	case e.disposed:
		abort(errors.New("webview: disposed"))
	case e.err != nil:
		abort(e.err)
	case e.isReady:
		f()
	default:
		e.pending = append(e.pending, op{f, abort})
		e.start()
	}
}

func (e *wv2Engine) resize() {
	if e.ctl == 0 || e.host.IsDisposed() {
		return
	}
	a := e.host.GetClientAreaInPixels()
	r := heap[[4]int32]()
	*r = [4]int32{0, 0, a.Width, a.Height}
	if isARM64 {
		// Windows ARM64 passes a 16-byte struct in two registers, x64 by reference.
		w := (*[2]uintptr)(unsafe.Pointer(r))
		vcall(e.ctl, 6, w[0], w[1])
		return
	}
	vcall(e.ctl, 6, addr(r)) // put_Bounds
}

func (e *wv2Engine) hookEvents() {
	w := e.w
	add := func(slot int, target uintptr, fn func(a, b uintptr)) {
		h := newHandler(func(a, b uintptr) uintptr {
			if !e.disposed {
				fn(a, b)
			}
			return sOK
		})
		tok := heap[int64]()
		vcall(target, slot, h.ptr(), addr(tok))
		release(h.ptr())
	}
	str := func(obj uintptr, slot int) string {
		out := heap[uintptr]()
		if vcall(obj, slot, addr(out)) != sOK {
			return ""
		}
		return takeString(*out)
	}
	add(7, e.view, func(_, args uintptr) { // NavigationStarting
		e.cancelled = false
		e.lastURL = str(args, 3)
		if e.policy != nil && !e.policy(e.lastURL, true) {
			e.cancelled = true
			vcall(args, 8, 1) // put_Cancel
			return
		}
		if w.OnNavigationStarted != nil {
			w.OnNavigationStarted(e.lastURL)
		}
	})
	add(17, e.view, func(_, args uintptr) { // FrameNavigationStarting
		if e.policy != nil && !e.policy(str(args, 3), false) {
			vcall(args, 8, 1) // put_Cancel
		}
	})
	add(21, e.view, func(_, args uintptr) { e.scriptDialog(args) }) // ScriptDialogOpening
	add(15, e.view, func(_, args uintptr) {                         // NavigationCompleted
		ok := heap[int32]()
		status := heap[int32]()
		vcall(args, 3, addr(ok))
		vcall(args, 4, addr(status))
		if e.cancelled && *ok == 0 && *status == wv2OperationCanceled {
			e.cancelled = false
			return
		}
		u := e.url()
		if *ok != 0 {
			if w.OnNavigationFinished != nil {
				w.OnNavigationFinished(u)
			}
		} else if w.OnNavigationFailed != nil {
			w.OnNavigationFailed(u, fmt.Errorf("webview: navigation failed (COREWEBVIEW2_WEB_ERROR_STATUS %d)", *status))
		}
	})
	add(46, e.view, func(_, _ uintptr) { // DocumentTitleChanged
		if w.OnTitleChanged != nil {
			w.OnTitleChanged(str(e.view, 48))
		}
	})
	add(34, e.view, func(_, args uintptr) { // WebMessageReceived
		if w.OnMessage != nil {
			w.OnMessage(messageText(str(args, 4)))
		}
	})
	add(55, e.view, func(_, args uintptr) { e.serve(args) }) // WebResourceRequested
	add(13, e.ctl, func(_, args uintptr) {                   // MoveFocusRequested
		reason := heap[int32]()
		vcall(args, 3, addr(reason))
		switch *reason {
		case 0:
			e.host.TraverseTraversal(swt.TRAVERSE_TAB_NEXT)
		case 1:
			e.host.TraverseTraversal(swt.TRAVERSE_TAB_PREVIOUS)
		default:
			return
		}
		vcall(args, 5, 1) // put_Handled
	})
}

// scriptDialog answers the prompt window.gowt.call raises, the way the darwin engine does: the page blocks
// in prompt() until the handler is done, and the reply is the prompt's result. Other dialogs and prompts are
// left to the default UI.
func (e *wv2Engine) scriptDialog(args uintptr) {
	kind := heap[int32]()
	if e.call == nil || vcall(args, 4, addr(kind)) != sOK || *kind != wv2DialogPrompt { // get_Kind
		return
	}
	def := heap[uintptr]()
	if vcall(args, 7, addr(def)) != sOK || takeString(*def) != wv2CallMark { // get_DefaultText
		return
	}
	from := heap[uintptr]()
	if vcall(args, 3, addr(from)) != sOK || !sameDocument(takeString(*from), e.url()) { // get_Uri
		// A subframe asked: answer empty so the marker prompt does not reach the user, and never call f.
		empty := utf16Z("")
		vcall(args, 9, addr(&empty[0]))
		vcall(args, 6)
		return
	}
	msg := heap[uintptr]()
	if vcall(args, 5, addr(msg)) != sOK { // get_Message
		return
	}
	reply := utf16Z(e.call(takeString(*msg)))
	vcall(args, 9, addr(&reply[0])) // put_ResultText
	vcall(args, 6)                  // Accept
}

func (e *wv2Engine) serve(args uintptr) {
	req := heap[uintptr]()
	if vcall(args, 3, addr(req)) != sOK { // get_Request
		return
	}
	defer release(*req)
	str := func(slot int) string {
		out := heap[uintptr]()
		if vcall(*req, slot, addr(out)) != sOK {
			return ""
		}
		return takeString(*out)
	}
	url, method := str(3), str(5)
	var r *Response
	if h := e.schemes[schemeOf(url)]; h != nil {
		r = h(Request{URL: url, Method: method})
	}
	if r == nil {
		r = &Response{Status: 404}
	} else if r.Status == 0 {
		cp := *r
		cp.Status = 200
		r = &cp
	}
	var body unsafe.Pointer
	if len(r.Body) > 0 {
		body = unsafe.Pointer(&r.Body[0])
	}
	stream, _, _ := procMemStream.Call(uintptr(body), uintptr(len(r.Body)))
	if stream == 0 {
		return
	}
	defer release(stream)
	reason, headers := utf16Z(statusText(r.Status)), utf16Z(responseHeaders(r))
	resp := heap[uintptr]()
	if failed(vcall(e.env, 4, stream, uintptr(r.Status), addr(&reason[0]), addr(&headers[0]), addr(resp))) {
		return
	}
	vcall(args, 5, *resp) // put_Response
	release(*resp)
}

func (e *wv2Engine) control() *swt.Composite { return e.host }

func (e *wv2Engine) navigationFailed(url string) func(error) {
	return func(err error) {
		if e.w.OnNavigationFailed != nil {
			e.w.OnNavigationFailed(url, err)
		}
	}
}

func (e *wv2Engine) load(slot int, content, url string) {
	e.do(func() {
		u := utf16Z(content)
		if hr := vcall(e.view, slot, addr(&u[0])); failed(hr) {
			e.navigationFailed(url)(hrErr("Navigate", hr))
		}
	}, e.navigationFailed(url))
}

func (e *wv2Engine) navigate(url string) { e.load(5, url, url) }

// baseURL is ignored: NavigateToString has no base, and its content is limited to 2 MB.
func (e *wv2Engine) setHTML(html, baseURL string) { e.load(6, html, "") }

func (e *wv2Engine) eval(js string, done func(string, error)) {
	e.do(func() {
		u := utf16Z(evalWrapper(js))
		h := newHandler(func(hr, res uintptr) uintptr {
			if failed(hr) {
				done("", hrErr("ExecuteScript", hr))
				return sOK
			}
			s, err := evalResult(goString(res))
			done(s, err)
			return sOK
		})
		hr := vcall(e.view, 29, addr(&u[0]), h.ptr())
		release(h.ptr())
		if failed(hr) {
			done("", hrErr("ExecuteScript", hr))
		}
	}, func(err error) { done("", err) })
}

func (e *wv2Engine) addScript(js string) {
	if !e.isReady {
		e.scripts = append(e.scripts, js)
		return
	}
	u := utf16Z(js)
	h := newHandler(func(hr, id uintptr) uintptr { return sOK })
	vcall(e.view, 27, addr(&u[0]), h.ptr())
	release(h.ptr())
}

func (e *wv2Engine) handleScheme(scheme string, h SchemeHandler) error {
	if e.started {
		return errors.New("webview: HandleScheme after the first load")
	}
	e.schemes[strings.ToLower(scheme)] = h
	return nil
}

func (e *wv2Engine) goBack()    { e.nav(40) }
func (e *wv2Engine) goForward() { e.nav(41) }
func (e *wv2Engine) reload()    { e.nav(31) }

func (e *wv2Engine) canGoBack() bool    { return e.flag(38) }
func (e *wv2Engine) canGoForward() bool { return e.flag(39) }

func (e *wv2Engine) flag(slot int) bool {
	if e.view == 0 || e.disposed {
		return false
	}
	out := heap[int32]()
	return vcall(e.view, slot, addr(out)) == sOK && *out != 0
}

func (e *wv2Engine) stop() { e.nav(43) }

// Set before the first load, so the events are hooked after it.
func (e *wv2Engine) setNavigationPolicy(f func(url string, mainFrame bool) bool) { e.policy = f }

// window.gowt.call exists only with a handler, as without one the default prompt UI would show.
func (e *wv2Engine) setCallHandler(f func(msg string) string) {
	if e.call == nil {
		e.addScript(wv2CallBridge)
	}
	e.call = f
}

func (e *wv2Engine) nav(slot int) {
	if e.view != 0 && !e.disposed {
		vcall(e.view, slot)
	}
}

func (e *wv2Engine) url() string {
	if e.view == 0 || e.disposed {
		return ""
	}
	out := heap[uintptr]()
	if vcall(e.view, 4, addr(out)) != sOK { // get_Source
		return e.lastURL
	}
	return takeString(*out)
}

func (e *wv2Engine) closeController() {
	if e.ctl != 0 {
		vcall(e.ctl, 24) // Close
	}
}

func (e *wv2Engine) release() {
	if e.disposed {
		return
	}
	e.disposed = true
	e.closeController()
	release(e.view)
	release(e.ctl)
	release(e.env)
	e.view, e.ctl, e.env = 0, 0, 0
	e.fail(errors.New("webview: disposed"))
}

func (e *wv2Engine) dispose() { e.host.Dispose() }
