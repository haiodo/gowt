package browser

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestNormalizeAndEscapeURL(t *testing.T) {
	for in, want := range map[string]string{
		"/tmp/a b.html":          "file:///tmp/a%20b.html",
		"example.com/x":          "http://example.com/x",
		"https://h/p?q=a b#frag": "https://h/p?q=a%20b#frag",
		"about:blank":            "about:blank",
		"http://h/%41é":          "http://h/%41%C3%A9",
	} {
		if got := escapeURL(normalizeURL(in)); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestDecodeEvaluate(t *testing.T) {
	for _, c := range []struct {
		in   string
		want any
		code int32
	}{
		{`{"v":123}`, 123.0, 0},
		{`{"v":"s"}`, "s", 0},
		{`{"v":true}`, true, 0},
		{`{"v":null}`, nil, 0},
		{`{"v":[1,"a",[true,null]]}`, []any{1.0, "a", []any{true, nil}}, 0},
		{`{"e":"boom"}`, nil, swt.ERROR_FAILED_EVALUATE},
		{`{"e":"\u0001invalid"}`, nil, swt.ERROR_INVALID_RETURN_VALUE},
		{`not json`, nil, swt.ERROR_FAILED_EVALUATE},
	} {
		got, code, _ := decodeEvaluate(c.in)
		if code != c.code || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s -> %#v, %d; want %#v, %d", c.in, got, code, c.want, c.code)
		}
	}
}

func TestEncodeJavaValue(t *testing.T) {
	for _, c := range []struct {
		in   any
		want string
		ok   bool
	}{
		{nil, "null", true},
		{int32(42), "42", true},
		{1.5, "1.5", true},
		{"a\t\"b\\ß", `"a\t\"b\\ß"`, true},
		{true, "true", true},
		{[]any{int32(1), "x", nil}, `[1,"x",null]`, true},
		{struct{}{}, "", false},
		{[]any{struct{}{}}, "", false},
	} {
		got, ok := encodeJavaValue(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("%#v -> %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

type fnImpl struct {
	*BrowserFunction
	fn func([]any) any
}

func (f *fnImpl) function_(args []any) any { return f.fn(args) }

func newTestFunction(index int32, token string, fn func([]any) any) *BrowserFunction {
	f := &BrowserFunction{index: index, token: token, functionString: "define" + token + "();"}
	f.impl = &fnImpl{f, fn}
	return f
}

func TestHandleCall(t *testing.T) {
	w := &webViewBrowser{WebBrowser: newWebBrowser()}
	var got []any
	w.functions.Put(int32(2), newTestFunction(2, "b", func(a []any) any { got = a; return int32(7) }))
	w.functions.Put(int32(1), newTestFunction(1, "a", func([]any) any { panic("kaput") }))

	if r := w.handleCall(`["init"]`); r != "defineA();\n" && r != "definea();\ndefineb();\n" {
		t.Errorf("init = %q", r)
	}
	if r := w.handleCall(`[2,"b",[1,"x",[true]]]`); r != "7" || !reflect.DeepEqual(got, []any{1.0, "x", []any{true}}) {
		t.Errorf("call = %q, args %#v", r, got)
	}
	if r := w.handleCall(`[1,"a",[]]`); !strings.HasPrefix(r, `"`+WebBrowserERROR_ID) || !strings.Contains(r, "kaput") {
		t.Errorf("panic reply = %q", r)
	}
	for _, bad := range []string{`[2,"wrong",[]]`, `[9,"b",[]]`, `[2,"b"]`, `nonsense`, ``} {
		if r := w.handleCall(bad); r != "null" {
			t.Errorf("%q -> %q, want null", bad, r)
		}
	}
}

// The JS side runs in node: arrays and scalars come back, a Date is refused, errors carry their message.
func TestEvaluateWrapperInNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	for script, want := range map[string]string{
		"return 1+2":                         `{"v":3}`,
		"var x = 1; return 'hello'":          `{"v":"hello"}`,
		"return new Array(1,2,3)":            `{"v":[1,2,3]}`,
		"return [1,'a',true,null]":           `{"v":[1,"a",true,null]}`,
		"return null":                        `{"v":null}`,
		"1+1":                                `{"v":null}`,
		"return new Date()":                  `{"e":"\u0001invalid"}`,
		"return undefinedFn()":               `{"e":"undefinedFn is not defined"}`,
		"throw new Error('x')":               `{"e":"x"}`,
		"return (":                           "",
		"return 'a\\u2028b' // line comment": `{"v":"a b"}`,
	} {
		cmd := exec.Command(node, "-e", `console.log(JSON.stringify((0,eval)(require("fs").readFileSync(0,"utf8"))))`)
		cmd.Stdin = strings.NewReader(evaluateWrapper(script))
		out, err := cmd.CombinedOutput()
		got := strings.TrimSpace(string(out))
		if want == "" {
			if !strings.Contains(got, `"e":"`) {
				t.Errorf("%q: %s (%v), want an error result", script, got, err)
			}
			continue
		}
		if got != want {
			t.Errorf("%q: %s (%v), want %s", script, got, err, want)
		}
	}
}

// The page-side preamble runs the init script and routes callJava through window.gowt.call.
func TestPreambleInNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	js := `global.window = {gowt: {call: m => m === '["init"]' ? 'window.foo=41' : JSON.stringify(JSON.parse(m)[2][0] + 1)}};
window.top = window;
(0,eval)(require("fs").readFileSync(0,"utf8"));
console.log(window.foo, window.callJava(1, "t", [1]))`
	cmd := exec.Command(node, "-e", js)
	cmd.Stdin = strings.NewReader(preamble)
	out, err := cmd.CombinedOutput()
	if got := strings.TrimSpace(string(out)); got != "41 2" {
		t.Errorf("preamble: %q (%v), want \"41 2\"", got, err)
	}
}

func TestExecuteWrapperInNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	for script, want := range map[string]string{
		"var x = 1;":      "true",
		"({a: {}}).a":     "true",
		"throw new Error": "false",
		"var = ;":         "false",
	} {
		cmd := exec.Command(node, "-e", `console.log(JSON.stringify((0,eval)(require("fs").readFileSync(0,"utf8"))))`)
		cmd.Stdin = strings.NewReader(executeWrapper(script))
		out, _ := cmd.CombinedOutput()
		if got := strings.TrimSpace(string(out)); got != want {
			t.Errorf("%q: %s, want %s", script, got, want)
		}
	}
}
