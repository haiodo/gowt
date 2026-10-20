// Command sidebartable shows a List and a row-with-icon list side by side in a look.SetSidebar sidebar.
// A List row cannot have an icon; a Table without columns, header and grid lines does, and looks the same.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
)

var names = []string{"a.cpuprofile", "b.heapsnapshot", "container-1"}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Sidebar table")
		w.SetSize(640, 420)

		var sidebar, content *gowt.Panel
		var list *gowt.ListView
		var table *gowt.TableView
		w.Add(
			gowt.VBox(gowt.Ref(&sidebar), gowt.Padding(0), gowt.Gap(0),
				gowt.List(names, gowt.NoBorder(), gowt.GrowX(), gowt.Size(0, 110), gowt.Ref(&list),
					gowt.OnSelect(func(i int) { log.Print("list ", i) })),
				gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Ref(&table),
					gowt.OnSelect(func(i int) { log.Print("table ", i) }),
					gowt.Init(func(t *gowt.TableView) {
						t.ShowHeader(false)
						t.ShowLines(false)
						for _, n := range names {
							t.Row(n).SetImageAt(0, icons.Get(app, "file"))
						}
					}))),
			gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
		gowt.PopupMenu(list).Item("Close", func() { log.Print("close list") })
		gowt.PopupMenu(table).Item("Close", func() { log.Print("close table") })
		look.SetSidebar(w, sidebar, content, 240)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
