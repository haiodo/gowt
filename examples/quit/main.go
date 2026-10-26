// Command quit shows a document window that asks "Save changes?" on every way of quitting: its
// close button, Cmd+Q, the Dock, logout and shutdown.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		dirty := false
		var w *gowt.Window
		var status *gowt.LabelView
		mark := func(d bool) {
			dirty = d
			if d {
				w.SetTitle("Untitled *")
				status.SetText("Unsaved changes")
			} else {
				w.SetTitle("Untitled")
				status.SetText("Saved")
			}
		}
		w = app.Window("Untitled")
		w.Add(
			gowt.Text(gowt.GrowX(), gowt.Hint("Type to make the document dirty"),
				gowt.OnChange(func(string) { mark(true) })),
			gowt.Label("Saved", gowt.Ref(&status)),
			gowt.Button("Save", gowt.OnClick(func() { mark(false) })))

		// Every quit path asks the windows' OnClose first, in turn: Cmd+Q, Dock Quit and logout on
		// macOS, logout and shutdown on Windows, the close of the last window everywhere.
		w.OnClose(func() bool {
			if !dirty {
				return true
			}
			switch w.MessageBox(gowt.Message{Title: "Save changes?", Text: "The document has unsaved changes.",
				Icon: gowt.IconQuestion, Buttons: gowt.ButtonsYesNoCancel}) {
			case gowt.AnswerYes:
				mark(false) // save here
				return true
			case gowt.AnswerNo:
				return true
			}
			return false // Cancel, or the box closed: keep the window and cancel the quit
		})
		// OnQuit is asked after every window agreed; use it for what belongs to the app, not a window.
		app.OnQuit(func() bool {
			log.Println("quitting")
			return true
		})
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
