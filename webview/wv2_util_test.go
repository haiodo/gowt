package webview

import (
	"strings"
	"testing"
	"unsafe"
)

func TestVtableLayout(t *testing.T) {
	tbl, keep := vtable([]uintptr{11, 22, 33, 44})
	for i, want := range []uintptr{11, 22, 33, 44} {
		got := *(*uintptr)(unsafe.Add(*(*unsafe.Pointer)(unsafe.Pointer(&tbl)), uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if got != want {
			t.Fatalf("slot %d = %d, want %d", i, got, want)
		}
	}
	keep[0] = 1 // the table is a copy
	if _, k2 := vtable(keep); &k2[0] == &keep[0] {
		t.Fatal("vtable aliases its input")
	}
}

func TestParseGUID(t *testing.T) {
	g := parseGUID("2fde08a8-1e9a-4766-8c05-95a9ceb9d1c5")
	want := [16]byte{0xa8, 0x08, 0xde, 0x2f, 0x9a, 0x1e, 0x66, 0x47, 0x8c, 0x05, 0x95, 0xa9, 0xce, 0xb9, 0xd1, 0xc5}
	if g != want {
		t.Fatalf("got % x", g)
	}
}

func TestStrings(t *testing.T) {
	u := utf16Z("a\u00e9\U0001F600")
	if u[len(u)-1] != 0 {
		t.Fatal("no terminator")
	}
	if s := goString(uintptr(unsafe.Pointer(&u[0]))); s != "a\u00e9\U0001F600" {
		t.Fatalf("round trip: %q", s)
	}
	if goString(0) != "" {
		t.Fatal("nil string")
	}
}

func TestEvalResult(t *testing.T) {
	for _, c := range []struct{ raw, res, err string }{
		{`{"v":"42"}`, "42", ""},
		{`{"v":"\"x\""}`, `"x"`, ""},
		{`{}`, "", ""},
		{`{"e":"ReferenceError: x"}`, "", "ReferenceError: x"},
	} {
		res, err := evalResult(c.raw)
		if res != c.res || (err != nil) != (c.err != "") || (err != nil && err.Error() != c.err) {
			t.Errorf("%s: %q, %v", c.raw, res, err)
		}
	}
	if _, err := evalResult("null"); err != nil {
		t.Errorf("null: %v", err)
	}
}

func TestMessageText(t *testing.T) {
	if messageText(`"hi"`) != "hi" || messageText(`{"a":1}`) != `{"a":1}` {
		t.Fatal("messageText")
	}
}

func TestResponseHeaders(t *testing.T) {
	h := responseHeaders(&Response{MimeType: "text/html", Headers: map[string]string{"X-A": "1", "content-type": "x"}})
	if h != "X-A: 1\r\nContent-Type: text/html\r\n" {
		t.Fatalf("%q", h)
	}
}

func TestSchemeOf(t *testing.T) {
	if schemeOf("APP://x/y") != "app" || schemeOf("/x:y") != "" || schemeOf("nocolon") != "" {
		t.Fatal("schemeOf")
	}
}

func TestUserDataDir(t *testing.T) {
	if got := userDataDir(`C:\Users\a\AppData\Local`, `C:\Apps\Foo.exe`); got != `C:\Users\a\AppData\Local\Foo.WebView2` {
		t.Fatal(got)
	}
	if userDataDir("", `C:\Foo.exe`) != "" {
		t.Fatal("empty LOCALAPPDATA")
	}
}

func fakeEnv(reg map[string]string, files ...string) wv2Env {
	have := map[string]bool{}
	for _, f := range files {
		have[f] = true
	}
	return wv2Env{
		reg:    func(root, key, name string) (string, bool) { v, ok := reg[root+"|"+key+"|"+name]; return v, ok },
		exists: func(p string) bool { return have[p] },
		glob: func(pat string) []string {
			var out []string
			pre := pat[:strings.Index(pat, "*")]
			for _, f := range files {
				if strings.HasPrefix(f, pre) {
					out = append(out, f)
				}
			}
			return out
		},
		getenv: func(n string) string {
			if n == "ProgramFiles(x86)" {
				return `C:\PF86`
			}
			return ""
		},
	}
}

func TestFindRuntimeDLL(t *testing.T) {
	const dll = `\x64\EmbeddedBrowserWebView.dll`
	key := `WOW6432Node\` + wv2ClientsKey
	// EBWebView value names the directory with the arch subdirectory.
	e := fakeEnv(map[string]string{`HKLM|` + key + `|EBWebView`: `C:\W\1.2\EBWebView`}, `C:\W\1.2\EBWebView`+dll)
	if p, err := findRuntimeDLL(e, "x64"); err != nil || p != `C:\W\1.2\EBWebView`+dll {
		t.Fatal(p, err)
	}
	// location + pv, per-user install.
	hkcu := `HKCU|` + wv2ClientsKey
	e = fakeEnv(map[string]string{hkcu + `|location`: `C:\U`, hkcu + `|pv`: `9.9`}, `C:\U\9.9\EBWebView`+dll)
	if p, err := findRuntimeDLL(e, "x64"); err != nil || p != `C:\U\9.9\EBWebView`+dll {
		t.Fatal(p, err)
	}
	// directory scan picks the numerically newest version.
	base := `C:\PF86\Microsoft\EdgeWebView\Application\`
	e = fakeEnv(nil, base+`99.0.1\EBWebView`+dll, base+`100.0.1\EBWebView`+dll)
	if p, err := findRuntimeDLL(e, "x64"); err != nil || p != base+`100.0.1\EBWebView`+dll {
		t.Fatal(p, err)
	}
	// nothing installed: the error says what to install.
	if _, err := findRuntimeDLL(fakeEnv(nil), "x64"); err == nil || !strings.Contains(err.Error(), "WebView2 Runtime is not installed") {
		t.Fatal(err)
	}
}
