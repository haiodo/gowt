package gowt

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// openBox holds the OnOpenFiles handlers and the paths that came before the first one.
type openBox struct {
	handlers []func([]string)
	queue    []string
}

func (b *openBox) deliver(paths []string) {
	if len(paths) == 0 {
		return
	}
	if len(b.handlers) == 0 {
		b.queue = dedupe(append(b.queue, paths...))
		return
	}
	for _, h := range b.handlers {
		h(append([]string(nil), paths...))
	}
}

// openListener adapts swt.OpenDocument, whose Text is one path per event.
type openListener func(raw ...string)

func (l openListener) HandleEvent(e *swt.Event) { l(e.Text) }

// openIn takes raw paths or file URLs on the UI thread; the ones that arrive in the same turn of
// the event loop are delivered in one call.
func (a *App) openIn(raw ...string) {
	if len(a.openPend) == 0 {
		a.display.AsyncExec(jrt.NewRunnable(a.live(a.openFlush)))
	}
	a.openPend = append(a.openPend, raw...)
}

func (a *App) openFlush() {
	raw := a.openPend
	a.openPend = nil
	a.open.deliver(openPaths(raw))
}

var urlScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]+:`)

// openPath turns a path or a file: URL into an absolute path; other URL schemes are not files.
func openPath(s string) (string, bool) {
	if strings.HasPrefix(s, "file:") {
		u, err := url.Parse(s)
		if err != nil || (u.Host != "" && u.Host != "localhost") {
			return "", false
		}
		s = u.Path
		if runtime.GOOS == "windows" && len(s) > 2 && s[0] == '/' && s[2] == ':' {
			s = s[1:]
		}
	} else if urlScheme.MatchString(s) && !strings.HasPrefix(s[strings.IndexByte(s, ':'):], ":\\") {
		return "", false
	}
	if s == "" {
		return "", false
	}
	p, err := filepath.Abs(filepath.FromSlash(s))
	return p, err == nil
}

func openPaths(raw []string) []string {
	var out []string
	for _, s := range raw {
		if p, ok := openPath(s); ok {
			out = append(out, p)
		}
	}
	return dedupe(out)
}

func dedupe(paths []string) []string {
	seen := map[string]bool{}
	out := paths[:0:0]
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// argFiles returns the arguments that name an existing file or directory, as absolute paths.
func argFiles(args []string) []string {
	var out []string
	for _, s := range openPaths(args) {
		if _, err := os.Stat(s); err == nil {
			out = append(out, s)
		}
	}
	return out
}

// inbox takes the files that other processes send, from any goroutine; before Run they wait here.
var inbox struct {
	sync.Mutex
	app     *App
	backlog []string
	pending bool // a request came before Run and wants its window raised
}

func inboxPush(raw []string) {
	inbox.Lock()
	defer inbox.Unlock()
	if inbox.app == nil {
		inbox.backlog = append(inbox.backlog, raw...)
		inbox.pending = true
		return
	}
	inbox.app.takeForwarded(raw)
}

// takeForwarded delivers files that another process sent and raises a window; any goroutine.
func (a *App) takeForwarded(raw []string) {
	a.Async(func() {
		a.openIn(raw...)
		for _, w := range a.windows {
			if !w.shell.IsDisposed() {
				w.shell.SetMinimized(false)
				w.shell.ForceActive()
				return
			}
		}
	})
}

// attachInbox sends the backlog and later pushes to a; the returned func detaches.
func (a *App) attachInbox() func() {
	inbox.Lock()
	defer inbox.Unlock()
	inbox.app = a
	if inbox.pending {
		a.takeForwarded(inbox.backlog)
		inbox.backlog, inbox.pending = nil, false
	}
	return func() {
		inbox.Lock()
		inbox.app = nil
		inbox.Unlock()
	}
}

var singleCalled atomic.Bool

var appIDPart = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// SingleInstance makes the program a single instance per user under id, a reverse-DNS name such as
// "org.example.MyApp" (at least two dot-separated parts of letters, digits, "_" and "-"). Call it
// once, before Run. It returns true when this process is the running instance, and false when
// another one already holds id: this process then sent it the file arguments (see OnOpenFiles) or,
// with none, asked it to raise its window, and should exit without calling Run.
//
// If the check cannot be done, it returns true and the error: a malformed id, or on Linux no
// session bus. The program then runs as an ordinary extra instance, getting command-line files only.
//
// Linux: the id is the D-Bus name of the freedesktop Application interface; with
// DBusActivatable=true in the .desktop file the desktop calls Open on it. Windows: a Unix socket
// named after id in the user's temp folder; if two first launches start in the same instant both
// may win, and a crashed instance leaves a file that the next launch replaces. macOS: Launch
// Services already runs one instance, and the call does nothing. The instance holds id until the
// process ends.
func SingleInstance(id string) (bool, error) {
	parts := strings.Split(id, ".")
	ok := len(parts) >= 2 && len(id) <= 255
	for _, p := range parts {
		ok = ok && appIDPart.MatchString(p)
	}
	if !ok {
		return true, errors.New("gowt: SingleInstance: id must be a reverse-DNS name like org.example.App, got " + id)
	}
	if singleCalled.Swap(true) {
		return true, errors.New("gowt: SingleInstance is already called")
	}
	return singleInstance(id, argFiles(os.Args[1:]))
}
