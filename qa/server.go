//go:build gowtqa

package qa

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

// Host is what the server needs from the running app; gowt.Run fills it. Package gowt imports qa, so
// qa cannot import gowt.
type Host struct {
	Display    *swt.Display
	Sync       func(f func())
	Flush      func()
	Pending    func() string // what settle waits for: queued updates, jobs, tracks, short timers; "" when nothing
	Jobs       func() []Job
	Quit       func()
	ShotHandle func(*swt.Shell) int64
}

// request and response are one JSON object per line. Token is set only on the TCP fallback.
type request struct {
	ID    int64           `json:"id"`
	Cmd   string          `json:"cmd"`
	Args  json.RawMessage `json:"args,omitempty"`
	Token string          `json:"token,omitempty"`
}

type response struct {
	ID     int64  `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

const (
	cmdTimeout    = 10 * time.Second
	settleTimeout = 5 * time.Second
)

type server struct {
	h     Host
	token string
}

// Start serves in the background on the socket named by GOWT_QA_SOCKET; without it Start does nothing.
func Start(h Host) error {
	path := os.Getenv(EnvSocket)
	if path == "" {
		return nil
	}
	l, token, err := listen(path)
	if err != nil {
		return fmt.Errorf("qa: %w", err)
	}
	activateDialogs()
	s := &server{h: h, token: token}
	go s.accept(l)
	return nil
}

// listen opens a unix socket in a private directory. Windows without AF_UNIX gets 127.0.0.1 and a
// random token instead; path+".tcp" holds "<addr> <token>" for the client, readable by this user only.
func listen(path string) (net.Listener, string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, "", err
	}
	os.Remove(path)
	if runtime.GOOS != "windows" && len(path) >= 104 {
		return nil, "", fmt.Errorf("the socket path %q is %d bytes; unix sockets take under 104 (macOS) or 108 (Linux): set %s to a shorter path", path, len(path), EnvSocket)
	}
	l, err := net.Listen("unix", path)
	if err == nil {
		os.Chmod(path, 0o600)
		return l, "", nil
	}
	if runtime.GOOS != "windows" {
		return nil, "", err
	}
	l, terr := net.Listen("tcp", "127.0.0.1:0")
	if terr != nil {
		return nil, "", errors.Join(err, terr)
	}
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	if err := os.WriteFile(path+".tcp", []byte(l.Addr().String()+" "+token+"\n"), 0o600); err != nil {
		l.Close()
		return nil, "", err
	}
	return l, token, nil
}

func (s *server) accept(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go s.serve(c)
	}
}

func (s *server) serve(c net.Conn) {
	defer c.Close()
	dec, enc := json.NewDecoder(c), json.NewEncoder(c)
	for {
		var req request
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := response{ID: req.ID}
		if s.token != "" && req.Token != s.token {
			resp.Error = "bad token"
		} else {
			v, err := s.run(req)
			resp.OK, resp.Result = err == nil, v
			if err != nil {
				resp.Error = err.Error()
			}
		}
		if err := enc.Encode(resp); err != nil {
			return
		}
		if req.Cmd == "quit" && resp.OK {
			s.h.Quit()
			return
		}
	}
}

func (s *server) run(req request) (v any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	switch req.Cmd {
	case "ping":
		return s.ui(cmdTimeout, func() (any, error) {
			return map[string]any{"pid": os.Getpid(), "goos": runtime.GOOS}, nil
		})
	case "windows":
		return s.ui(cmdTimeout, func() (any, error) { return s.windows(), nil })
	case "dump":
		return s.ui(cmdTimeout, func() (any, error) { return qatree.Dump(s.h.Display), nil })
	case "find":
		var sel qatree.Selector
		if err := json.Unmarshal(req.Args, &sel); err != nil {
			return nil, fmt.Errorf("find: %w", err)
		}
		return s.ui(cmdTimeout, func() (any, error) { return qatree.Find(s.h.Display, sel) })
	case "act":
		return s.act(req.Args)
	case "settle":
		var a struct {
			TimeoutMS int `json:"timeout_ms"`
		}
		if len(req.Args) > 0 {
			if err := json.Unmarshal(req.Args, &a); err != nil {
				return nil, fmt.Errorf("settle: %w", err)
			}
		}
		return nil, errors.Join(s.settle(durationMS(a.TimeoutMS, settleTimeout)), takeDialogFailures())
	case "answer":
		return nil, answerCmd(req.Args)
	case "dialogs":
		return dialogsCmd(req.Args)
	case "shot":
		return s.shot(req.Args)
	case "jobs":
		return s.ui(cmdTimeout, func() (any, error) { return s.h.Jobs(), nil })
	case "state":
		return s.state(req.Args)
	case "quit":
		return nil, nil
	}
	return nil, fmt.Errorf("unknown command %q", req.Cmd)
}

func durationMS(ms int, def time.Duration) time.Duration {
	if ms <= 0 {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}

// ui runs f on the UI thread through Sync, recovering its panic, and gives up waiting after timeout.
// The goroutine stays behind when the UI thread is stuck, a modal dialog for one.
func (s *server) ui(timeout time.Duration, f func() (any, error)) (any, error) {
	type result struct {
		v   any
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var r result
		ran := false
		defer func() {
			if p := recover(); p != nil {
				r.err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
			}
			ch <- r
		}()
		s.h.Sync(func() {
			ran = true
			defer func() {
				if p := recover(); p != nil {
					r.err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
				}
			}()
			r.v, r.err = f()
		})
		if !ran {
			r.err = errors.New("the app is not running")
		}
	}()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("the UI thread did not answer in %v (a modal dialog?)", timeout)
	}
}

// shells are the windows a test can address: open ones, not the popovers, which belong to their window.
func (s *server) shells() []*swt.Shell {
	var out []*swt.Shell
	for _, sh := range s.h.Display.GetShells() {
		if _, pop := qaview.PopoverOf(sh); !pop && !sh.IsDisposed() && sh.GetVisible() {
			out = append(out, sh)
		}
	}
	return out
}

func (s *server) windows() []map[string]any {
	var out []map[string]any
	for i, sh := range s.shells() {
		b := sh.GetBounds()
		out = append(out, map[string]any{
			"index": i, "title": sh.GetText(), "visible": sh.GetVisible(),
			"bounds": [4]int{int(b.X), int(b.Y), int(b.Width), int(b.Height)},
		})
	}
	return out
}

// act runs one semantic action on the single control the selector picks, then settles unless the
// request says "settle": false.
func (s *server) act(raw json.RawMessage) (any, error) {
	var a struct {
		Selector  qatree.Selector `json:"selector"`
		Action    string          `json:"action"`
		Args      json.RawMessage `json:"args"`
		Settle    *bool           `json:"settle"`
		TimeoutMS int             `json:"timeout_ms"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("act: %w", err)
	}
	_, err := s.ui(cmdTimeout, func() (any, error) {
		return nil, qatree.Act(s.h.Display, a.Selector, a.Action, a.Args)
	})
	if err == nil && (a.Settle == nil || *a.Settle) {
		err = s.settle(durationMS(a.TimeoutMS, settleTimeout))
	}
	// A dialog without a scripted answer was cancelled; that is the cause of whatever else went wrong.
	return nil, errors.Join(takeDialogFailures(), err)
}

