// Command finderdemo shows the macOS 26 Finder and System Settings look: a floating Liquid Glass
// sidebar (look.SetSidebar), a toolbar with an item over the sidebar, a back/forward capsule, a
// button and a search field (look.SetToolbar) and grouped sections (gowt.Section). Windows and Linux
// get a plain sidebar with the sidebar item in its first row, the rest of the toolbar as a bar of flat
// buttons and a search field, and painted sections.
// GOWT_FINDERDEMO_SECS=n exits after n seconds; GOWT_FINDERDEMO_HIDE=1 starts with the sidebar hidden,
// GOWT_FINDERDEMO_NARROW=1 at the narrowest window.
package main

import (
	"log"
	"os"
	"runtime"
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
			nav.Select(i)
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
		toggle := func() { look.SetSidebarVisible(w, !look.SidebarVisible(w)) }
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: toggle},
			look.ToolbarItem{Title: "New", Symbol: "plus", Sidebar: true, OnClick: func() { log.Print("new") }},
			look.ToolbarItem{Title: "Back/Forward", Nav: true, Group: []look.ToolbarItem{
				{Title: "Back", Symbol: "chevron.backward", OnClick: func() { step(-1) }},
				{Title: "Forward", Symbol: "chevron.forward", OnClick: func() { step(1) }},
			}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up", OnClick: func() { log.Print("share") }},
			look.ToolbarItem{Title: "Search", OnSearch: find})

		var sidebar, content *gowt.Panel
		// Finder: the list runs edge to edge, its rows inset their selection
		w.Add(
			gowt.VBox(gowt.Ref(&sidebar), gowt.List(pages, gowt.NoBorder(), gowt.Grow(), gowt.Ref(&nav), gowt.OnSelect(func(i int) {
				if i >= 0 {
					open(i)
				}
			}))),
			gowt.VBox(gowt.Ref(&content),
				gowt.Pages(gowt.Grow(), gowt.Ref(&views), appearance(), general(), dock())))

		look.SetSidebar(w, sidebar, content, 220)
		// The sections need more than the 200 points of content SetSidebar keeps; with a wider sidebar
		// the content gets narrower again, since the minimum is the whole window's.
		w.SetMinSize(640, 400)
		var keys []gowt.Option
		if runtime.GOOS == "darwin" {
			keys = append(keys, gowt.Shortcut("Ctrl+Cmd+S")) // elsewhere Cmd is Ctrl, and Ctrl+S is Save
		}
		hide := w.MenuBar().Submenu("&View").Item("Hide Sidebar", toggle, keys...)
		look.OnSidebarChange(w, func(visible bool, width int) {
			if visible {
				hide.SetText("Hide Sidebar")
			} else {
				hide.SetText("Show Sidebar")
			}
			log.Printf("sidebar visible %v, width %d", visible, width)
		})
		if os.Getenv("GOWT_FINDERDEMO_HIDE") == "1" {
			look.SetSidebarVisible(w, false)
		}
		if os.Getenv("GOWT_FINDERDEMO_NARROW") == "1" {
			w.SetSize(100, 620) // the sidebar's minimum window size widens it
		}
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

func page(children ...gowt.Option) gowt.Option {
	return gowt.VBox(append([]gowt.Option{gowt.Padding(20)}, children...)...)
}

func heading(text string) gowt.Option {
	return gowt.Label(text, gowt.Init(func(l *gowt.LabelView) { l.SetFont(gowt.Font{Weight: gowt.Semibold}) }))
}

// row is a settings row: the label on the left, the control on the right.
func row(label string, control gowt.Option) gowt.Option {
	return gowt.Grid(2, gowt.Padding(6), gowt.Gap(12),
		gowt.Label(label, gowt.GrowX(), gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
		control)
}

func choice(items ...string) gowt.Option {
	return gowt.Combo(items, gowt.ReadOnly(), gowt.Selected(0))
}

func toggle(on bool) gowt.Option {
	return gowt.Switch("", gowt.Checked(on))
}

func storage(value int) gowt.Option {
	return gowt.Progress(100, gowt.Value(value))
}

func appearance() gowt.Option {
	return page(
		gowt.Section(gowt.GrowX(),
			row("Appearance", choice("Auto", "Light", "Dark")),
			row("Liquid Glass", choice("Clear", "Tinted"))),
		heading("Theme"),
		gowt.Section(gowt.GrowX(),
			row("Text highlight color", choice("Accent color", "Orange", "Blue")),
			row("Folder color", choice("Automatic", "Blue"))),
		heading("Windows"),
		gowt.Section(gowt.GrowX(),
			row("Sidebar icon size", choice("Medium", "Small", "Large")),
			row("Tint window background with wallpaper color", toggle(true))))
}

func general() gowt.Option {
	return page(
		gowt.Section(gowt.GrowX(),
			row("Name", gowt.Label("MacBook Pro")), // a Text would keep its white field
			row("Software Update", toggle(true)),
			row("Storage", storage(62))),
		heading("Sharing"),
		gowt.Section(gowt.GrowX(),
			row("File Sharing", toggle(false)),
			row("Remote Login", toggle(false))))
}

func dock() gowt.Option {
	return page(
		heading("Dock"),
		gowt.Section(gowt.GrowX(),
			row("Size", gowt.Scale(16, 128, gowt.Value(48))),
			row("Magnification", toggle(false)),
			row("Position on screen", choice("Bottom", "Left", "Right")),
			row("Automatically hide and show the Dock", toggle(false))))
}
