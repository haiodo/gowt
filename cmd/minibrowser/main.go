// Command minibrowser is a small tabbed browser: a SashForm with the list of open pages on the left
// and one WebView per page, stacked, on the right.
package main

import (
	"cmp"
	"log"
	"strings"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
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
	var (
		list  *g.ListView
		stack *g.PagesView
		pages []*page
	)
	addr, url := rx.New(""), rx.New("")
	w := app.Window("Mini browser", g.Padding(0))
	selected := func() *page {
		if i := list.Index(); i >= 0 {
			return pages[i]
		}
		return nil
	}
	show := func() {
		p := selected()
		if p == nil {
			url.Set("")
			return
		}
		stack.Show(list.Index())
		url.Set(p.wv.URL())
		w.SetTitle(label(p, "Mini browser"))
	}
	open := func(u string) {
		// Each page gets its own host: the stack shows hosts, and webview takes a plain composite.
		var host *g.Panel
		stack.Add(g.Fill(g.Ref(&host)))
		wv, err := webview.New(host, webview.Options{Inspectable: true})
		if err != nil {
			host.Dispose()
			url.Set(err.Error())
			return
		}
		p := &page{wv: wv, host: host}
		pages = append(pages, p)
		list.Add(label(p, cmp.Or(u, "Start")))
		list.Select(len(pages) - 1)
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
				url.Set(u)
			}
		}
		if u == "" {
			wv.SetHTML(startPage, "")
		} else {
			wv.Navigate(u)
		}
		show()
	}
	add := func() {
		if u := normalize(addr.Peek()); u != "" {
			open(u)
			addr.Set("")
		}
	}
	remove := func() {
		i := list.Index()
		if i < 0 {
			return
		}
		pages[i].wv.Dispose()
		pages[i].host.Dispose()
		pages = append(pages[:i], pages[i+1:]...)
		list.Remove(i)
		if n := len(pages); n > 0 {
			list.Select(min(i, n-1))
		}
		show()
	}
	// act runs f on the selected page, if any.
	act := func(f func(*webview.WebView)) func() {
		return func() {
			if p := selected(); p != nil {
				f(p.wv)
			}
		}
	}

	w.Add(g.Split(g.Grow(), g.Init(func(s *g.SplitView) { s.SetWeights(10, 90) }),
		g.Grid(3, g.Padding(5),
			g.Text(g.Border(), g.GrowX(), g.Bind(addr), g.OnActivate(add)),
			g.Button("+", g.OnClick(add)),
			g.Button("-", g.OnClick(remove)),
			g.List(nil, g.Grow(), g.Span(3), g.Ref(&list), g.OnSelect(func(int) { show() }))),
		g.Grid(4, g.Padding(5),
			g.Button("<", g.OnClick(act((*webview.WebView).GoBack))),
			g.Button(">", g.OnClick(act((*webview.WebView).GoForward))),
			g.Button("Reload", g.OnClick(act((*webview.WebView).Reload))),
			g.Label("", g.GrowX(), g.BindText(url)),
			g.Pages(g.Grow(), g.Span(4), g.Ref(&stack)))))

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
