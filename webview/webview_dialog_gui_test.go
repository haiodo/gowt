//go:build windows

package webview_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/webview"
)

// The marker prompt of gowt.call is answered by the handler and never reaches the dialog path; the
// page's own dialogs do, and their answers come back to the page. A native dialog would block the
// page and fail the wait.
const dialogHTML = `<title>d</title><script>
var r = gowt.call("init");
var a = alert("A");
var c = confirm("C");
var p = prompt("P", "def");
gowt.postMessage([r, a, c, p].join("|"));
</script>`

func TestWebViewScriptDialogs(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var shown []string
	webview.SetPageDialog(func(kind int32, msg, def string) (bool, string) {
		shown = append(shown, fmt.Sprintf("%d:%s:%s", kind, msg, def))
		return true, "typed"
	})
	var msgs []string
	var err error
	onMain(func() {
		err = g.Run(func(app *g.App) {
			w := app.Window("webview-dialogs")
			var host *g.Panel
			w.Add(g.Fill(g.Grow(), g.Ref(&host)))
			wv, e := webview.NewOn(host, webview.Options{})
			if e != nil {
				t.Error(e)
				app.Quit()
				return
			}
			if !wv.SetCallHandler(func(m string) string { return "re:" + m }) {
				t.Error("engine lacks the call handler")
			}
			wv.HandleScheme("app", func(r webview.Request) *webview.Response {
				return &webview.Response{MimeType: "text/html", Body: []byte(dialogHTML)}
			})
			wv.OnMessage = func(m string) { msgs = append(msgs, m) }
			w.SetSize(400, 300)
			w.Show()
			wv.Navigate("app://host/index.html")
			deadline := time.Now().Add(30 * time.Second)
			var tick func()
			tick = func() {
				if len(msgs) > 0 || time.Now().After(deadline) {
					w.Close()
					return
				}
				app.After(50*time.Millisecond, tick)
			}
			tick()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	// alert gives undefined, which join renders empty; confirm true; prompt the text.
	if len(msgs) != 1 || msgs[0] != "re:init||true|typed" {
		t.Errorf("messages %q", msgs)
	}
	want := []string{"0:A:", "1:C:", "2:P:def"}
	if fmt.Sprint(shown) != fmt.Sprint(want) {
		t.Errorf("page dialogs %q, want %q", shown, want)
	}
}
