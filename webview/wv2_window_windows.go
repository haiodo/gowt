package webview

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strings"
	"unsafe"

	"github.com/haiodo/gowt/internal/jrt"
)

// Slots and IIDs below are from WebView2.h of the Microsoft.Web.WebView2 NuGet package.
var (
	iidWebView2_2      = parseGUID("9E8F0CF8-E670-4B5E-B2BC-73E061E3184C")
	iidWebView2_12     = parseGUID("35D69927-BCFA-4566-9349-6B3E0D154CAC")
	iidEnvironment2    = parseGUID("41f3632b-5ef4-404f-ad82-2d606c5a9a21")
	liveEngines        []*wv2Engine
	errNeedsNewRuntime = errors.New("webview: this WebView2 runtime lacks ICoreWebView2_2, which sending a request needs")
)

// queryInterface returns obj as the interface iid, or 0; the caller releases it.
func queryInterface(obj uintptr, iid [16]byte) uintptr {
	if obj == 0 {
		return 0
	}
	id := heap[[16]byte]()
	*id = iid
	out := heap[uintptr]()
	if vcall(obj, 0, addr(id), addr(out)) != sOK {
		return 0
	}
	return *out
}

func (e *wv2Engine) setNewWindowHandler(f func(n *NewWindow) *WebView) { e.newWin = f }
func (e *wv2Engine) setShowHandler(f func(WindowFeatures))             { e.show = f }
func (e *wv2Engine) setCloseHandler(f func())                          { e.closeH = f }
func (e *wv2Engine) setStatusHandler(f func(string))                   { e.status = f }

func (e *wv2Engine) setScriptEnabled(on bool) {
	e.script = &on
	e.applyScript()
}

// applyScript passes the switch to the settings; WebView2 applies it from the next navigation.
func (e *wv2Engine) applyScript() {
	if e.script == nil || e.view == 0 || e.disposed {
		return
	}
	out := heap[uintptr]()
	if vcall(e.view, 3, addr(out)) == sOK { // get_Settings
		vcall(*out, 4, b2u(*e.script)) // put_IsScriptEnabled
		release(*out)
	}
}

// hookWindowEvents is part of hookEvents: the new window, close and status events. NewWindowRequested is
// hooked only with a handler, as without one WebView2 opens its own window.
func (e *wv2Engine) hookWindowEvents(add func(slot int, target uintptr, fn func(a, b uintptr))) {
	if e.newWin != nil {
		add(44, e.view, func(_, args uintptr) { e.newWindowRequested(args) }) // NewWindowRequested
	}
	add(59, e.view, func(_, _ uintptr) { // WindowCloseRequested
		if e.closeH != nil {
			e.closeH()
		}
	})
	if e.status != nil {
		// ICoreWebView2_12; an older runtime has no status bar event.
		if v12 := queryInterface(e.view, iidWebView2_12); v12 != 0 {
			add(102, v12, func(_, _ uintptr) { // StatusBarTextChanged
				out := heap[uintptr]()
				if vcall(v12, 104, addr(out)) == sOK && e.status != nil { // get_StatusBarText
					e.status(takeString(*out))
				}
			})
		}
	}
}

func (e *wv2Engine) newWindowRequested(args uintptr) {
	n := &NewWindow{opener: e, native: args}
	if out := heap[uintptr](); vcall(args, 3, addr(out)) == sOK { // get_Uri
		n.URL = takeString(*out)
	}
	n.features = windowFeatures(args)
	if out := heap[uintptr](); vcall(args, 9, addr(out)) == sOK { // GetDeferral
		n.deferral = *out
	}
	vcall(args, 1) // AddRef: the answer comes after this handler returns
	wv := e.newWin(n)
	if pe, ok := viewEngine(wv); ok && pe.popup == n {
		return // pe completes the request when its view is ready
	}
	vcall(args, 6, 1) // put_Handled: the window is refused
	finishPopup(n)
}

func viewEngine(wv *WebView) (*wv2Engine, bool) {
	if wv == nil {
		return nil, false
	}
	pe, ok := wv.e.(*wv2Engine)
	return pe, ok
}

