// Command webviewdemo shows a Tree beside a system web view: the pages come from an embed.FS over
// the app:// scheme, a button evaluates JS in the page, and page messages land in a Label.
package main

import (
	"embed"
	"fmt"
	"mime"
	"path"
	"runtime"

	"github.com/haiodo/gowt/swt"
	"github.com/haiodo/gowt/webview"
)

//go:embed assets
var assets embed.FS

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	// Without it macOS lists the process (and WebKit's helpers) as "SWT".
	swt.DisplaySetAppName("WebView Demo")
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("WebView demo")
	shell.SetLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(2, false))

	tree := swt.NewTree(shell, swt.BORDER)
	treeData := swt.NewGridDataStyle(swt.GridDataFILL_VERTICAL)
	treeData.WidthHint = 160
	tree.SetLayoutData(treeData)
	pages := map[*swt.Widget]string{}
	for _, p := range []string{"index.html", "about.html"} {
		it := swt.NewTreeItem(tree, swt.NONE)
		it.SetText(p)
		pages[it.AsWidget()] = p
	}

	right := swt.NewCompositeParentStyle(shell, swt.NONE)
	right.SetLayout(swt.NewGridLayout())
	right.SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_BOTH))
	button := swt.NewButton(right, swt.PUSH)
	button.SetText("Eval in page")
	label := swt.NewLabel(right, swt.WRAP)
	label.SetText("Messages from the page appear here")
	label.SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL))

	wv, err := webview.New(right, webview.Options{Inspectable: true})
	if err != nil {
		fmt.Println(err)
		return
	}
	wv.Control().SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_BOTH))
	wv.HandleScheme("app", func(r webview.Request) *webview.Response {
		name := path.Join("assets", path.Base(r.URL))
		body, err := assets.ReadFile(name)
		if err != nil {
			return nil
		}
		return &webview.Response{MimeType: mime.TypeByExtension(path.Ext(name)), Body: body}
	})
	wv.OnMessage = func(m string) { label.SetText("Message: " + m) }
	wv.OnTitleChanged = func(t string) { shell.SetText("WebView demo - " + t) }
	wv.OnNavigationFailed = func(u string, err error) { label.SetText("Navigation failed: " + u + ": " + err.Error()) }

	button.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		wv.Eval(`document.getElementById("out").textContent = "Eval from Go at " + new Date().toLocaleTimeString(); document.title`,
			func(result string, err error) {
				if err != nil {
					label.SetText("Eval error: " + err.Error())
					return
				}
				label.SetText("Eval result: " + result)
			})
	}))
	tree.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		wv.Navigate("app://localhost/" + pages[e.Item])
	}))

	shell.SetSize(800, 520)
	shell.Open()
	wv.Navigate("app://localhost/index.html")
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
