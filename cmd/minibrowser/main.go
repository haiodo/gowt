// Command minibrowser is a small tabbed browser: a SashForm with the list of open pages on the left
// and one WebView per page, stacked, on the right.
package main

import (
	"cmp"
	"log"
	"strings"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/webview"
)

const startPage = `<!doctype html><meta charset="utf-8"><title>Start</title>
<body style="font:15px -apple-system,system-ui,sans-serif;margin:24px">
<h2>Mini browser</h2>
<p>Type an address on the left and press Enter or "+" to open it in a new page. Select a page in the list to switch to it; "-" closes it.
Pages keep their state while hidden.</p></body>`

type page struct {
	wv    *webview.WebView
	host  *g.Panel
	title string
}

func main() {
	// Without it macOS lists the process (and WebKit's helpers) as "SWT".
	g.SetAppName("Mini Browser")
	if err := g.Run(build); err != nil {
		log.Fatal(err)
	}
}

func build(app *g.App) {
	fillX := g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true})
	w := app.Window("Mini browser")
	w.SetLayout(g.Fill{})
	split := w.Split()

	left := split.Panel()
	left.SetLayout(g.Grid{Columns: 3, Margin: 5, Spacing: 5})
	addr := left.Text(g.Border(), fillX)
	plus := left.Button("+", nil)
	minus := left.Button("-", nil)
	list := left.List(nil, g.Cell(g.GridCell{Align: g.AlignFill, VAlign: g.AlignFill, GrowX: true, GrowY: true, SpanX: 3}))

	right := split.Panel()
	right.SetLayout(g.Grid{Columns: 4, Margin: 5, Spacing: 5})
	back := right.Button("<", nil)
	forward := right.Button(">", nil)
	reload := right.Button("Reload", nil)
	urlLabel := right.Label("", fillX)
	stackHost := right.Panel(g.Cell(g.GridCell{Align: g.AlignFill, VAlign: g.AlignFill, GrowX: true, GrowY: true, SpanX: 4}))
	stackHost.SetLayout(g.Stack{})
	split.SetWeights(10, 90)

	var pages []*page
	selected := func() *page {
		if i := list.Index(); i >= 0 {
			return pages[i]
		}
		return nil
	}
	show := func() {
		p := selected()
		if p == nil {
			urlLabel.SetText("")
			return
		}
		stackHost.ShowTop(p.host)
		urlLabel.SetText(p.wv.URL())
		w.SetTitle(label(p, "Mini browser"))
	}
	open := func(url string) {
		// Each page gets its own host: the stack shows hosts, and webview takes a plain composite.
		host := stackHost.Panel()
		host.SetLayout(g.Fill{})
		wv, err := webview.New(host.Unwrap(), webview.Options{Inspectable: true})
		if err != nil {
			host.Unwrap().Dispose()
			urlLabel.SetText(err.Error())
			return
		}
		p := &page{wv: wv, host: host}
		pages = append(pages, p)
		list.Add(label(p, cmp.Or(url, "Start")))
		list.SetSelection(len(pages) - 1)
		wv.OnTitleChanged = func(t string) {
			p.title = t
			for i, q := range pages {
				if q == p {
					list.SetItem(i, label(p, wv.URL()))
				}
			}
			if selected() == p {
				w.SetTitle(label(p, "Mini browser"))
			}
		}
		wv.OnNavigationFinished = func(u string) {
			if selected() == p {
				urlLabel.SetText(u)
			}
		}
		if url == "" {
			wv.SetHTML(startPage, "")
		} else {
			wv.Navigate(url)
		}
		show()
	}
	add := func() {
		if u := normalize(addr.Text()); u != "" {
			open(u)
			addr.SetText("")
		}
	}

	plus.OnClick(add)
	addr.OnActivate(add)
	list.OnSelect(func(int) { show() })
	minus.OnClick(func() {
		i := list.Index()
		if i < 0 {
			return
		}
		pages[i].wv.Dispose()
		pages[i].host.Unwrap().Dispose()
		pages = append(pages[:i], pages[i+1:]...)
		list.Remove(i)
		if n := len(pages); n > 0 {
			list.SetSelection(min(i, n-1))
		}
		show()
	})
	back.OnClick(func() {
		if p := selected(); p != nil {
			p.wv.GoBack()
		}
	})
	forward.OnClick(func() {
		if p := selected(); p != nil {
			p.wv.GoForward()
		}
	})
	reload.OnClick(func() {
		if p := selected(); p != nil {
			p.wv.Reload()
		}
	})

	open("")
	w.SetSize(1100, 700)
	w.Show()
}

// label is the page's title, else its URL, else fallback.
func label(p *page, fallback string) string {
	if p.title != "" {
		return p.title
	}
	if u := p.wv.URL(); u != "" {
		return u
	}
	return fallback
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	if s != "" && !strings.Contains(s, "://") {
		s = "https://" + s
	}
	return s
}
