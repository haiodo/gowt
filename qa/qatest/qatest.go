// Package qatest drives a QA build of a gowt app (go build -tags gowtqa) from go test. It imports
// no toolkit package, so the test binary does not link AppKit, Win32 or GTK.
//
//	func TestMain(m *testing.M) { qatest.Main(m, "./bin/myapp") }
//
//	func TestSend(t *testing.T) {
//		app := qatest.App(t)
//		app.Find(qa.ID("composer")).Type("hi")
//		app.Find(qa.ID("send")).Click()
//	}
//
// One app process serves a whole test package, so the tests must not run in parallel. Every method
// fails the test with t.Fatalf; there are no error returns.
package qatest

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/qa"
)

// RestartPerTest starts a fresh app for every App(t) instead of calling the app's "reset" handler.
// Set it in TestMain before Main.
var RestartPerTest bool

var (
	binary string
	args   []string
	cur    *session
)

// Main runs the tests of the package with the app at binary (extra args are passed to it) started
// on first use with its own socket and data directory, and stops the app at the end. It exits.
func Main(m *testing.M, bin string, appArgs ...string) {
	binary, args = bin, appArgs
	code := m.Run()
	if cur != nil {
		cur.stop()
	}
	os.Exit(code)
}

// Client is the app as one test sees it.
type Client struct {
	t testing.TB
	s *session
}

// App returns the app of the package, started if this is the first call, after the app's "reset"
// handler (qa.Handle("reset", ...)) has run. Without such a handler the state carries over.
func App(t testing.TB) *Client {
	t.Helper()
	if binary == "" {
		t.Fatalf("qatest: App called without Main in TestMain")
	}
	if cur != nil && (RestartPerTest || cur.broken != nil) {
		cur.stop()
		cur = nil
	}
	if cur == nil {
		s, err := launch(binary, args)
		if err != nil {
			t.Fatalf("qatest: %v", err)
		}
		cur = s
	}
	c := &Client{t: t, s: cur}
	if _, err := c.s.call("dialogs", map[string]bool{"clear": true}, 15*time.Second); err != nil {
		t.Fatalf("qatest: clear dialogs: %v", err)
	}
	if _, err := c.s.call("state", map[string]any{"name": "reset"}, 15*time.Second); err != nil && !strings.Contains(err.Error(), `no handler "reset"`) {
		t.Fatalf("qatest: reset: %v", err)
	}
	t.Cleanup(c.saveOnFailure)
	return c
}

// Call sends a raw protocol command and returns its result, for what the fluent API does not cover.
func (c *Client) Call(cmd string, arg any) (json.RawMessage, error) {
	return c.s.call(cmd, arg, 15*time.Second)
}

func (c *Client) must(cmd string, arg any) json.RawMessage {
	c.t.Helper()
	r, err := c.Call(cmd, arg)
	if err != nil {
		c.t.Fatalf("qatest: %s: %v", cmd, err)
	}
	return r
}

// Settle waits until the app has nothing queued for the UI thread, no running Job, Track or short
// timer, and two dumps in a row are equal. Every action settles before it returns. On a timeout the
// error names what was still pending.
func (c *Client) Settle() {
	c.t.Helper()
	c.must("settle", nil)
}

