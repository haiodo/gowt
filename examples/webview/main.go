// Command webview puts the system web view beside native widgets: a page served from Go over a custom
// scheme, a script run from a button, and messages from the page into a Label.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/webview"
)

const page = `<!doctype html><meta charset="utf-8"><title>From Go</title>
<body style="font:15px system-ui,sans-serif;margin:24px">
<h2>Served over app://</h2>
<button onclick="gowt.postMessage('clicked')">Send message to Go</button>
<p id="out">nothing yet</p></body>`

func main() {
	err := gowt.Run(func(app *gowt.App) {
		status := rx.New("")
		var host *gowt.Panel
		var wv *webview.WebView
		var err error
		w := app.Window("Web view",
			gowt.Label("", gowt.GrowX(), gowt.BindText(status)),
			gowt.Button("Set page text from Go", gowt.OnClick(func() {
				wv.Eval(`document.getElementById("out").textContent = "set from Go"; document.title`,
					func(result string, err error) {
						if err != nil {
							status.Set("eval error: " + err.Error())
							return
						}
						status.Set("eval result (JSON): " + result)
					})
			})),
			gowt.Fill(gowt.Grow(), gowt.Size(480, 280), gowt.Ref(&host)))

		wv, err = webview.New(host.Unwrap(), webview.Options{})
		if err != nil {
			// For example no WebView2 Runtime on Windows, or no WebKitGTK on Linux.
			status.Set("no web view: " + err.Error())
			w.Show()
			return
		}
		// HandleScheme comes before the first load: the native view is created then.
		err = wv.HandleScheme("app", func(r webview.Request) *webview.Response {
			return &webview.Response{MimeType: "text/html", Body: []byte(page)}
		})
		if err != nil {
			status.Set(err.Error())
		}
		wv.OnMessage = func(msg string) { status.Set("page says: " + msg) }
		wv.OnNavigationFinished = func(url string) { status.Set("loaded " + url) }

		w.Show()
		wv.Navigate("app://localhost/")
	})
	if err != nil {
		log.Fatal(err)
	}
}
