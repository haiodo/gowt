//go:build linux

package webkit

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The page-side half of window.gowt.postMessage.
const bridge = `window.gowt={postMessage:function(m){window.webkit.messageHandlers.gowt.postMessage(typeof m==="string"?m:JSON.stringify(m))}};`

// callShim is the page-side half of window.gowt.call. It is injected in the main frame only, with a
// per-view mark in the prompt's default text: the script-dialog signal does not say which frame
// prompted, so a subframe, which never sees the mark, cannot reach Call.
const callShim = `window.gowt.call=function(m){return window.prompt(typeof m==="string"?m:JSON.stringify(m),"%s")};`

// Config is fixed at New: WebKitGTK registers schemes on the context, so they cannot follow the view.
type Config struct {
	Schemes     []string
	Scripts     []string
	Inspectable bool
}

type View struct {
	Message      func(string)
	NavStarted   func(url string)
	NavFinished  func(url string)
	NavFailed    func(url, msg string)
	TitleChanged func(string)
	// Scheme answers a request on one of Config.Schemes; Content-Type goes in headers.
	Scheme func(url, method string) (status int, headers map[string]string, body []byte)
	// Decide is asked before each navigation; false cancels it. WebKitGTK does not say which frame
	// navigates, so mainFrame is always true.
	Decide func(url string, mainFrame bool) bool
	// Call answers window.gowt.call(msg) from the page, which blocks in window.prompt meanwhile.
	Call func(msg string) string

	ctx, widget, ucm uintptr
	handlers         []uintptr
	id               uintptr
	failed           bool
	callMark         string
	disposed         bool
}

var (
	cbOnce                                                 sync.Once
	cbScheme, cbMessage, cbLoad, cbFailed, cbTitle, cbEval uintptr
	cbPolicy, cbDialog                                     uintptr
	gFree                                                  uintptr

	mu      sync.Mutex
	views   = map[uintptr]*View{}
	pending = map[uintptr]func(res uintptr){}
	nextID  uintptr
)

func lookupView(id uintptr) *View {
	mu.Lock()
	defer mu.Unlock()
	return views[id]
}

func initCallbacks() {
	gFree = lookup("g_free")
	cbScheme = purego.NewCallback(func(req, id uintptr) {
		if v := lookupView(id); v != nil && !v.disposed {
			v.serve(req)
		}
	})
	cbMessage = purego.NewCallback(func(_, val, id uintptr) {
		defer fixSignalFlags()
		if v := lookupView(id); v != nil && !v.disposed && v.Message != nil {
			v.Message(takeString(jsc_value_to_string(webkit_javascript_result_get_js_value(val))))
		}
	})
	cbLoad = purego.NewCallback(func(_, event, id uintptr) {
		defer fixSignalFlags()
		v := lookupView(id)
		if v == nil || v.disposed {
			return
		}
		switch int32(event) {
		case 0: // WEBKIT_LOAD_STARTED
			v.failed = false
			if v.NavStarted != nil {
				v.NavStarted(v.URL())
			}
		case 3: // WEBKIT_LOAD_FINISHED
			if !v.failed && v.NavFinished != nil {
				v.NavFinished(v.URL())
			}
		}
	})
	cbFailed = purego.NewCallback(func(_, _, uri, gerr, id uintptr) uintptr {
		if v := lookupView(id); v != nil && !v.disposed {
			v.failed = true
			if v.NavFailed != nil {
				// GError: guint32 domain, gint code, gchar *message.
				v.NavFailed(goString(uri), goString(*(*uintptr)(cptr(gerr + 8))))
			}
		}
		return 0
	})
	cbTitle = purego.NewCallback(func(_, _, id uintptr) {
		if v := lookupView(id); v != nil && !v.disposed && v.TitleChanged != nil {
			v.TitleChanged(goString(webkit_web_view_get_title(v.widget)))
		}
	})
	cbPolicy = purego.NewCallback(func(_, decision, typ, id uintptr) uintptr {
		v := lookupView(id)
		// WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION = 0, _NEW_WINDOW_ACTION = 1.
		if v == nil || v.disposed || v.Decide == nil || int32(typ) > 1 {
			return 0
		}
		action := webkit_navigation_policy_decision_get_navigation_action(decision)
		if v.Decide(goString(webkit_uri_request_get_uri(webkit_navigation_action_get_request(action))), true) {
			webkit_policy_decision_use(decision)
		} else {
			webkit_policy_decision_ignore(decision)
		}
		return 1
	})
	cbDialog = purego.NewCallback(func(_, dialog, id uintptr) uintptr {
		v := lookupView(id)
		// WEBKIT_SCRIPT_DIALOG_PROMPT = 2.
		if v == nil || v.disposed || v.Call == nil || webkit_script_dialog_get_dialog_type(dialog) != 2 ||
			goString(webkit_script_dialog_prompt_get_default_text(dialog)) != v.callMark {
			return 0
		}
		webkit_script_dialog_prompt_set_text(dialog, v.Call(goString(webkit_script_dialog_get_message(dialog))))
		return 1
	})
	cbEval = purego.NewCallback(func(src, res, id uintptr) {
		defer fixSignalFlags()
		mu.Lock()
		f := pending[id]
		delete(pending, id)
		mu.Unlock()
		if f != nil {
			f(res)
		}
	})
}

