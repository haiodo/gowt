package qatest_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os/exec"
	"runtime"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

type clipRead struct {
	Text, HTML, RTF string
	Files           []string
	Types           []string
	Image           []int
}

func readClip(t *testing.T, app *qatest.Client) clipRead {
	t.Helper()
	var r clipRead
	if err := json.Unmarshal(app.State("clip-read"), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func pngBytes(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 200, 255
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

// The app's Clipboard is the virtual one: formats set by the test are read by the app, formats the app
// sets are read by the test, and the system clipboard keeps what the user copied.
func TestVirtualClipboard(t *testing.T) {
	guiOnly(t)
	if runtime.GOOS == "darwin" {
		if err := exec.Command("sh", "-c", "printf gowt-user-clip | pbcopy").Run(); err != nil {
			t.Skip("no pbcopy:", err)
		}
		defer func() {
			out, _ := exec.Command("pbpaste").Output()
			if string(out) != "gowt-user-clip" {
				t.Errorf("the system clipboard now holds %q", out)
			}
		}()
	}
	app := qatest.App(t)
	if c := app.Clipboard(); c.Text != "" || c.HTML != "" || c.Files != nil {
		t.Fatalf("the clipboard is not empty at the start: %+v", c)
	}

	app.State("clip-html", "html", "<b>bold</b>", "plain", "bold")
	if c := app.Clipboard(); c.HTML != "<b>bold</b>" || c.Text != "bold" {
		t.Fatalf("app set html: %+v", c)
	}

	app.SetClipboard(qa.Clipboard{Text: "t", RTF: `{\rtf1 r}`, HTML: "<i>h</i>", Files: []string{"/a/b", "/c"},
		Image: pngBytes(7, 5), Custom: map[string][]byte{"x-gowt-test": {1, 2, 3}}})
	r := readClip(t, app)
	if r.Text != "t" || r.RTF != `{\rtf1 r}` || r.HTML != "<i>h</i>" || len(r.Files) != 2 || r.Files[1] != "/c" {
		t.Errorf("app read %+v", r)
	}
	if len(r.Image) != 2 || r.Image[0] != 7 || r.Image[1] != 5 {
		t.Errorf("image size %v, want [7 5]", r.Image)
	}
	found := false
	for _, n := range r.Types {
		found = found || n == "x-gowt-test"
	}
	if !found {
		t.Errorf("the custom type is not in %q", r.Types)
	}

	// Setting one format clears the others, as on a system clipboard.
	app.State("clip-files", "paths", []string{"/only"})
	if c := app.Clipboard(); len(c.Files) != 1 || c.Text != "" || c.HTML != "" || c.Image != nil {
		t.Errorf("after SetFiles: %+v", c)
	}
	app.SetClipboard(qa.Clipboard{})
	if r := readClip(t, app); r.Text != "" || len(r.Files) != 0 {
		t.Errorf("after the empty set: %+v", r)
	}
}

// Tab sends Traverse before the key: a listener that takes Tab keeps the focus where it is.
func TestKeyTraverse(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	tree := app.Find(qa.ID("lazy"))
	tree.Focus()
	tree.Key("Tab")
	tree.Key("Shift+Tab")
	tree.Key("Esc")
	tree.Key("Enter")
	var got []string
	for _, e := range events(t, app) {
		if len(e) > 8 && e[:8] == "traverse" {
			got = append(got, e)
		}
	}
	want := []string{"traverse lazy detail=16", "traverse lazy detail=8", "traverse lazy detail=2", "traverse lazy detail=4"}
	if len(got) != len(want) {
		t.Fatalf("traverse events %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d is %q, want %q", i, got[i], want[i])
		}
	}
	if !tree.Node().Focused {
		t.Error("Tab, vetoed by the listener, moved the focus")
	}
	app.Find(qa.ID("name")).Focus()
	app.Key("Tab")
	if app.Find(qa.ID("name")).Node().Focused {
		t.Error("Tab in a text field did not traverse")
	}
}
