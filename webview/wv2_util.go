package webview

// Platform-independent parts of the WebView2 backend, kept out of the windows-only files so that they are
// unit-tested on any host.

import (
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"
)

const errNoRuntime = "webview: the WebView2 Runtime is not installed (Windows 10 needs the Evergreen runtime from " +
	"https://developer.microsoft.com/microsoft-edge/webview2/, Windows 11 has it)"

const wv2ClientsKey = `Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
const wv2DLL = "EmbeddedBrowserWebView.dll"

// wv2Env is what the runtime discovery needs from the machine; tests fake it.
type wv2Env struct {
	// reg reads a REG_SZ: root is "HKLM" or "HKCU", key is relative to SOFTWARE.
	reg    func(root, key, name string) (string, bool)
	exists func(path string) bool
	glob   func(pattern string) []string
	getenv func(name string) string
}

// findRuntimeDLL returns the path of EmbeddedBrowserWebView.dll of the installed Evergreen runtime, the library
// that exports CreateWebViewEnvironmentWithOptionsInternal (the part of WebView2Loader.dll that matters).
// arch is the directory the runtime names its builds by: "x64" or "arm64".
func findRuntimeDLL(env wv2Env, arch string) (string, error) {
	var tried []string
	try := func(p string) (string, bool) {
		if p == "" {
			return "", false
		}
		tried = append(tried, p)
		if env.exists(p) {
			return p, true
		}
		return "", false
	}
	// The Clients key is written by the runtime installer: per machine (32-bit view on 64-bit Windows) or per user.
	for _, k := range []struct{ root, key string }{
		{"HKLM", `WOW6432Node\` + wv2ClientsKey}, {"HKLM", wv2ClientsKey}, {"HKCU", wv2ClientsKey},
	} {
		if dir, ok := env.reg(k.root, k.key, "EBWebView"); ok {
			for _, d := range []string{dir, dir + `\` + arch} {
				if p, ok := try(d + `\` + wv2DLL); ok {
					return p, nil
				}
			}
		}
		loc, ok1 := env.reg(k.root, k.key, "location")
		pv, ok2 := env.reg(k.root, k.key, "pv")
		if ok1 && ok2 {
			if p, ok := try(loc + `\` + pv + `\EBWebView\` + arch + `\` + wv2DLL); ok {
				return p, nil
			}
		}
	}
	// Last resort: the default install directory, the newest version.
	for _, v := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		pf := env.getenv(v)
		if pf == "" {
			continue
		}
		found := env.glob(pf + `\Microsoft\EdgeWebView\Application\*\EBWebView\` + arch + `\` + wv2DLL)
		sort.Slice(found, func(i, j int) bool { return versionLess(versionOf(found[j]), versionOf(found[i])) })
		if len(found) > 0 {
			return found[0], nil
		}
		tried = append(tried, pf+`\Microsoft\EdgeWebView\Application\*`)
	}
	return "", errors.New(errNoRuntime + "; looked in: " + strings.Join(tried, ", "))
}

// versionOf takes the version directory out of ...\Application\<version>\EBWebView\...
func versionOf(p string) string {
	parts := strings.Split(strings.ReplaceAll(p, "/", `\`), `\`)
	for i, s := range parts {
		if s == "EBWebView" && i > 0 {
			return parts[i-1]
		}
	}
	return ""
}

func versionLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		x, _ := strconv.Atoi(as[i])
		y, _ := strconv.Atoi(bs[i])
		if x != y {
			return x < y
		}
	}
	return len(as) < len(bs)
}

// utf16Z is s as a NUL-terminated UTF-16 string; the slice must stay reachable while the pointer is used.
func utf16Z(s string) []uint16 { return append(utf16.Encode([]rune(s)), 0) }

// goString reads the NUL-terminated UTF-16 string at p ("" for 0).
func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	q := *(*unsafe.Pointer)(unsafe.Pointer(&p))
	n := 0
	for *(*uint16)(unsafe.Add(q, 2*n)) != 0 {
		n++
	}
	return string(utf16.Decode(unsafe.Slice((*uint16)(q), n)))
}

// evalWrapper makes ExecuteScript report script errors: the runtime gives "null" for a throw.
func evalWrapper(js string) string {
	return "(function(){try{return {v:" + js + "}}catch(e){return {e:String(e&&e.stack||e)}}})()"
}

// evalResult decodes the JSON ExecuteScript returns for evalWrapper's object.
func evalResult(raw string) (string, error) {
	var r struct {
		V *string `json:"v"`
		E *string `json:"e"`
	}
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return "", err
	}
	if r.E != nil {
		return "", errors.New(*r.E)
	}
	if r.V == nil {
		return "", nil
	}
	return *r.V, nil
}

// messageText turns WebMessageAsJson into what the darwin backend delivers: a JSON string literal is the string
// itself, anything else stays JSON text.
func messageText(j string) string {
	var s string
	if json.Unmarshal([]byte(j), &s) == nil {
		return s
	}
	return j
}

// wv2Bridge is the page-side half of window.gowt.postMessage.
const wv2Bridge = `window.gowt={postMessage:function(m){window.chrome.webview.postMessage(typeof m==="string"?m:JSON.stringify(m))}};`

// responseHeaders is the CRLF block CreateWebResourceResponse takes.
func responseHeaders(r *Response) string {
	var b strings.Builder
	keys := make([]string, 0, len(r.Headers))
	for k := range r.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if r.MimeType != "" && strings.EqualFold(k, "Content-Type") {
			continue
		}
		b.WriteString(k + ": " + r.Headers[k] + "\r\n")
	}
	if r.MimeType != "" {
		b.WriteString("Content-Type: " + r.MimeType + "\r\n")
	}
	return b.String()
}

func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Unknown"
}

// schemeOf is the scheme of a URL without parsing the rest ("" if there is none).
func schemeOf(u string) string {
	s, _, ok := strings.Cut(u, ":")
	if !ok || s == "" || strings.ContainsAny(s, "/?#") {
		return ""
	}
	return strings.ToLower(s)
}

// userDataDir is where the runtime keeps profile data: the default (next to the exe) fails for installs in
// Program Files, so it goes to %LOCALAPPDATA%.
func userDataDir(localAppData, exe string) string {
	if localAppData == "" {
		return ""
	}
	base := path.Base(strings.ReplaceAll(exe, `\`, "/"))
	return localAppData + `\` + strings.TrimSuffix(base, ".exe") + ".WebView2"
}

// vtable lays out slots as a COM vtable; the memory is kept by the returned slice.
func vtable(slots []uintptr) (tbl uintptr, keep []uintptr) {
	keep = append([]uintptr(nil), slots...)
	p := unsafe.Pointer(&keep[0])
	return *(*uintptr)(unsafe.Pointer(&p)), keep
}

// parseGUID is the in-memory layout of a GUID given as "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
func parseGUID(s string) (g [16]byte) {
	h := strings.ReplaceAll(s, "-", "")
	var b [16]byte
	for i := range b {
		v, _ := strconv.ParseUint(h[2*i:2*i+2], 16, 8)
		b[i] = byte(v)
	}
	// Data1..Data3 are little-endian in memory.
	g[0], g[1], g[2], g[3] = b[3], b[2], b[1], b[0]
	g[4], g[5] = b[5], b[4]
	g[6], g[7] = b[7], b[6]
	copy(g[8:], b[8:])
	return
}