// settle flushes until nothing is pending and two dumps in a row are equal. Pending includes the
// running Jobs, Tracks and short timers; background jobs and long timers are not waited for.
func (s *server) settle(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var prev []byte
	var pending string
	changing := false
	for {
		var cur []byte
		left := time.Until(deadline)
		if left <= 0 {
			return fmt.Errorf("settle: not quiet after %v: waiting: %s", timeout, describe(pending, changing))
		}
		_, err := s.ui(cmdTimeout, func() (any, error) {
			s.h.Flush()
			pending = s.h.Pending()
			cur, _ = json.Marshal(qatree.Dump(s.h.Display))
			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("settle: %w", err)
		}
		if pending == "" && bytes.Equal(cur, prev) {
			return nil
		}
		changing = prev != nil && !bytes.Equal(cur, prev)
		prev = cur
		time.Sleep(10 * time.Millisecond)
	}
}

func describe(pending string, changing bool) string {
	if pending == "" {
		pending = "nothing queued"
	}
	if changing {
		pending += "; the dump kept changing"
	}
	return pending
}

// shot writes the PNG of one window to args.path, or returns it base64 encoded in "png".
func (s *server) shot(raw json.RawMessage) (any, error) {
	var a shotArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, fmt.Errorf("shot: %w", err)
		}
	}
	path := a.Path
	if path == "" {
		f, err := os.CreateTemp("", "gowtqa-*.png")
		if err != nil {
			return nil, err
		}
		f.Close()
		path = f.Name()
		defer os.Remove(path)
	}
	var done func() bool
	if _, err := s.ui(cmdTimeout, func() (any, error) {
		sh, err := s.shellOf(a)
		if err == nil {
			done = qaview.Request(sh)
		}
		return nil, err
	}); err != nil {
		return nil, err
	}
	s.waitFrames(done)
	_, err := s.ui(cmdTimeout, func() (any, error) {
		sh, err := s.shellOf(a)
		if err != nil {
			return nil, err
		}
		shot.Passive = len(qaview.Popovers(s.h.Display, sh)) > 0
		defer func() { shot.Passive = false }()
		frame, err := shot.Capture(s.h.ShotHandle(sh), path)
		if err != nil {
			return nil, err
		}
		if err := qaview.OverlayPNG(sh, path); err != nil {
			return nil, err
		}
		return nil, s.addPopovers(sh, path, frame)
	})
	if err != nil {
		return nil, err
	}
	if a.Path != "" {
		return map[string]string{"path": a.Path}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]string{"png": base64.StdEncoding.EncodeToString(b)}, nil
}

