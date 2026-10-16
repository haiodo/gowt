// Command finderdemo shows the macOS 26 Finder and System Settings look: a floating Liquid Glass
// sidebar (look.SetSidebar), a toolbar with a back/forward capsule, a button and a search field
// (look.SetToolbar) and grouped sections (gowt.Section). Windows and Linux get a plain sidebar, the
// same toolbar as a bar of flat buttons and a search field, and painted sections.
// GOWT_FINDERDEMO_SECS=n exits after n seconds.
package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
)

var pages = []string{"Appearance", "General", "Desktop & Dock"}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window(pages[0])
		w.SetSize(860, 620)

		var (
			nav   *gowt.ListView
			views *gowt.PagesView
			hist  = []int{0}
			pos   int
		)
		show := func(i int) {
			nav.SetSelection(i)
			views.Show(i)
			w.SetTitle(pages[i])
		}
		open := func(i int) {
			if i == hist[pos] {
				return
			}
			hist = append(hist[:pos+1], i)
			pos++
			show(i)
		}
		step := func(d int) {
			if p := pos + d; p >= 0 && p < len(hist) {
				pos = p
				show(hist[p])
			}
		}
		find := func(text string) {
			for i, p := range pages {
				if text != "" && strings.Contains(strings.ToLower(p), strings.ToLower(text)) {
					open(i)
					return
				}
			}
		}
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Back/Forward", Nav: true, Group: []look.ToolbarItem{
				{Title: "Back", Symbol: "chevron.backward", OnClick: func() { step(-1) }},
				{Title: "Forward", Symbol: "chevron.forward", OnClick: func() { step(1) }},
			}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up", OnClick: func() { log.Print("share") }},
			look.ToolbarItem{Title: "Search", OnSearch: find},
		)

		sidebar := w.Panel()
		sidebar.SetLayout(gowt.Grid{Columns: 1}) // Finder: the list runs edge to edge, its rows inset their selection
		nav = sidebar.List(pages, gowt.NoBorder(), gowt.Grow())
		nav.OnSelect(func(i int) {
			if i >= 0 {
				open(i)
			}
		})

		content := w.Panel()
		content.SetLayout(gowt.Grid{Columns: 1})
		views = content.Pages(gowt.Grow())
		appearance(views.Page())
		general(views.Page())
		dock(views.Page())

		look.SetSidebar(w, sidebar, content, 220)
		show(0)
		if s, _ := strconv.Atoi(os.Getenv("GOWT_FINDERDEMO_SECS")); s > 0 {
			app.After(time.Duration(s)*time.Second, app.Quit)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func page(p *gowt.Panel) *gowt.Panel {
	return p.SetLayout(gowt.Grid{Columns: 1, Margin: 20, Spacing: 8})
}

func heading(p *gowt.Panel, text string) {
	p.Label(text).SetFont(gowt.Font{Weight: gowt.Semibold})
}

// row adds a settings row: the label on the left, the control that add creates on the right.
func row(s *gowt.SectionView, label string, add func(r *gowt.Panel)) {
	r := s.Row().SetLayout(gowt.Grid{Columns: 2, Margin: 6, Spacing: 12})
	r.Label(label, gowt.GrowX(), gowt.Place(gowt.AlignStart, gowt.AlignCenter))
	add(r)
}

func choice(items ...string) func(*gowt.Panel) {
	return func(r *gowt.Panel) { r.Combo(items, gowt.ReadOnly()).Select(0) }
}

func toggle(on bool) func(*gowt.Panel) {
	return func(r *gowt.Panel) { r.Switch("", nil).SetChecked(on) }
}

func appearance(p *gowt.Panel) *gowt.Panel {
	page(p)
	s := p.Section(gowt.GrowX())
	row(s, "Appearance", choice("Auto", "Light", "Dark"))
	row(s, "Liquid Glass", choice("Clear", "Tinted"))
	heading(p, "Theme")
	s = p.Section(gowt.GrowX())
	row(s, "Text highlight color", choice("Accent color", "Orange", "Blue"))
	row(s, "Folder color", choice("Automatic", "Blue"))
	heading(p, "Windows")
	s = p.Section(gowt.GrowX())
	row(s, "Sidebar icon size", choice("Medium", "Small", "Large"))
	row(s, "Tint window background with wallpaper color", toggle(true))
	return p
}

func general(p *gowt.Panel) *gowt.Panel {
	page(p)
	s := p.Section(gowt.GrowX())
	row(s, "Name", func(r *gowt.Panel) { r.Label("MacBook Pro") }) // a Text would keep its white field
	row(s, "Software Update", toggle(true))
	row(s, "Storage", func(r *gowt.Panel) { r.Progress(100).SetValue(62) })
	heading(p, "Sharing")
	s = p.Section(gowt.GrowX())
	row(s, "File Sharing", toggle(false))
	row(s, "Remote Login", toggle(false))
	return p
}

func dock(p *gowt.Panel) *gowt.Panel {
	page(p)
	heading(p, "Dock")
	s := p.Section(gowt.GrowX())
	row(s, "Size", func(r *gowt.Panel) { r.Scale(16, 128, 48) })
	row(s, "Magnification", toggle(false))
	row(s, "Position on screen", choice("Bottom", "Left", "Right"))
	row(s, "Automatically hide and show the Dock", toggle(false))
	return p
}
