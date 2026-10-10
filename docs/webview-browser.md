# Web view and Browser

Examples: [`examples/webview`](../examples/webview), [`examples/browser`](../examples/browser).

Two packages put a web page next to native widgets. Both use the system engine, so no browser is bundled.

| | `webview` | `browser` |
|---|---|---|
| What | small hand-written Go API | SWT's `Browser` widget, translated, with `webview` as its engine |
| Engine | WKWebView (macOS), WebKitGTK 4.1 (Linux), WebView2 (Windows) | the same |
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

- Browser does not support `OpenWindow`, `VisibilityWindow`, `CloseWindow`, `Authentication` and `StatusText` events, post data and headers in `SetUrl`, `SetJavascriptEnabled`, and the cookie statics.
- `Evaluate`, `Execute` and `GetText` called from inside a `BrowserFunction` throw at once.
- History queries, `Stop`, the navigation policy that lets `LocationListener.Changing` cancel, and BrowserFunction calls are implemented on macOS only. Elsewhere `Back`/`Forward` always go, `Changing` cannot cancel and BrowserFunctions return `undefined`.
- Browser tests: 173 of 209 pass on macOS, with 6 failures not diagnosed.
- Linux: WebKitGTK does not report which frame navigates, so the navigation policy always gets `mainFrame=true`.
- Windows: the WebView2 code (COM callbacks written in Go, no `WebView2Loader.dll`) was never run on a real Windows; CrossOver had no WebView2 Runtime.
- The examples in this repository were type-checked, not run, for all three OSes.