// Eventually polls cond every 50 ms until it reports ok, for what Settle cannot see: a server
// push, a file written by another process. cond returns the value it looked at; the failure
// names the last one.
func (c *Client) Eventually(timeout time.Duration, cond func() (observed any, ok bool)) {
	c.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		v, ok := cond()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			c.t.Fatalf("qatest: condition not met in %v, last observed: %v", timeout, v)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Key sends a key or combination to the control that has the focus, or to the active window.
// Window.OnKey shortcuts and menu accelerators run; see Element.Key for the names.
func (c *Client) Key(keys string) {
	c.t.Helper()
	if _, err := c.Call("act", map[string]any{"selector": qa.Selector{}, "action": "key", "args": map[string]string{"keys": keys}}); err != nil {
		c.t.Fatalf("qatest: key %s: %v", keys, err)
	}
}

// Answer queues the reply to a dialog the app will open. The dialog does not appear: the app's
// call returns the reply. Answers of one kind are used in the order queued. A dialog with no
// answer fails the action that opened it and gets Cancel. Unused answers are dropped by the
// next App(t).
func (c *Client) Answer(d qa.Dialog) {
	c.t.Helper()
	c.must("answer", d)
}

// Dialogs returns the dialogs the app opened since App(t), with the reply each got.
func (c *Client) Dialogs() []qa.Dialog {
	c.t.Helper()
	var ds []qa.Dialog
	if err := json.Unmarshal(c.must("dialogs", nil), &ds); err != nil {
		c.t.Fatalf("qatest: dialogs: %v", err)
	}
	return ds
}

// Jobs returns the Jobs and Tracks the app is running now, background ones included.
func (c *Client) Jobs() []qa.Job {
	c.t.Helper()
	var jobs []qa.Job
	if err := json.Unmarshal(c.must("jobs", nil), &jobs); err != nil {
		c.t.Fatalf("qatest: jobs: %v", err)
	}
	return jobs
}

// Dump returns the control tree of every window.
func (c *Client) Dump() []qa.Node {
	c.t.Helper()
	var nodes []qa.Node
	if err := json.Unmarshal(c.must("dump", nil), &nodes); err != nil {
		c.t.Fatalf("qatest: dump: %v", err)
	}
	return nodes
}

// State calls the app's handler name with key-value pairs: app.State("chat", "room", "general").
// It returns what the handler returned, JSON encoded.
func (c *Client) State(name string, kv ...any) json.RawMessage {
	c.t.Helper()
	if len(kv)%2 != 0 {
		c.t.Fatalf("qatest: State(%q): odd number of key-value arguments", name)
	}
	m := map[string]any{}
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			c.t.Fatalf("qatest: State(%q): key %d is %T, not a string", name, i/2, kv[i])
		}
		m[k] = kv[i+1]
	}
	return c.must("state", map[string]any{"name": name, "args": m})
}

// Shot saves a PNG of the first window as <name>.png in the artifact directory and returns its path.
func (c *Client) Shot(name string) string {
	c.t.Helper()
	path := filepath.Join(c.artifactDir(), name+".png")
	if err := c.shotTo(path); err != nil {
		c.t.Fatalf("qatest: shot: %v", err)
	}
	c.t.Logf("qatest: shot %s", path)
	return path
}

func (c *Client) shotTo(path string) error {
	r, err := c.Call("shot", nil)
	if err != nil {
		return err
	}
	var res struct{ PNG string }
	if err := json.Unmarshal(r, &res); err != nil {
		return err
	}
	b, err := base64.StdEncoding.DecodeString(res.PNG)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// artifactDir is $GOWT_QA_ARTIFACTS/<test> or a temp directory (t.TempDir would vanish with the test).
func (c *Client) artifactDir() string {
	base := os.Getenv(qa.EnvArtifacts)
	dir := ""
	if base != "" {
		dir = filepath.Join(base, unsafeName.ReplaceAllString(c.t.Name(), "_"))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			c.t.Fatalf("qatest: %v", err)
		}
	} else {
		d, err := os.MkdirTemp("", "gowtqa-artifacts-")
		if err != nil {
			c.t.Fatalf("qatest: %v", err)
		}
		dir = d
	}
	return dir
}

// saveOnFailure leaves a shot, the dump and the command log of a failed test, and the app's log tail.
func (c *Client) saveOnFailure() {
	if !c.t.Failed() {
		return
	}
	dir := c.artifactDir()
	var notes []string
	if c.s.broken == nil {
		if err := c.shotTo(filepath.Join(dir, "failure.png")); err != nil {
			notes = append(notes, "shot: "+err.Error())
		}
		if r, err := c.Call("dump", nil); err != nil {
			notes = append(notes, "dump: "+err.Error())
		} else {
			os.WriteFile(filepath.Join(dir, "dump.json"), r, 0o644)
		}
	}
	os.WriteFile(filepath.Join(dir, "commands.log"), []byte(c.s.commandLog()), 0o644)
	os.WriteFile(filepath.Join(dir, "app.log.tail"), []byte(c.s.logTail()), 0o644)
	c.t.Logf("qatest: artifacts of the failed test in %s %s", dir, strings.Join(notes, "; "))
}

