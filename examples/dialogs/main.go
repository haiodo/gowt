// Command dialogs shows a menu bar, a context menu, a tool bar and the modal dialogs.
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Dialogs and menus")
		var out *gowt.TextView
		var goBtn *gowt.ButtonView

		bar := w.MenuBar()
		file := bar.Submenu("&File")
		view := bar.Submenu("&View")

		say := func(s string) { out.SetText(out.Text() + s + "\n") }

		openFile := func() {
			files := w.FileDialog(gowt.FileDialog{
				Title: "Open",
				Multi: true,
				Filters: []gowt.FileFilter{
					{Name: "Text", Pattern: "*.txt;*.md"},
					{Name: "All files", Pattern: "*"},
				},
			})
			if files == nil { // nil means cancelled
				say("open: cancelled")
				return
			}
			say("open: " + strings.Join(files, ", "))
		}
		file.Item("&Open...", openFile, gowt.Shortcut("Cmd+O"))
		file.Item("&Save as...", func() {
			say(fmt.Sprint("save: ", w.FileDialog(gowt.FileDialog{Title: "Save", Save: true, Name: "untitled.txt"})))
		})
		file.Separator()
		file.Item("&Quit", w.Close)

		wrap := view.Item("Word wrap", func() { say("word wrap toggled") }, gowt.Check())
		wrap.SetChecked(true)
		view.OnShow(func() { say("view menu opens, wrap is " + fmt.Sprint(wrap.Checked())) })

		w.Add(
			gowt.ToolBar(gowt.Flat(), gowt.GrowX(), gowt.Init(func(t *gowt.ToolBarView) {
				t.Item("Open", openFile)
				t.Separator()
				t.Item("Folder", func() {
					if dir, ok := w.DirDialog("Pick a folder", ""); ok {
						say("folder: " + dir)
					}
				})
			})),
			gowt.Text(gowt.Multiline(), gowt.ReadOnly(), gowt.Border(),
				gowt.Grow(), gowt.Size(420, 160), gowt.Ref(&out)),
			gowt.Button("Message box", gowt.OnClick(func() {
				a := w.MessageBox(gowt.Message{
					Title:   "Delete",
					Text:    "Delete the file?",
					Icon:    gowt.IconQuestion,
					Buttons: gowt.ButtonsYesNoCancel,
				})
				say(fmt.Sprint("answer: ", a, " yes=", a == gowt.AnswerYes))
			})),
			gowt.Button("Colour", gowt.OnClick(func() {
				if c, ok := w.ColorDialog(gowt.RGB{R: 200, G: 60, B: 60}); ok {
					say(fmt.Sprintf("colour: #%02x%02x%02x", c.R, c.G, c.B))
				}
			})),
			gowt.Button("Go to item...", gowt.Ref(&goBtn), gowt.OnClick(func() {
				gotoItem(app, w, goBtn, say)
			})),
			gowt.Button("Font", gowt.OnClick(func() {
				if f, ok := w.FontDialog(gowt.Font{Name: "Helvetica", Size: 12}); ok {
					say(fmt.Sprintf("font: %s %d bold=%v", f.Name, f.Size, f.Bold))
				}
			})))

		// Right click on the text opens this menu.
		ctx := gowt.PopupMenu(out)
		ctx.Item("Clear", func() { out.SetText("") })

		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

var items = []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India", "Juliet", "Kilo", "Lima"}

// gotoItem opens a search field over a list that narrows as you type. Enter or a double click picks
// the selected item, or the first match; Esc and a click outside close the popover.
func gotoItem(app *gowt.App, w *gowt.Window, anchor gowt.Widget, say func(string)) {
	var field *gowt.TextView
	var list *gowt.ListView
	var matches []string
	p := w.Popover()
	pick := func(i int) {
		if i < 0 || i >= len(matches) {
			return
		}
		say("go to: " + matches[i])
		p.Close()
	}
	filter := func() {
		matches = matches[:0]
		for _, it := range items {
			if strings.Contains(strings.ToLower(it), strings.ToLower(field.Text())) {
				matches = append(matches, it)
			}
		}
		list.SetItems(matches)
		list.Select(0)
	}
	p.Add(
		gowt.Text(gowt.Focused(), gowt.Hint("Go to item"), gowt.GrowX(), gowt.Ref(&field)),
		gowt.List(nil, gowt.Grow(), gowt.Size(240, 160), gowt.Ref(&list),
			gowt.OnActivate(pick)))
	field.OnChange(func(string) { filter() })
	field.OnActivate(func() { pick(max(list.Index(), 0)) })
	p.OnClose(func() { app.Async(p.Dispose) })
	filter()
	p.ShowAt(anchor, gowt.PopoverBelow)
}
