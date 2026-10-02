// Command minibrowser is a small tabbed browser: a SashForm with the list of open pages on the left
// and one WebView per page, stacked, on the right.
package main

import (
	"cmp"
	"runtime"
	"strings"

	"github.com/haiodo/gowt/swt"
	"github.com/haiodo/gowt/webview"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

const startPage = `<!doctype html><meta charset="utf-8"><title>Start</title>
<body style="font:15px -apple-system,system-ui,sans-serif;margin:24px">
<h2>Mini browser</h2>
<p>Type an address on the left and press Enter or "+" to open it in a new page. Select a page in the list to switch to it; "-" closes it.
Pages keep their state while hidden.</p></body>`

type page struct {
	wv    *webview.WebView
	title string
}

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("Mini browser")
	shell.SetLayout(swt.NewFillLayout())
	sash := swt.NewSashForm(shell, swt.HORIZONTAL)

	left := swt.NewCompositeParentStyle(sash, swt.NONE)
	left.SetLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(3, false))
	addr := swt.NewText(left, swt.SINGLE|swt.BORDER)
	addr.SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL))
	plus := swt.NewButton(left, swt.PUSH)
	plus.SetText("+")
	minus := swt.NewButton(left, swt.PUSH)
	minus.SetText("-")
	list := swt.NewList(left, swt.BORDER|swt.V_SCROLL)
	listData := swt.NewGridDataStyle(swt.GridDataFILL_BOTH)
	listData.HorizontalSpan = 3
	list.SetLayoutData(listData)

	right := swt.NewCompositeParentStyle(sash, swt.NONE)
	right.SetLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(4, false))
	back := swt.NewButton(right, swt.PUSH)
	back.SetText("<")
	forward := swt.NewButton(right, swt.PUSH)
	forward.SetText(">")
	reload := swt.NewButton(right, swt.PUSH)
	reload.SetText("Reload")
	urlLabel := swt.NewLabel(right, swt.NONE)
	urlLabel.SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL))
	stackHost := swt.NewCompositeParentStyle(right, swt.NONE)
	stack := swt.NewStackLayout()
	stackHost.SetLayout(stack)
	hostData := swt.NewGridDataStyle(swt.GridDataFILL_BOTH)
	hostData.HorizontalSpan = 4
	stackHost.SetLayoutData(hostData)
	sash.SetWeights([]int32{10, 90})

	var pages []*page
	selected := func() *page {
		if i := list.GetSelectionIndex(); i >= 0 {
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
		stack.TopControl = p.wv.Control().AsControl()
		stackHost.Layout()
		urlLabel.SetText(p.wv.URL())
		shell.SetText(label(p, "Mini browser"))
	}
	open := func(url string) {
		wv, err := webview.New(stackHost, webview.Options{Inspectable: true})
		if err != nil {
			urlLabel.SetText(err.Error())
			return
		}
		p := &page{wv: wv}
		pages = append(pages, p)
		list.Add(label(p, cmp.Or(url, "Start")))
		list.Select(int32(len(pages) - 1))
		wv.OnTitleChanged = func(t string) {
			p.title = t
			for i, q := range pages {
				if q == p {
					list.SetItem(int32(i), label(p, wv.URL()))
				}
			}
			if selected() == p {
				shell.SetText(label(p, "Mini browser"))
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
		if u := normalize(addr.GetText()); u != "" {
			open(u)
			addr.SetText("")
		}
	}

	plus.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) { add() }))
	addr.AddSelectionListener(swt.SelectionListenerWidgetDefaultSelectedAdapter(func(*swt.SelectionEvent) { add() }))
	list.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) { show() }))
	minus.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
		i := list.GetSelectionIndex()
		if i < 0 {
			return
		}
		pages[i].wv.Dispose()
		pages = append(pages[:i], pages[i+1:]...)
		list.Remove(i)
		if n := int32(len(pages)); n > 0 {
			list.Select(min(i, n-1))
		}
		show()
	}))
	back.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
		if p := selected(); p != nil {
			p.wv.GoBack()
		}
	}))
	forward.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
		if p := selected(); p != nil {
			p.wv.GoForward()
		}
	}))
	reload.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
		if p := selected(); p != nil {
			p.wv.Reload()
		}
	}))

	open("")
	shell.SetSize(1100, 700)
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
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
