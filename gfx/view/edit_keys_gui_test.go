package view

import (
	"os"
	"slices"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

func TestEditEnterKeysGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			var v *View
			var ed *Editor
			var commits []string
			w := app.Window("edit Enter keys", New(gfx.NewScene(), func(x *View) { v = x }, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			start := func(multiline bool, text string) {
				ed = v.Edit(EditOptions{Rect: gfx.R(10, 20, 160, 60), Text: text, Multiline: multiline,
					OnCommit: func(s string) { commits = append(commits, s) }})
			}
			whenActive(t, app, w, func() {
				steps := []func(){
					func() {
						start(true, "ab")
						ed.Widget().Unwrap().SetSelection(1)
						postKeyWithMods(app.Unwrap(), swt.CR, swt.SHIFT)
					},
					func() {
						if got := ed.Widget().Text(); got != "a\nb" {
							t.Errorf("Shift+Enter text %q, want %q", got, "a\nb")
						}
						if len(commits) != 0 || len(v.overlays) != 1 {
							t.Errorf("Shift+Enter commits %v overlays %d, want an open edit", commits, len(v.overlays))
						}
						postKey(app.Unwrap(), swt.CR)
					},
					func() {
						if !slices.Equal(commits, []string{"a\nb"}) || len(v.overlays) != 0 {
							t.Errorf("Enter commits %q overlays %d", commits, len(v.overlays))
						}
						start(false, "one")
						postKeyWithMods(app.Unwrap(), swt.CR, swt.SHIFT)
					},
					func() {
						if !slices.Equal(commits, []string{"a\nb", "one"}) || len(v.overlays) != 0 {
							t.Errorf("single-line Shift+Enter commits %q overlays %d", commits, len(v.overlays))
						}
						start(true, "ctrl")
						postKeyWithMods(app.Unwrap(), swt.CR, swt.CTRL)
					},
					func() {
						if !slices.Equal(commits, []string{"a\nb", "one", "ctrl"}) || len(v.overlays) != 0 {
							t.Errorf("Ctrl+Enter commits %q overlays %d", commits, len(v.overlays))
						}
					},
				}
				step := 300 * time.Millisecond
				for i, f := range steps {
					app.After(time.Duration(i+1)*step, func() {
						if !t.Failed() {
							f()
						}
					})
				}
				app.After(time.Duration(len(steps)+1)*step, w.Close)
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