// addPopovers draws the popovers of sh that are on screen into the capture at path.
func (s *server) addPopovers(sh *swt.Shell, path string, frame shot.Frame) error {
	for _, ps := range qaview.Popovers(s.h.Display, sh) {
		tmp := path + ".popover.png"
		pf, err := shot.CaptureView(s.h.ShotHandle(ps), tmp)
		if err == nil && pf.W == 0 {
			p, _ := qaview.PopoverOf(ps)
			x, y, w, h := p.Bounds()
			pf = shot.Frame{X: float64(x), Y: float64(y), W: float64(w), H: float64(h)}
		}
		if err == nil {
			frame, err = shot.Compose(path, frame, tmp, pf)
		}
		os.Remove(tmp)
		if err != nil {
			return fmt.Errorf("shot: popover: %w", err)
		}
	}
	return nil
}

// shotArgs say which window a shot takes: the one that holds the Selector's control, else the first
// whose title is Title, else the one at index Window (0 is the first).
type shotArgs struct {
	Window   int              `json:"window"`
	Title    string           `json:"title"`
	Selector *qatree.Selector `json:"selector"`
	Path     string           `json:"path"`
}

func (s *server) shellOf(a shotArgs) (*swt.Shell, error) {
	switch {
	case a.Selector != nil:
		return qatree.ShellOf(s.h.Display, *a.Selector)
	case a.Title != "":
		for _, sh := range s.shells() {
			if sh.GetText() == a.Title {
				return sh, nil
			}
		}
		return nil, fmt.Errorf("shot: no window titled %q", a.Title)
	}
	return s.shell(a.Window)
}

func (s *server) shell(window int) (*swt.Shell, error) {
	shells := s.shells()
	if window < 0 || window >= len(shells) {
		return nil, fmt.Errorf("shot: window %d, the app has %d", window, len(shells))
	}
	return shells[window], nil
}

// frameWait bounds the wait of a shot for the gfx views; a view that animates all the time never
// gets quiet.
const frameWait = 3 * time.Second

// waitFrames returns when done reports that the views presented a frame after the request, or after
// frameWait; the capture then goes ahead with what is on screen.
func (s *server) waitFrames(done func() bool) {
	deadline := time.Now().Add(frameWait)
	for time.Now().Before(deadline) {
		ok := false
		if _, err := s.ui(cmdTimeout, func() (any, error) { ok = done(); return nil, nil }); err != nil || ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// state calls the handler the app registered with Handle under args.name.
func (s *server) state(raw json.RawMessage) (any, error) {
	var a struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("state: %w", err)
	}
	fn := handler(a.Name)
	if fn == nil {
		return nil, fmt.Errorf("state: no handler %q", a.Name)
	}
	args, err := NewArgs(a.Args)
	if err != nil {
		return nil, fmt.Errorf("state %s: %w", a.Name, err)
	}
	v, err := s.ui(cmdTimeout, func() (any, error) { return fn(args) })
	err = errors.Join(takeDialogFailures(), err)
	if err != nil {
		return nil, fmt.Errorf("state %s: %w", a.Name, err)
	}
	return v, nil
}
