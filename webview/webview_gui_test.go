//go:build linux || darwin

package webview_test

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/webview"
)

func init() { runtime.LockOSThread() }

var mainq = make(chan func())

// The toolkit needs the main thread: the tests run on their own goroutine and hand the UI work back.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func onMain(f func()) {
	ok := make(chan struct{})
	mainq <- func() { defer close(ok); f() }
	<-ok
}

const indexHTML = `<title>first</title><body><a id=l href="app://host/two.html">two</a>
<script>gowt.postMessage("hello"); gowt.postMessage({a:1})</script>`

func TestWebView(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var started, finished, msgs, titles []string
	var failedURL string
	evals := map[string]string{}
	var cur string
	var err error
	onMain(func() {
		err = g.Run(func(app *g.App) {
			w := app.Window("webview")
			w.SetLayout(g.Fill{})
			host := w.Panel()
			host.SetLayout(g.Fill{})
			wv, e := webview.New(host.Unwrap(), webview.Options{Inspectable: true})
			if e != nil {
				t.Error(e)
				app.Quit()
				return
			}
			wv.HandleScheme("app", func(r webview.Request) *webview.Response {
				switch {
				case strings.HasSuffix(r.URL, "/index.html"):
					return &webview.Response{MimeType: "text/html", Body: []byte(indexHTML)}
				case strings.HasSuffix(r.URL, "/two.html"):
					return &webview.Response{MimeType: "text/html", Body: []byte("<title>second</title>two")}
				}
				return nil
			})
			wv.AddScript("window.__s = 42")
			wv.OnNavigationStarted = func(u string) { started = append(started, u) }
			wv.OnNavigationFinished = func(u string) { finished = append(finished, u) }
			wv.OnNavigationFailed = func(u string, _ error) { failedURL = u }
			wv.OnMessage = func(m string) { msgs = append(msgs, m) }
			wv.OnTitleChanged = func(s string) { titles = append(titles, s) }
			w.SetSize(400, 300)
			w.Show()

			eval := func(js string) {
				wv.Eval(js, func(r string, e error) {
					if e != nil {
						r = "ERR:" + e.Error()
					}
					evals[js] = r
				})
			}
			steps := []struct {
				run  func()
				done func() bool
			}{
				{func() { wv.Navigate("app://host/index.html") }, func() bool { return len(finished) == 1 && len(msgs) == 2 }},
				{func() {
					eval("window.__s")
					eval("1+1")
					eval("undefined")
					eval("throw new Error('boom')")
					eval("document.title")
				}, func() bool { return len(evals) == 5 }},
				{func() { wv.Navigate("app://host/two.html") }, func() bool { return len(finished) == 2 }},
				{func() { wv.GoBack() }, func() bool { return len(finished) == 3 }},
				{func() { cur = wv.URL(); wv.GoForward() }, func() bool { return len(finished) == 4 }},
				{func() { wv.Reload() }, func() bool { return len(finished) == 5 }},
				{func() { wv.SetHTML("<title>inline</title>x", "") }, func() bool { return len(finished) == 6 }},
				{func() { wv.Navigate("http://127.0.0.1:1/") }, func() bool { return failedURL != "" }},
			}
			i, deadline := 0, time.Now().Add(30*time.Second)
			var tick func()
			tick = func() {
				if i < len(steps) && steps[i].run != nil {
					steps[i].run()
					steps[i].run = nil
				}
				if i < len(steps) && steps[i].done() {
					i++
					if i < len(steps) {
						steps[i].run()
						steps[i].run = nil
					}
				}
				if i == len(steps) || time.Now().After(deadline) {
					if i < len(steps) {
						t.Errorf("stuck at step %d: started=%v finished=%v msgs=%v", i, started, finished, msgs)
					}
					w.Close()
					return
				}
				app.After(20*time.Millisecond, tick)
			}
			tick()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"window.__s": "42", "1+1": "2", "undefined": "", "document.title": `"first"`}
	for js, r := range want {
		if evals[js] != r {
			t.Errorf("eval %s = %q, want %q", js, evals[js], r)
		}
	}
	if !strings.HasPrefix(evals["throw new Error('boom')"], "ERR:") {
		t.Errorf("script error not reported: %q", evals["throw new Error('boom')"])
	}
	if len(msgs) < 2 || msgs[0] != "hello" || msgs[1] != `{"a":1}` {
		t.Errorf("messages %q", msgs)
	}
	if cur != "app://host/index.html" {
		t.Errorf("URL after GoBack = %q", cur)
	}
	if len(started) < 6 || !strings.Contains(strings.Join(titles, ","), "second") {
		t.Errorf("started=%v titles=%v", started, titles)
	}
}
