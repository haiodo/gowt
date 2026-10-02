//go:build darwin

package cocoa

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ebitengine/purego"
)

// WebKit needs the main thread: TestMain keeps it for onMain and runs the tests in another goroutine.
func init() { runtime.LockOSThread() }

var mainQ = make(chan func())

func TestMain(m *testing.M) {
	go func() { os.Exit(m.Run()) }()
	for f := range mainQ {
		f()
	}
}

func onMain(f func()) {
	done := make(chan struct{})
	mainQ <- func() { f(); close(done) }
	<-done
}

// TestWKViewHeadless uses a view in no window and an app that never activates: nothing is shown.
func TestWKViewHeadless(t *testing.T) {
	if os.Getenv("GOWT_WK_HEADLESS") == "" {
		t.Skip("set GOWT_WK_HEADLESS=1: starts WebContent processes")
	}
	onMain(func() {
		objcInit()
		msg(msg(class("NSApplication"), "sharedApplication"), "setActivationPolicy:", 2) // Prohibited
		var runLoop func(mode uintptr, sec float64, ret bool) int32
		purego.RegisterLibFunc(&runLoop, purego.RTLD_DEFAULT, "CFRunLoopRunInMode")
		mode := nsString("kCFRunLoopDefaultMode")

		parent := msg(msg(class("NSView"), "alloc"), "init")
		v := NewWKView(WKConfig{Parent: parent, Frame: NSRect{Width: 300, Height: 200}, Schemes: []string{"gowt"}})
		defer v.Dispose()
		got := map[string]string{}
		v.Message = func(b string) { got["msg:"+b] = "" }
		v.NavFinished = func(u string) { got["fin:"+u] = "" }
		v.TitleChanged = func(s string) { got["title:"+s] = "" }
		v.Scheme = func(url, method string) (int, map[string]string, []byte) {
			if url != "gowt://app/index.html" {
				return 404, nil, nil
			}
			return 200, map[string]string{"Content-Type": "text/html"},
				[]byte(`<title>scheme page</title><script>gowt.postMessage({a:1})</script>`)
		}
		wait := func(what string, ok func() bool) {
			for deadline := time.Now().Add(10 * time.Second); !ok(); {
				if time.Now().After(deadline) {
					t.Fatalf("timeout waiting for %s; got %v", what, got)
				}
				runLoop(mode, 0.05, false)
			}
		}
		has := func(k string) func() bool { return func() bool { _, ok := got[k]; return ok } }

		v.LoadHTML(`<title>html page</title><script>gowt.postMessage("hello")</script>`, "")
		wait("SetHTML message", has("msg:hello"))
		wait("title", has("title:html page"))

		var res, errs string
		fin := false
		v.Eval(`JSON.stringify(1+2)`, func(r, e string) { res, errs, fin = r, e, true })
		wait("eval", func() bool { return fin })
		if res != "3" || errs != "" {
			t.Fatalf("eval = %q, %q", res, errs)
		}
		fin = false
		v.Eval(`throw new Error("boom")`, func(r, e string) { res, errs, fin = r, e, true })
		wait("eval error", func() bool { return fin })
		if errs == "" {
			t.Fatalf("eval error: no error, result %q", res)
		}

		v.LoadURL("gowt://app/index.html")
		wait("scheme message", has(`msg:{"a":1}`))
		wait("scheme title", has("title:scheme page"))
		wait("scheme finish", has("fin:gowt://app/index.html"))
	})
}

// A global block made by NewBlock is invoked by Foundation itself (synchronously, on this thread).
func TestBlockEnumerate(t *testing.T) {
	onMain(func() {
		var seen []uintptr
		arr := msg(class("NSArray"), "arrayWithObject:", nsString("x"))
		msg(arr, "enumerateObjectsUsingBlock:", NewBlock(3, func(a []uintptr) { seen = append(seen, a[0], a[1]) }))
		if len(seen) != 2 || goString(seen[0]) != "x" || seen[1] != 0 {
			t.Fatalf("block args = %v", seen)
		}
	})
}
