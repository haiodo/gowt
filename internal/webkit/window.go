//go:build linux

package webkit

import (
	"unsafe"
)

// sharedDM is the data manager of the first context: every view uses it, so cookies and storage are
// shared by all of them the way WKWebsiteDataStore.default and a WebView2 user data folder are.
var sharedDM uintptr

func dataManager() uintptr {
	if sharedDM == 0 {
		// Never released: it holds the manager for the whole process.
		ctx := webkit_web_context_new()
		sharedDM = g_object_ref(webkit_web_context_get_website_data_manager(ctx))
	}
	return sharedDM
}

func newContext() uintptr {
	if sharedDM == 0 {
		dataManager()
	}
	return webkit_web_context_new_with_website_data_manager(sharedDM)
}

func (v *View) features() WindowFeatures {
	p := webkit_web_view_get_window_properties(v.widget)
	if p == 0 {
		return WindowFeatures{}
	}
	var r [4]int32 // GdkRectangle: x, y, width, height
	webkit_window_properties_get_geometry(p, uintptr(unsafe.Pointer(&r[0])))
	return WindowFeatures{
		X: int(r[0]), Y: int(r[1]), Width: int(r[2]), Height: int(r[3]),
		MenuBar:     webkit_window_properties_get_menubar_visible(p) != 0,
		StatusBar:   webkit_window_properties_get_statusbar_visible(p) != 0,
		ToolBar:     webkit_window_properties_get_toolbar_visible(p) != 0,
		LocationBar: webkit_window_properties_get_locationbar_visible(p) != 0,
	}
}

// SetScriptEnabled switches the page scripts of this view for the pages loaded from now on. A view
// opened by the page gets settings of its own first, as WebKitGTK shares the opener's.
func (v *View) SetScriptEnabled(on bool) {
	if v.related && !v.ownSettings {
		st := webkit_settings_new()
		webkit_settings_set_javascript_can_open_windows_automatically(st, 1)
		webkit_web_view_set_settings(v.widget, st)
		v.ownSettings = true
	}
	var b int32
	if on {
		b = 1
	}
	webkit_settings_set_enable_javascript(webkit_web_view_get_settings(v.widget), b)
}

// LoadRequest loads url with extra request headers (GET only: WebKitGTK has no API for another method).
func (v *View) LoadRequest(url string, headers map[string]string) {
	req := webkit_uri_request_new(url)
	hd := webkit_uri_request_get_http_headers(req)
	if hd != 0 {
		for k, val := range headers {
			soup_message_headers_append(hd, k, val)
		}
	}
	webkit_web_view_load_request(v.widget, req)
	g_object_unref(req)
}

// LoadBytes shows data as a page of mimeType in encoding ("" lets WebKit decide), with baseURL as its address.
func (v *View) LoadBytes(data []byte, mimeType, encoding, baseURL string) {
	var b uintptr
	if len(data) > 0 {
		b = g_bytes_new(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data))) // copies
	} else {
		b = g_bytes_new(0, 0)
	}
	mt, base, enc := cstr(mimeType), cstr(baseURL), uintptr(0)
	if encoding != "" {
		enc = cstr(encoding)
	}
	webkit_web_view_load_bytes(v.widget, b, mt, enc, base)
	g_free(mt)
	g_free(base)
	if enc != 0 {
		g_free(enc)
	}
	g_bytes_unref(b)
}

// Cookie is the part of a cookie the webview package reports.
type Cookie struct{ Name, Value string }

func cookieManager() uintptr {
	return webkit_website_data_manager_get_cookie_manager(dataManager())
}

// async registers f as the GAsyncReadyCallback of a call; the call must pass cbEval and the returned id.
func async(f func(res uintptr)) uintptr {
	mu.Lock()
	nextID++
	id := nextID
	pending[id] = f
	mu.Unlock()
	return id
}

// glist reads the data pointers of a GList (data at 0, next at 8) and frees the list cells.
func glist(l uintptr) []uintptr {
	var out []uintptr
	for n := l; n != 0; n = *(*uintptr)(cptr(n + 8)) {
		out = append(out, *(*uintptr)(cptr(n)))
	}
	g_list_free(l)
	return out
}

// GetCookies reports the cookies the stored jar would send to url.
func GetCookies(url string, done func([]Cookie)) {
	cm := cookieManager()
	id := async(func(res uintptr) {
		defer fixSignalFlags()
		perr := g_malloc(8)
		*(*uintptr)(cptr(perr)) = 0
		l := webkit_cookie_manager_get_cookies_finish(cm, res, perr)
		freeError(perr)
		done(takeCookies(l))
	})
	webkit_cookie_manager_get_cookies(cm, url, 0, cbEval, id)
}

func takeCookies(l uintptr) []Cookie {
	var out []Cookie
	for _, c := range glist(l) {
		out = append(out, Cookie{goString(soup_cookie_get_name(c)), goString(soup_cookie_get_value(c))})
		soup_cookie_free(c)
	}
	return out
}

func freeError(perr uintptr) {
	if e := *(*uintptr)(cptr(perr)); e != 0 {
		g_error_free(e)
	}
	g_free(perr)
}

// SetCookie stores header, a Set-Cookie value without the name of the field, as if url had sent it.
func SetCookie(url, header string, done func(ok bool)) {
	origin := g_uri_parse(url, 0, 0)
	if origin == 0 {
		done(false)
		return
	}
	c := soup_cookie_parse(header, origin)
	g_uri_unref(origin)
	if c == 0 {
		done(false)
		return
	}
	cm := cookieManager()
	id := async(func(res uintptr) {
		defer fixSignalFlags()
		perr := g_malloc(8)
		*(*uintptr)(cptr(perr)) = 0
		ok := webkit_cookie_manager_add_cookie_finish(cm, res, perr) != 0
		freeError(perr)
		done(ok)
	})
	webkit_cookie_manager_add_cookie(cm, c, 0, cbEval, id)
	soup_cookie_free(c)
}

// ClearSessionCookies deletes the cookies that have no expiry date.
func ClearSessionCookies(done func()) {
	cm := cookieManager()
	id := async(func(res uintptr) {
		defer fixSignalFlags()
		perr := g_malloc(8)
		*(*uintptr)(cptr(perr)) = 0
		l := webkit_cookie_manager_get_all_cookies_finish(cm, res, perr)
		freeError(perr)
		var session []uintptr
		for _, c := range glist(l) {
			if soup_cookie_get_expires(c) == 0 {
				session = append(session, c)
			} else {
				soup_cookie_free(c)
			}
		}
		deleteAll(cm, session, done)
	})
	webkit_cookie_manager_get_all_cookies(cm, 0, cbEval, id)
}

// deleteAll deletes the cookies one after another (the next starts when the last one finished) and
// frees them; done runs after the last.
func deleteAll(cm uintptr, cs []uintptr, done func()) {
	if len(cs) == 0 {
		done()
		return
	}
	c := cs[0]
	id := async(func(res uintptr) {
		defer fixSignalFlags()
		perr := g_malloc(8)
		*(*uintptr)(cptr(perr)) = 0
		webkit_cookie_manager_delete_cookie_finish(cm, res, perr)
		freeError(perr)
		soup_cookie_free(c)
		deleteAll(cm, cs[1:], done)
	})
	webkit_cookie_manager_delete_cookie(cm, c, 0, cbEval, id)
}