// Element is the control a selector picks, looked up again on every call.
type Element struct {
	c   *Client
	sel qa.Selector
}

// Find picks the control for sel. Nothing is looked up until the element is used.
func (c *Client) Find(sel qa.Selector) *Element { return &Element{c, sel} }

func (e *Element) act(action string, args any) {
	e.c.t.Helper()
	body := map[string]any{"selector": e.sel, "action": action}
	if args != nil {
		body["args"] = args
	}
	if _, err := e.c.Call("act", body); err != nil {
		e.c.t.Fatalf("qatest: %s %s: %v", action, e.describe(), err)
	}
}

// Click presses a button, toggles a check or radio, follows a link: the control's own selection event.
func (e *Element) Click() { e.c.t.Helper(); e.act("click", nil) }

// Type replaces the text of an input control.
func (e *Element) Type(text string) { e.c.t.Helper(); e.act("type", map[string]string{"text": text}) }

// Check sets a check box, radio or toggle.
func (e *Element) Check(on bool) { e.c.t.Helper(); e.act("check", map[string]bool{"on": on}) }

// Select selects the item at index in a list, combo, table or tab folder.
func (e *Element) Select(index int) { e.c.t.Helper(); e.act("select", map[string]int{"index": index}) }

// Activate does what a double click or Enter does on a row of a list, table or tree (the selected
// row of the control itself), in a single-line text field, on a push button, or on a window (its
// default button). A row is selected first.
func (e *Element) Activate() { e.c.t.Helper(); e.act("activate", nil) }

// Expand opens a tree row: the Expand event runs first, so the app can build the children.
func (e *Element) Expand() { e.c.t.Helper(); e.act("expand", nil) }

// Collapse closes a tree row.
func (e *Element) Collapse() { e.c.t.Helper(); e.act("collapse", nil) }

// Focus gives the control the keyboard focus. Its window must be the active one.
func (e *Element) Focus() { e.c.t.Helper(); e.act("focus", nil) }

// Close closes a window as its close button does: OnClose handlers can keep it open.
func (e *Element) Close() { e.c.t.Helper(); e.act("close", nil) }

// Key sends a key or combination to the control: "Enter", "Esc", "Shift+Tab", "Mod+K". Mod is Cmd
// on macOS and Ctrl elsewhere; Cmd works on macOS only.
func (e *Element) Key(keys string) {
	e.c.t.Helper()
	e.act("key", map[string]string{"keys": keys})
}

// ScrollTo scrolls a ScrollView so that the control target picks is shown.
func (e *Element) ScrollTo(target qa.Selector) {
	e.c.t.Helper()
	e.act("scroll", map[string]any{"to": target})
}

// ScrollBy moves the content of a ScrollView by dx, dy pixels; the origin stops at the content's edge.
func (e *Element) ScrollBy(dx, dy int) {
	e.c.t.Helper()
	e.act("scroll", map[string]int{"dx": dx, "dy": dy})
}

// ScrollIntoView scrolls the ScrollView that holds the control until the control is shown.
func (e *Element) ScrollIntoView() { e.c.t.Helper(); e.act("scroll", nil) }

// SelectText selects the item with exactly this text in a list, combo, table, tree or tab folder.
func (e *Element) SelectText(text string) {
	e.c.t.Helper()
	e.act("select", map[string]string{"text": text})
}

// Nodes returns every control the selector matches now.
func (e *Element) Nodes() []qa.Node {
	e.c.t.Helper()
	var nodes []qa.Node
	if err := json.Unmarshal(e.c.must("find", e.sel), &nodes); err != nil {
		e.c.t.Fatalf("qatest: find %s: %v", e.describe(), err)
	}
	return nodes
}

// Exists reports whether the selector matches anything now.
func (e *Element) Exists() bool { e.c.t.Helper(); return len(e.Nodes()) > 0 }

// Node returns the one control the selector matches; none or several fail the test.
func (e *Element) Node() qa.Node {
	e.c.t.Helper()
	nodes := e.Nodes()
	if len(nodes) != 1 {
		e.c.t.Fatalf("qatest: %s matches %d controls, want 1", e.describe(), len(nodes))
	}
	return nodes[0]
}

func (e *Element) describe() string {
	b, _ := json.Marshal(e.sel)
	return string(b)
}