func New(cfg Config) (*View, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	cbOnce.Do(initCallbacks)
	v := &View{}
	mu.Lock()
	nextID++
	v.id = nextID
	views[v.id] = v
	mu.Unlock()

	v.ctx = webkit_web_context_new()
	sm := webkit_web_context_get_security_manager(v.ctx)
	for _, s := range cfg.Schemes {
		webkit_web_context_register_uri_scheme(v.ctx, s, cbScheme, v.id, 0)
		webkit_security_manager_register_uri_scheme_as_secure(sm, s)
		webkit_security_manager_register_uri_scheme_as_cors_enabled(sm, s)
	}
	v.widget = webkit_web_view_new_with_context(v.ctx)
	fixSignalFlags()
	v.ucm = webkit_web_view_get_user_content_manager(v.widget)
	if cfg.Inspectable {
		webkit_settings_set_enable_developer_extras(webkit_web_view_get_settings(v.widget), 1)
	}
	v.connect(v.ucm, "script-message-received::gowt", cbMessage)
	webkit_user_content_manager_register_script_message_handler(v.ucm, "gowt")
	v.connect(v.widget, "load-changed", cbLoad)
	v.connect(v.widget, "load-failed", cbFailed)
	v.connect(v.widget, "notify::title", cbTitle)
	v.connect(v.widget, "decide-policy", cbPolicy)
	v.connect(v.widget, "script-dialog", cbDialog)
	v.AddScript(bridge)
	var nonce [16]byte
	rand.Read(nonce[:])
	v.callMark = "\x01gowt" + hex.EncodeToString(nonce[:])
	// WEBKIT_USER_CONTENT_INJECT_TOP_FRAME_ONLY = 1.
	us := webkit_user_script_new(fmt.Sprintf(callShim, "\\u0001"+v.callMark[1:]), 1, 0, 0, 0)
	webkit_user_content_manager_add_script(v.ucm, us)
	webkit_user_script_unref(us)
	for _, s := range cfg.Scripts {
		v.AddScript(s)
	}
	return v, nil
}

func (v *View) connect(obj uintptr, signal string, cb uintptr) {
	v.handlers = append(v.handlers, obj, g_signal_connect_data(obj, signal, cb, v.id, 0, 0))
}

// Widget is the GtkWidget to put in a container.
func (v *View) Widget() uintptr { return v.widget }

// Attach adds the view to a GtkContainer and shows it.
func (v *View) Attach(container uintptr) {
	gtk_container_add(container, v.widget)
	gtk_widget_show(v.widget)
}

func (v *View) SetSize(w, h int) { gtk_widget_set_size_request(v.widget, int32(w), int32(h)) }

// AddScript runs js at document start in every frame of pages loaded from now on.
func (v *View) AddScript(js string) {
	// WEBKIT_USER_CONTENT_INJECT_ALL_FRAMES = 0, WEBKIT_USER_SCRIPT_INJECT_AT_DOCUMENT_START = 0.
	us := webkit_user_script_new(js, 0, 0, 0, 0)
	webkit_user_content_manager_add_script(v.ucm, us)
	webkit_user_script_unref(us)
}

func (v *View) LoadURL(url string) { webkit_web_view_load_uri(v.widget, url) }

