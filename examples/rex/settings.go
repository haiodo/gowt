package main

import (
	"strings"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/rx"
)

type category struct {
	name, status string
	icon         *gowt.Image
	page         gowt.Option
}

type preferences struct {
	app        *gowt.App
	categories []category
	title      *rx.State[string]
	pages      *gowt.PagesView
	list       *drawnList
	search     *gowt.TextView

	filtered []int
	current  int
	suppress bool

	history    []int
	historyPos int
}

func settingsWindow(app *gowt.App) (*gowt.Window, *drawnList) {
	p := newPreferences(app)
	w := app.Window("Preferences")
	w.SetSize(1180, 760)
	w.SetMinSize(860, 560)
	w.SetTitle("")
	w.SetBackground(bgSidebar)
	look.SetFullSizeContent(w, true)
	look.SetSidebarFullHeight(w, true)

	var sidebar, content *gowt.Panel
	w.Add(p.sidebarPane(&sidebar), p.contentPane(&content))
	look.SetSidebar(w, sidebar, content, gowt.Scaled(300))
	p.show(1, true)
	w.OnClose(func() bool {
		if quitting {
			return true
		}
		w.SetVisible(false)
		return false
	})
	return w, p.list
}

func newPreferences(app *gowt.App) *preferences {
	cats := categories(app)
	p := &preferences{
		app:        app,
		categories: cats,
		title:      rx.New("General"),
		current:    1,
		historyPos: -1,
		filtered:   make([]int, 0, len(cats)),
	}
	p.list = newDrawnList(app, bgSidebar, 30, p.listItems(""), p.current, func(v int) { p.show(v, true) })
	p.list.id = "settings-list"
	p.list.focus = true // the arrows walk the categories as soon as the window opens
	return p
}

func (p *preferences) sidebarPane(ref **gowt.Panel) gowt.Option {
	return gowt.VBox(gowt.Ref(ref), gowt.Padding(14), gowt.Gap(gowt.Spacing.SM),
		gowt.Background(bgSidebar), gowt.Foreground(fg),
		gowt.Label("", gowt.Size(0, 32)),
		gowt.Text(gowt.Search(), gowt.Hint("Search Settings"), gowt.GrowX(), gowt.Small(),
			gowt.ID("settings-search"),
			gowt.Ref(&p.search), gowt.OnChange(func(q string) { p.filter(q) })),
		p.list.Option())
}

func (p *preferences) contentPane(ref **gowt.Panel) gowt.Option {
	pages := make([]gowt.Option, 0, len(p.categories)+3)
	pages = append(pages, gowt.Grow(), gowt.Ref(&p.pages), gowt.Background(bgApp))
	for _, c := range p.categories {
		pages = append(pages, c.page)
	}
	return gowt.VBox(gowt.Ref(ref), gowt.Gap(0), gowt.Padding(0),
		gowt.Background(bgApp), gowt.Foreground(fg),
		p.header(),
		gowt.Pages(pages...))
}

func (p *preferences) header() gowt.Option {
	return gowt.HBox(gowt.GrowX(), gowt.Padding(20), gowt.Gap(gowt.Spacing.MD),
		gowt.Background(bgApp), gowt.Foreground(fg),
		gowt.HBox(gowt.Place(gowt.AlignStart, gowt.AlignCenter),
			gowt.Padding(gowt.Scaled(2)), gowt.Gap(0),
			gowt.Init(func(cap *gowt.Panel) {
				look.SetGlass(cap, true)
				look.SetGlassRadius(cap, look.Capsule)
			}),
			gowt.Button("", gowt.Borderless(), gowt.Size(gowt.Scaled(28), gowt.Scaled(28)),
				gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Tooltip("Back"),
				gowt.Init(func(b *gowt.ButtonView) { themedIcon(p.app, "back", 16, b.SetImage) }),
				gowt.OnClick(p.back)),
			gowt.Button("", gowt.Borderless(), gowt.Size(gowt.Scaled(28), gowt.Scaled(28)),
				gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Tooltip("Forward"),
				gowt.Init(func(b *gowt.ButtonView) { themedIcon(p.app, "forward", 16, b.SetImage) }),
				gowt.OnClick(p.forward))),
		label("", gowt.BindText(p.title), gowt.Typeface(gowt.Type.Title),
			gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
		gowt.Spacer())
}

func (p *preferences) filter(query string) {
	if p.suppress || p.list == nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(query))
	p.list.SetItems(p.listItems(q))
	if len(p.filtered) == 0 {
		return
	}
	if p.rowOf(p.current) < 0 {
		p.show(p.filtered[0], true)
		return
	}
	p.list.Select(p.current)
}

func (p *preferences) listItems(query string) []listItem {
	p.filtered = p.filtered[:0]
	items := make([]listItem, 0, len(p.categories))
	for i, c := range p.categories {
		if query != "" && !strings.Contains(strings.ToLower(c.name), query) {
			continue
		}
		it := listItem{title: c.name, icon: c.icon, value: i}
		if c.status != "" {
			it.detail = c.status
		}
		items = append(items, it)
		p.filtered = append(p.filtered, i)
	}
	return items
}

func (p *preferences) resetFilter() {
	if p.search == nil || p.search.Text() == "" {
		return
	}
	p.suppress = true
	p.search.SetText("")
	p.suppress = false
	p.list.SetItems(p.listItems(""))
}

func (p *preferences) show(cat int, record bool) {
	if cat < 0 || cat >= len(p.categories) {
		return
	}
	p.current = cat
	if p.pages != nil {
		p.pages.Show(cat)
	}
	if p.title != nil {
		p.title.Set(p.categories[cat].name)
	}
	if p.rowOf(cat) < 0 && p.search != nil && p.search.Text() != "" {
		p.resetFilter()
	}
	if record {
		p.remember(cat)
	}
	if p.list != nil {
		p.list.Select(cat)
	}
}

func (p *preferences) back() {
	if p.historyPos <= 0 {
		return
	}
	p.historyPos--
	p.show(p.history[p.historyPos], false)
}

func (p *preferences) forward() {
	if p.historyPos+1 >= len(p.history) {
		return
	}
	p.historyPos++
	p.show(p.history[p.historyPos], false)
}

func (p *preferences) remember(cat int) {
	if p.historyPos >= 0 && p.history[p.historyPos] == cat {
		return
	}
	p.history = append(p.history[:p.historyPos+1], cat)
	p.historyPos = len(p.history) - 1
}

func (p *preferences) rowOf(cat int) int {
	for row, i := range p.filtered {
		if i == cat {
			return row
		}
	}
	return -1
}
