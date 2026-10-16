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
	var tree *g.TreeView
	var button *g.ButtonView
	var label *g.LabelView
	var host *g.Panel
	w := app.Window("WebView demo",
		g.Grid(2, g.Grow(),
			g.Tree(g.GrowY(), g.Size(160, 0), g.Ref(&tree)),
			g.VBox(g.Grow(),
				g.Button("Eval in page", g.Ref(&button)),
				g.Label("Messages from the page appear here", g.Wrap(), g.GrowX(), g.Ref(&label)),
				g.Fill(g.Grow(), g.Ref(&host)),
			),
		),
	)
	for _, p := range []string{"index.html", "about.html"} {
		tree.Node(p)
	}

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