func (v *View) LoadHTML(html, baseURL string) {
	base := uintptr(0)
	if baseURL != "" {
		base = cstr(baseURL)
		defer g_free(base)
	}
	webkit_web_view_load_html(v.widget, html, base)
}

func (v *View) GoBack()    { webkit_web_view_go_back(v.widget) }
func (v *View) GoForward() { webkit_web_view_go_forward(v.widget) }
func (v *View) Reload()    { webkit_web_view_reload(v.widget) }
func (v *View) Stop()      { webkit_web_view_stop_loading(v.widget) }

func (v *View) CanGoBack() bool    { return webkit_web_view_can_go_back(v.widget) != 0 }
func (v *View) CanGoForward() bool { return webkit_web_view_can_go_forward(v.widget) != 0 }

func (v *View) URL() string { return goString(webkit_web_view_get_uri(v.widget)) }

// Eval runs js; done gets the value as text ("" for undefined) or the error message.
func (v *View) Eval(js string, done func(result, err string)) {
	mu.Lock()
	nextID++
	id := nextID
	pending[id] = func(res uintptr) {
		if v.disposed {
			return
		}
		perr := g_malloc(8)
		*(*uintptr)(cptr(perr)) = 0
		val := webkit_web_view_evaluate_javascript_finish(v.widget, res, perr)
		gerr := *(*uintptr)(cptr(perr))
		g_free(perr)
		if val == 0 {
			msg := "script failed"
			if gerr != 0 {
				msg = goString(*(*uintptr)(cptr(gerr + 8)))
				g_error_free(gerr)
			}
			done("", msg)
			return
		}
		out := ""
		if jsc_value_is_undefined(val) == 0 {
			out = takeString(jsc_value_to_string(val))
		}
		g_object_unref(val)
		done(out, "")
	}
	mu.Unlock()
	webkit_web_view_evaluate_javascript(v.widget, js, -1, 0, 0, 0, cbEval, id)
}

func (v *View) serve(req uintptr) {
	status, headers, body := 404, map[string]string(nil), []byte(nil)
	if v.Scheme != nil {
		status, headers, body = v.Scheme(goString(webkit_uri_scheme_request_get_uri(req)), goString(webkit_uri_scheme_request_get_http_method(req)))
	}
	data := uintptr(0)
	if len(body) > 0 {
		data = g_malloc(uintptr(len(body)))
		copy(unsafe.Slice((*byte)(cptr(data)), len(body)), body)
	}
	stream := g_memory_input_stream_new_from_data(data, len(body), gFree)
	resp := webkit_uri_scheme_response_new(stream, int64(len(body)))
	webkit_uri_scheme_response_set_status(resp, uint32(status), 0)
	hd := soup_message_headers_new(1) // SOUP_MESSAGE_HEADERS_RESPONSE
	for k, val := range headers {
		if k == "Content-Type" {
			webkit_uri_scheme_response_set_content_type(resp, val)
			continue
		}
		soup_message_headers_append(hd, k, val)
	}
	webkit_uri_scheme_response_set_http_headers(resp, hd)
	webkit_uri_scheme_request_finish_with_response(req, resp)
	g_object_unref(resp)
	g_object_unref(stream)
}

func (v *View) Dispose() {
	if v.disposed {
		return
	}
	v.disposed = true
	for i := 0; i < len(v.handlers); i += 2 {
		g_signal_handler_disconnect(v.handlers[i], v.handlers[i+1])
	}
	webkit_user_content_manager_unregister_script_message_handler(v.ucm, "gowt")
	gtk_widget_destroy(v.widget)
	g_object_unref(v.ctx)
	mu.Lock()
	delete(views, v.id)
	mu.Unlock()
}

// goString reads a NUL-terminated C string; 0 gives "".
func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(cptr(p + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(cptr(p)), n))
}

// takeString copies a g_malloc'ed C string and frees it.
func takeString(p uintptr) string {
	s := goString(p)
	if p != 0 {
		g_free(p)
	}
	return s
}

func cstr(s string) uintptr {
	p := g_malloc(uintptr(len(s) + 1))
	b := unsafe.Slice((*byte)(cptr(p)), len(s)+1)
	copy(b, s)
	b[len(s)] = 0
	return p
}

// cptr turns a C address into a pointer; reading it through a pointer variable keeps vet quiet, and
// the memory is C-owned so the garbage collector never sees it.
func cptr(p uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&p)) }
