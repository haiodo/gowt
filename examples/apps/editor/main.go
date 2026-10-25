// Command editor is a code editor: a file tree, closable tabs, a StyledText with highlighted code in the
// monospace font, line numbers and a status bar. Static data, fixed 1280x800 window
// (docs/look-check.md).
package main

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/internal/kit"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

const source = `package queue

import (
	"errors"
	"sync"
)

// ErrEmpty is returned by Pop when the queue has no items.
var ErrEmpty = errors.New("queue is empty")

// Queue is a first-in first-out list safe for concurrent use.
type Queue[T any] struct {
	mu    sync.Mutex
	items []T
}

// Push adds v to the end of the queue.
func (q *Queue[T]) Push(v T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, v)
}

// Pop removes and returns the first item.
func (q *Queue[T]) Pop() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.items) == 0 {
		return zero, ErrEmpty
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v, nil
}
`

var (
	keywords = []string{"package", "import", "var", "type", "struct", "func", "defer", "return", "if", "any"}
	kwColor  = gowt.RGB{R: 170, G: 50, B: 150}
	strColor = gowt.RGB{R: 200, G: 60, B: 40}
	cmtColor = gowt.RGB{R: 100, G: 140, B: 100}
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.SetDensity(gowt.DensityNative)
		kit.Appearance(app)
		w := app.Window("queue.go - gowt-sample")
		w.SetSize(kit.Width, kit.Height)
		w.SetMinSize(800, 500)

		var sidebar, content *gowt.Panel
		w.Add(sidebarPane(app, &sidebar), contentPane(&content))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Run", Symbol: "play.fill", OnClick: func() {}},
			look.ToolbarItem{Title: "Search", OnSearch: func(string) {}})
		look.SetSidebar(w, sidebar, content, gowt.Scaled(240))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func sidebarPane(app *gowt.App, ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.Tree(gowt.NoBorder(), gowt.Grow(), gowt.Init(func(t *gowt.TreeView) {
			t.Column("", gowt.Scaled(220))
			t.ShowHeader(false)
			folder := icons.Get(app, "folder", icons.Size(18))
			file := icons.Get(app, "file", icons.Size(18))
			root := t.Node("gowt-sample")
			root.SetImage(folder)
			cmd := root.Node("cmd")
			cmd.SetImage(folder)
			cmd.Node("main.go").SetImage(file)
			pkg := root.Node("queue")
			pkg.SetImage(folder)
			pkg.Node("queue.go").SetImage(file)
			pkg.Node("queue_test.go").SetImage(file)
			root.Node("go.mod").SetImage(file)
			root.Node("README.md").SetImage(file)
			root.SetExpanded(true)
			cmd.SetExpanded(true)
			pkg.SetExpanded(true)
			t.Select(pkg.Children()[0])
		})))
}

func contentPane(ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(0), gowt.Gap(0),
		gowt.CTabs(gowt.Grow(), gowt.Selected(1),
			gowt.Tab("main.go", gowt.Closable()),
			gowt.Tab("queue.go", gowt.Closable(), gowt.Padding(0), gowt.Gap(0),
				gowt.StyledText(gowt.Scrollbars(), gowt.Borderless(), gowt.Grow(), gowt.LineNumbers(), gowt.Init(editor)))),
		gowt.Separator(gowt.GrowX()),
		gowt.HBox(gowt.Gap(gowt.Spacing.LG), gowt.GrowX(), gowt.Padding(gowt.Spacing.SM),
			gowt.Label("Ln 27, Col 14", gowt.Typeface(gowt.Type.Caption)), gowt.Label("UTF-8", gowt.Typeface(gowt.Type.Caption)), gowt.Label("Go", gowt.Typeface(gowt.Type.Caption)),
			gowt.Spacer(), gowt.Label("0 problems", kit.Faint(), gowt.Typeface(gowt.Type.Caption))))
}

func editor(ed *gowt.StyledTextView) {
	ed.SetFont(gowt.Type.Mono)
	ed.SetTabs(4)
	ed.SetText(source)
	ed.OnStyle(func(line string) []gowt.StyleSpan {
		var spans []gowt.StyleSpan
		add := func(byteStart, byteLen int, c gowt.RGB) {
			spans = append(spans, gowt.StyleSpan{Start: utf8.RuneCountInString(line[:byteStart]), Length: utf8.RuneCountInString(line[byteStart : byteStart+byteLen]), Style: gowt.TextStyle{Foreground: &c}})
		}
		if i := strings.Index(line, "//"); i >= 0 {
			add(i, len(line)-i, cmtColor)
			return spans
		}
		for _, kw := range keywords {
			for from := 0; ; {
				i := strings.Index(line[from:], kw)
				if i < 0 {
					break
				}
				i += from
				from = i + len(kw)
				if (i == 0 || !isWord(line[i-1])) && (from >= len(line) || !isWord(line[from])) {
					add(i, len(kw), kwColor)
				}
			}
		}
		if i := strings.Index(line, `"`); i >= 0 {
			if j := strings.Index(line[i+1:], `"`); j >= 0 {
				add(i, j+2, strColor)
			}
		}
		return spans
	})
	ed.SetCaret(0)
}

func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