func windowFeatures(args uintptr) (f WindowFeatures) {
	out := heap[uintptr]()
	if vcall(args, 10, addr(out)) != sOK { // get_WindowFeatures
		return
	}
	wf := *out
	defer release(wf)
	flag := func(slot int) bool {
		b := heap[int32]()
		return vcall(wf, slot, addr(b)) == sOK && *b != 0
	}
	num := func(slot int) int {
		u := heap[uint32]()
		if vcall(wf, slot, addr(u)) != sOK {
			return 0
		}
		return int(*u)
	}
	if flag(3) { // get_HasPosition
		f.X, f.Y = num(5), num(6) // get_Left, get_Top
	}
	if flag(4) { // get_HasSize
		f.Height, f.Width = num(7), num(8)
	}
	f.MenuBar, f.StatusBar, f.ToolBar = flag(9), flag(10), flag(11)
	return
}

// finishPopup completes the deferral of the request and lets go of what it holds.
func finishPopup(n *NewWindow) {
	if n.deferral != 0 {
		vcall(n.deferral, 3) // Complete
		release(n.deferral)
		n.deferral = 0
	}
	release(n.native)
	n.native = 0
}

// completePopup hands the ready view to the page that asked for it, and reports the window as ready to show.
func (e *wv2Engine) completePopup() {
	n := e.popup
	if n == nil || n.native == 0 {
		return
	}
	vcall(n.native, 4, e.view) // put_NewWindow
	vcall(n.native, 6, 1)      // put_Handled
	finishPopup(n)
	e.host.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
		if !e.disposed && e.show != nil {
			e.show(n.features)
		}
	}))
}

// refusePopup answers a request whose view could not be made.
func (e *wv2Engine) refusePopup() {
	if n := e.popup; n != nil && n.native != 0 {
		vcall(n.native, 6, 1) // put_Handled, no NewWindow
		finishPopup(n)
	}
}

// loadRequest navigates with method, headers and body through ICoreWebView2_2.
func (e *wv2Engine) loadRequest(r LoadRequest) {
	e.do(func() {
		v2 := queryInterface(e.view, iidWebView2_2)
		env2 := queryInterface(e.env, iidEnvironment2)
		defer release(v2)
		defer release(env2)
		if v2 == 0 || env2 == 0 {
			e.navigationFailed(r.URL)(errNeedsNewRuntime)
			return
		}
		var stream uintptr
		if len(r.Body) > 0 {
			stream, _, _ = procMemStream.Call(uintptr(unsafe.Pointer(&r.Body[0])), uintptr(len(r.Body)))
			defer release(stream)
		}
		method := r.Method
		if method == "" {
			method = "GET"
		}
		var hd strings.Builder
		for k, v := range r.Headers {
			hd.WriteString(k + ": " + v + "\r\n")
		}
		uri, m, h := utf16Z(r.URL), utf16Z(method), utf16Z(hd.String())
		req := heap[uintptr]()
		// CreateWebResourceRequest(uri, method, postData, headers)
		if hr := vcall(env2, 8, addr(&uri[0]), addr(&m[0]), stream, addr(&h[0]), addr(req)); failed(hr) {
			e.navigationFailed(r.URL)(hrErr("CreateWebResourceRequest", hr))
			return
		}
		defer release(*req)
		if hr := vcall(v2, 63, *req); failed(hr) { // NavigateWithWebResourceRequest
			e.navigationFailed(r.URL)(hrErr("NavigateWithWebResourceRequest", hr))
		}
	}, e.navigationFailed(r.URL))
}

// ---------------------------------------------------------------- cookies

// cookieManager is the manager of the profile of any live view, or 0 (the caller releases it).
func cookieManager() uintptr {
	for _, e := range liveEngines {
		if e.view == 0 || e.disposed {
			continue
		}
		v2 := queryInterface(e.view, iidWebView2_2)
		if v2 == 0 {
			continue
		}
		out := heap[uintptr]()
		hr := vcall(v2, 66, addr(out)) // get_CookieManager
		release(v2)
		if !failed(hr) {
			return *out
		}
	}
	return 0
}

