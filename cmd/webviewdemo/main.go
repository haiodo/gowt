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
	"github.com/haiodo/gowt/rx"
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
	var wv *webview.WebView
	var host *g.Panel
	label := rx.New("")
	w := app.Window("WebView demo",
		g.Grid(2, g.Grow(),
			g.Tree(g.GrowY(), g.Size(160, 0),
				g.Init(func(t *g.TreeView) {
					for _, p := range []string{"index.html", "about.html"} {
						t.Node(p)
					}
				}),
				g.OnSelect(func(n *g.Node) {
					if n != nil {
						wv.Navigate("app://localhost/" + n.Text())
					}
				})),
			g.VBox(g.Grow(),
				g.Button("Eval in page", g.OnClick(func() {
					wv.Eval(`document.getElementById("out").textContent = "Eval from Go at " + new Date().toLocaleTimeString(); document.title`,
						func(result string, err error) {
							if err != nil {
								label.Set("Eval error: " + err.Error())
								return
							}
							label.Set("Eval result: " + result)
						})
				})),
				g.Label("Messages from the page appear here", g.Wrap(), g.GrowX(), g.BindText(label)),
				g.Fill(g.Grow(), g.Ref(&host)))))

	wv, err := webview.New(host, webview.Options{Inspectable: true})
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
	wv.OnMessage = func(m string) { label.Set("Message: " + m) }
	wv.OnTitleChanged = func(t string) { w.SetTitle("WebView demo - " + t) }
	wv.OnNavigationFailed = func(u string, err error) { label.Set("Navigation failed: " + u + ": " + err.Error()) }

	w.SetSize(800, 520)
	w.Show()
	wv.Navigate("app://localhost/index.html")
}
