// Command sidebartable shows a List and a row-with-icon list side by side in a look.SetSidebar sidebar.
// A List row cannot have an icon; a Table without columns, header and grid lines does, and looks the same.
package main

import (
	"flag"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

var names = []string{
	"a.cpuprofile", "b.heapsnapshot", "container-1", "container-2",
	"frontend.log", "backend.log", "worker.log", "database.log",
	"network.trace", "startup.trace", "shutdown.trace", "request.trace",
	"memory.heap", "memory.allocs", "cpu.samples", "goroutine.dump",
	"build.log", "test.log", "coverage.out", "benchmark.txt",
	"config.json", "settings.json", "routes.json", "schema.json",
}

func main() {
	fullHeight := flag.Bool("fullheight", false, "use a full-height macOS sidebar scrolling under the toolbar")
	flag.Parse()
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Sidebar table")
		w.SetSize(640, 420)

		var sidebar, content *gowt.Panel
		var list *gowt.ListView
		var table *gowt.TableView
		w.Add(
			gowt.VBox(gowt.Ref(&sidebar), gowt.Padding(0), gowt.Gap(0)),
			gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
		if !*fullHeight {
			sidebar.Add(gowt.List(names, gowt.NoBorder(), gowt.GrowX(), gowt.Size(0, 110), gowt.Ref(&list), gowt.Selected(1),
				gowt.OnSelect(func(i int) { log.Print("list ", i) })))
			gowt.PopupMenu(list).Item("Close", func() { log.Print("close list") })
		}
		sidebar.Add(gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Ref(&table),
			gowt.OnSelect(func(i int) { log.Print("table ", i) }),
			gowt.Init(func(t *gowt.TableView) {
				t.ShowHeader(false)
				t.ShowLines(false)
				for _, n := range names {
					t.Row(n).SetImageAt(0, icons.Get(app, "file"))
				}
				if !*fullHeight {
					t.Select(1) // both selections side by side, for comparing them
				}
			})))
		gowt.PopupMenu(table).Item("Close", func() { log.Print("close table") })
		look.SetSidebar(w, sidebar, content, 240)
		look.SetSidebarFullHeight(w, *fullHeight)
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Show or Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {
				look.SetSidebarVisible(w, !look.SidebarVisible(w))
			}})
		// icons.Get tints for the theme at call time: a switch needs the icon again.
		app.OnThemeChange(func(bool) {
			for i := range names {
				table.RowAt(i).SetImageAt(0, icons.Get(app, "file"))
			}
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