func cookies(rawURL string, done func([]Cookie, bool)) {
	cm := cookieManager()
	if cm == 0 {
		done(nil, false)
		return
	}
	defer release(cm)
	getCookies(cm, rawURL, func(list uintptr) {
		var out []Cookie
		for _, c := range cookieList(list) {
			out = append(out, Cookie{cookieString(c, 3), cookieString(c, 4)}) // get_Name, get_Value
			release(c)
		}
		release(list)
		done(out, true)
	}, func() { done(nil, false) })
}

// getCookies asks for the cookies of uri ("" for all of the profile); f gets the list and owns the cookies in it.
func getCookies(cm uintptr, uri string, f func(list uintptr), failedFn func()) {
	u := utf16Z(uri)
	h := newHandler(func(hr, list uintptr) uintptr {
		if failed(hr) || list == 0 {
			failedFn()
			return sOK
		}
		f(list)
		return sOK
	})
	hr := vcall(cm, 5, addr(&u[0]), h.ptr()) // GetCookies
	release(h.ptr())
	if failed(hr) {
		failedFn()
	}
}

// cookieList reads an ICoreWebView2CookieList; the caller releases each cookie.
func cookieList(list uintptr) []uintptr {
	n := heap[uint32]()
	if vcall(list, 3, addr(n)) != sOK { // get_Count
		return nil
	}
	var out []uintptr
	for i := uint32(0); i < *n; i++ {
		c := heap[uintptr]()
		if vcall(list, 4, uintptr(i), addr(c)) == sOK { // GetValueAtIndex
			out = append(out, *c)
		}
	}
	return out
}

func cookieString(c uintptr, slot int) string {
	out := heap[uintptr]()
	if vcall(c, slot, addr(out)) != sOK {
		return ""
	}
	return takeString(*out)
}

func setCookie(rawURL, header string, done func(bool)) {
	cm := cookieManager()
	if cm == 0 {
		done(false)
		return
	}
	defer release(cm)
	u, err := url.Parse(rawURL)
	hdr := http.Header{}
	hdr.Add("Set-Cookie", header)
	parsed := (&http.Response{Header: hdr}).Cookies()
	if err != nil || len(parsed) == 0 {
		done(false)
		return
	}
	pc := parsed[0]
	domain, path := pc.Domain, pc.Path
	if domain == "" {
		domain = u.Hostname()
	}
	if path == "" {
		path = "/"
	}
	name, value, d, p := utf16Z(pc.Name), utf16Z(pc.Value), utf16Z(domain), utf16Z(path)
	c := heap[uintptr]()
	if failed(vcall(cm, 3, addr(&name[0]), addr(&value[0]), addr(&d[0]), addr(&p[0]), addr(c))) { // CreateCookie
		done(false)
		return
	}
	defer release(*c)
	if !pc.Expires.IsZero() {
		// A double argument: Go's Windows syscall stub copies argument 1 into XMM1 as well as RDX.
		vcall(*c, 9, uintptr(math.Float64bits(float64(pc.Expires.Unix())))) // put_Expires
	}
	vcall(*c, 11, b2u(pc.HttpOnly)) // put_IsHttpOnly
	vcall(*c, 15, b2u(pc.Secure))   // put_IsSecure
	done(!failed(vcall(cm, 6, *c))) // AddOrUpdateCookie
}

func clearSessionCookies(done func()) {
	cm := cookieManager()
	if cm == 0 {
		done()
		return
	}
	getCookies(cm, "", func(list uintptr) {
		for _, c := range cookieList(list) {
			s := heap[int32]()
			if vcall(c, 16, addr(s)) == sOK && *s != 0 { // get_IsSession
				vcall(cm, 7, c) // DeleteCookie
			}
			release(c)
		}
		release(list)
		release(cm)
		done()
	}, func() {
		release(cm)
		done()
	})
}
