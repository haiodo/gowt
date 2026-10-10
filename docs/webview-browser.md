# Web view and Browser

Examples: [`examples/webview`](../examples/webview), [`examples/browser`](../examples/browser).

Two packages put a web page next to native widgets. Both use the system engine, so no browser is bundled.

| | `webview` | `browser` |
|---|---|---|
| What | small hand-written Go API | SWT's `Browser` widget, translated, with `webview` as its engine |
| Engine | WKWebView (macOS), WebKitGTK 4.1 (Linux), WebView2 (Windows, never run on real Windows) | the same |
| Use it for | new code | porting SWT code, or SWT's listener model |

## webview

```go
host := w.Panel()
host.SetLayout(gowt.Fill{})
wv, err := webview.New(host.Unwrap(), webview.Options{})
if err != nil { /* engine missing */ }
wv.HandleScheme("app", func(r webview.Request) *webview.Response {
	return &webview.Response{MimeType: "text/html", Body: page}
})
wv.OnMessage = func(msg string) { /* page called window.gowt.postMessage(...) */ }
w.Show()
wv.Navigate("app://localhost/")
```

- The parent is a plain composite: pass `panel.Unwrap()`. Give the panel a `Fill{}` layout so the view fills it.
- `SetCallHandler(f func(msg string) string) bool` sets a synchronous page-to-Go call (the page calls `window.gowt.call`); it is what `BrowserFunction` uses. `SetNavigationPolicy(f func(url string, mainFrame bool) bool) bool` lets Go cancel a navigation. Both return false if the engine lacks the feature.
- `HandleScheme(scheme, handler)` serves pages from Go (an `embed.FS` or a map), with no local server. `AddScript(js)` runs before the page's own scripts. The native view is created at the first load, so call both before `Navigate`/`SetHTML`.
- `Navigate(url)`, `SetHTML(html, baseURL)`, `Reload`, `Stop`, `GoBack`, `GoForward`, `URL()`.
- `Eval(js, func(resultJSON string, err error))` is asynchronous; the result is JSON text.
- Page to Go: the page calls `window.gowt.postMessage(x)`; Go sees it in `OnMessage` (non-strings arrive as JSON).
- Events are fields: `OnNavigationStarted`, `OnNavigationFinished`, `OnNavigationFailed`, `OnTitleChanged`.
- `New` returns an error when the engine is missing: the WebView2 Runtime on Windows (preinstalled on Windows 11), WebKitGTK on Linux. Show a message instead of failing.
- Everything runs on the UI thread, and callbacks run there too.
- `Options{Inspectable: true}` lets Safari's Web Inspector attach (macOS 13.3+).

A larger program is `cmd/webviewdemo` (a Tree beside a view, pages from `embed.FS`) and `cmd/minibrowser` (tabs of views).

## Browser

```go
b := browser.NewBrowser(w.AsComposite(), swt.NONE)
b.SetUrl("https://example.com")
b.SetText("<h2>Hello</h2>")
b.AddLocationListener(browser.LocationListenerChangedAdapter(func(e *browser.LocationEvent) { ... }))
```

`Browser` takes an swt composite as parent; every gowt container offers `AsComposite()`. Methods follow SWT: `SetUrl`, `SetText`, `Evaluate`, `Execute`, `GetText`, `Back`, `Forward`, `Refresh`, `Stop`, and the listeners for location, progress and title. `Evaluate`, `Execute` and `GetText` are synchronous: they run the event loop until the page answers (5 s limit). `BrowserFunction` exposes a Go function to the page.

## Known gaps

- Browser does not support the `OpenWindow`, `VisibilityWindow`, `CloseWindow`, `Authentication` and `StatusText` events, post data and headers in `SetUrl`, `SetJavascriptEnabled`, and the cookie statics.
- `Evaluate`, `Execute` and `GetText` called from inside a `BrowserFunction` throw at once.
- History queries, `Stop`, the navigation policy that lets `LocationListener.Changing` cancel, and the call handler behind `BrowserFunction` (`WebView.SetCallHandler`) are implemented for all three engines (`webview_darwin.go`, `webview_linux.go`, `webview_windows.go`).
- Browser tests (`tests/expected*.txt`): macOS 173 pass and 20 fail of 209, Linux 175 pass and 20 fail. On Windows all 209 Browser tests fail in the CrossOver bottle because it has no WebView2 Runtime, so Browser results exist only for macOS and Linux.
- Linux: WebKitGTK does not report which frame navigates, so the navigation policy always gets `mainFrame=true`.
- Windows: the WebView2 code (COM callbacks written in Go, no `WebView2Loader.dll`) has never run on a real Windows.
- The examples in this repository were type-checked for the three OSes but not run.
