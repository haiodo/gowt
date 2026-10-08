// Command webviewdemo shows a Tree beside a system web view: the pages come from an embed.FS over
// the app:// scheme, a button evaluates JS in the page, and page messages land in a Label.
package main

import (
	"embed"
	"fmt"
	"log"
	"mime"
	"path"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/webview"
)

//go:embed assets
var assets embed.FS

func main() {
	// Without it macOS lists the process (and WebKit's helpers) as "SWT".
	g.SetAppName("WebView Demo")
	if err := g.Run(build); err != nil {
		log.Fatal(err)
	}
}

func build(app *g.App) {
	both := g.Cell(g.GridCell{Align: g.AlignFill, VAlign: g.AlignFill, GrowX: true, GrowY: true})
	w := app.Window("WebView demo")
	w.SetLayout(g.Grid{Columns: 2, Margin: 5, Spacing: 5})

	tree := w.Tree(g.Border(), g.Cell(g.GridCell{VAlign: g.AlignFill, GrowY: true, Width: 160}))
	for _, p := range []string{"index.html", "about.html"} {
		tree.Node(p)
	}

	right := w.Panel(both)
	right.SetLayout(g.Grid{Columns: 1, Margin: 5, Spacing: 5})
	button := right.Button("Eval in page", nil)
	label := right.Label("Messages from the page appear here", g.Wrap(), g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true}))

	host := right.Panel(both)
	host.SetLayout(g.Fill{})
	wv, err := webview.New(host.Unwrap(), webview.Options{Inspectable: true})
	if err != nil {
		fmt.Println(err)
		app.Quit()
		return
	}
	wv.HandleScheme("app", func(r webview.Request) *webview.Response {
		name := path.Join("assets", path.Base(r.URL))
		body, err := assets.ReadFile(name)
		if err != nil {
			return nil
		}
		return &webview.Response{MimeType: mime.TypeByExtension(path.Ext(name)), Body: body}
	})
	wv.OnMessage = func(m string) { label.SetText("Message: " + m) }
	wv.OnTitleChanged = func(t string) { w.SetTitle("WebView demo - " + t) }
	wv.OnNavigationFailed = func(u string, err error) { label.SetText("Navigation failed: " + u + ": " + err.Error()) }

	button.OnClick(func() {
		wv.Eval(`document.getElementById("out").textContent = "Eval from Go at " + new Date().toLocaleTimeString(); document.title`,
			func(result string, err error) {
				if err != nil {
					label.SetText("Eval error: " + err.Error())
					return
				}
				label.SetText("Eval result: " + result)
			})
	})
	tree.OnSelect(func(n *g.Node) {
		if n != nil {
			wv.Navigate("app://localhost/" + n.Text())
		}
	})

	w.SetSize(800, 520)
	w.Show()
	wv.Navigate("app://localhost/index.html")
}
