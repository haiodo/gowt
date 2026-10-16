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
		var tools *gowt.ToolBarView
		var out *gowt.TextView

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
			gowt.ToolBar(gowt.Flat(), gowt.GrowX(), gowt.Ref(&tools)),
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
			gowt.Button("Font", gowt.OnClick(func() {
				if f, ok := w.FontDialog(gowt.Font{Name: "Helvetica", Size: 12}); ok {
					say(fmt.Sprintf("font: %s %d bold=%v", f.Name, f.Size, f.Bold))
				}
			})),
		)

		tools.Item("Open", openFile)
		tools.Separator()
		tools.Item("Folder", func() {
			if dir, ok := w.DirDialog("Pick a folder", ""); ok {
				say("folder: " + dir)
			}
		})

		// Right click on the text opens this menu.
		ctx := gowt.PopupMenu(out)
		ctx.Item("Clear", func() { out.SetText("") })

		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
