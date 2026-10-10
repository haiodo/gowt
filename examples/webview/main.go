// Command webview puts the system web view beside native widgets: a page served from Go over a custom
// scheme, a script run from a button, and messages from the page into a Label.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/webview"
)

const page = `<!doctype html><meta charset="utf-8"><title>From Go</title>
<body style="font:15px system-ui,sans-serif;margin:24px">
<h2>Served over app://</h2>
<button onclick="gowt.postMessage('clicked')">Send message to Go</button>
<p id="out">nothing yet</p></body>`

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Web view")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 8, Spacing: 8})
		fillX := gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true})
		status := w.Label("", fillX)
		button := w.Button("Set page text from Go", nil)

		host := w.Panel(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true, Width: 480, Height: 280}))
		host.SetLayout(gowt.Fill{})

		wv, err := webview.New(host.Unwrap(), webview.Options{})
		if err != nil {
			// For example no WebView2 Runtime on Windows, or no WebKitGTK on Linux.
			status.SetText("no web view: " + err.Error())
			w.Show()
			return
		}
		// HandleScheme comes before the first load: the native view is created then.
		err = wv.HandleScheme("app", func(r webview.Request) *webview.Response {
			return &webview.Response{MimeType: "text/html", Body: []byte(page)}
		})
		if err != nil {
			status.SetText(err.Error())
		}
		wv.OnMessage = func(msg string) { status.SetText("page says: " + msg) }
		wv.OnNavigationFinished = func(url string) { status.SetText("loaded " + url) }

		button.OnClick(func() {
			wv.Eval(`document.getElementById("out").textContent = "set from Go"; document.title`,
				func(result string, err error) {
					if err != nil {
						status.SetText("eval error: " + err.Error())
						return
					}
					status.SetText("eval result (JSON): " + result)
				})
		})
		w.Show()
		wv.Navigate("app://localhost/")
	})
	if err != nil {
		log.Fatal(err)
	}
}
