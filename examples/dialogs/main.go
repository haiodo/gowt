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
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
		fillX := gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true})

		bar := w.MenuBar()
		file := bar.Submenu("&File")
		view := bar.Submenu("&View")

		tools := w.ToolBar(gowt.Flat(), fillX)
		out := w.Text(gowt.Multiline(), gowt.ReadOnly(), gowt.Border(),
			gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true, Width: 420, Height: 160}))
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
		file.Item("&Open...\tCmd+O", openFile)
		file.Item("&Save as...", func() {
			say(fmt.Sprint("save: ", w.FileDialog(gowt.FileDialog{Title: "Save", Save: true, Name: "untitled.txt"})))
		})
		file.Separator()
		file.Item("&Quit", w.Close)

		wrap := view.Item("Word wrap", func() { say("word wrap toggled") }, gowt.Check())
		wrap.SetChecked(true)
		view.OnShow(func() { say("view menu opens, wrap is " + fmt.Sprint(wrap.Checked())) })

		tools.Item("Open", openFile)
		tools.Separator()
		tools.Item("Folder", func() {
			if dir, ok := w.DirDialog("Pick a folder", ""); ok {
				say("folder: " + dir)
			}
		})

		w.Button("Message box", func() {
			a := w.MessageBox(gowt.Message{
				Title:   "Delete",
				Text:    "Delete the file?",
				Icon:    gowt.IconQuestion,
				Buttons: gowt.ButtonsYesNoCancel,
			})
			say(fmt.Sprint("answer: ", a, " yes=", a == gowt.AnswerYes))
		})
		w.Button("Colour", func() {
			if c, ok := w.ColorDialog(gowt.RGB{R: 200, G: 60, B: 60}); ok {
				say(fmt.Sprintf("colour: #%02x%02x%02x", c.R, c.G, c.B))
			}
		})
		w.Button("Font", func() {
			if f, ok := w.FontDialog(gowt.Font{Name: "Helvetica", Size: 12}); ok {
				say(fmt.Sprintf("font: %s %d bold=%v", f.Name, f.Size, f.Bold))
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
