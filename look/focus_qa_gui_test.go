//go:build gowtqa

package look_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
)

// These tests check which control Show focuses. On macOS an app that is not active has no key window,
// so plain SetFocus is refused; under gowtqa with GOWT_QA_SOCKET Show uses QAFocus, which sets the
// first responder without activating the app.

// Without a list in the content, a macOS window is its own first responder: no control has the focus.
// On Linux, where the QA run keeps Open, SWT keeps its choice.
// A content list and Focused win as in a sidebar window.
func TestInspectorFocus(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	sink := "popup"
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		sink = "nothing" // Windows: the quiet Show does not activate the shell, so nothing takes the focus
	}
	for _, tc := range []struct {
		name    string
		content []gowt.Option
		want    string
	}{
		{"no list", []gowt.Option{gowt.Combo([]string{"Auto"}, gowt.ReadOnly(), name("popup")), gowt.Button("go", name("button"))}, sink},
		{"content table", []gowt.Option{gowt.Button("go"), gowt.Table(name("content table"))}, "content table"},
		{"focused", []gowt.Option{gowt.Combo([]string{"Auto"}, gowt.ReadOnly()), gowt.Text(name("text"), gowt.Focused())}, "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := "not checked"
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					w := app.Window("inspector")
					var content, insp *gowt.Panel
					w.Add(gowt.VBox(append([]gowt.Option{gowt.Ref(&content)}, tc.content...)...),
						gowt.VBox(gowt.Ref(&insp), gowt.Button("inspect")))
					look.SetInspector(w, content, insp, 240)
					t.Setenv("GOWT_QA_SOCKET", "unused") // after Run started, so no QA server listens
					w.Show()
					app.After(200*time.Millisecond, func() { got = focusName(app); w.Close() })
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("focus %s, want %s", got, tc.want)
			}
		})
	}
}

// Without gowt.Focused a sidebar window opens with the focus on the first list of the content, then of
// the sidebar, not on the button before it; Focused still wins.
func TestSidebarFocus(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, tc := range []struct {
		name    string
		content []gowt.Option
		want    string
	}{
		{"sidebar list", []gowt.Option{gowt.Label("content")}, "sidebar list"},
		{"content table", []gowt.Option{gowt.Button("open"), gowt.Table(name("content table"))}, "content table"},
		{"focused", []gowt.Option{gowt.Table(), gowt.Text(name("text"), gowt.Focused())}, "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					w := app.Window("sidebar")
					var sidebar, content *gowt.Panel
					w.Add(
						gowt.VBox(gowt.Ref(&sidebar), gowt.Button("new"), gowt.List([]string{"a"}, name("sidebar list"))),
						gowt.VBox(append([]gowt.Option{gowt.Ref(&content)}, tc.content...)...))
					look.SetSidebar(w, sidebar, content, 200)
					t.Setenv("GOWT_QA_SOCKET", "unused") // after Run started, so no QA server listens
					w.Show()
					// On macOS the first window of the process becomes key only after Show.
					if app.Unwrap().GetActiveShell() == w.Unwrap() {
						got = append(got, focusName(app))
					}
					app.After(200*time.Millisecond, func() { got = append(got, focusName(app)); w.Close() })
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, g := range got {
				if g != tc.want {
					t.Errorf("focus %v, want %s", got, tc.want)
					break
				}
			}
		})
	}
}
