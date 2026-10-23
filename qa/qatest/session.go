package qatest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/haiodo/gowt/qa"
)

// session is one running QA build of the app and the connection to it. Calls are serialised: the
// app's UI thread does one thing at a time anyway.
type session struct {
	cmd     *exec.Cmd
	exited  chan struct{}
	dir     string // socket, data directory and app.log live here
	conn    net.Conn
	dec     *json.Decoder
	token   string
	mu      sync.Mutex
	nextID  int64
	cmdLog  []string
	broken  error
	startAt time.Time
}

type response struct {
	ID     int64           `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

const launchTimeout = 30 * time.Second

func launch(binary string, args []string) (*session, error) {
	bin, err := filepath.Abs(binary)
	if err != nil {
		return nil, err
	}
	// A short path: unix socket names are limited to about 100 bytes.
	dir, err := os.MkdirTemp("", "gowtqa")
	if err != nil {
		return nil, err
	}
	sock, data := filepath.Join(dir, "s"), filepath.Join(dir, "data")
	if err := os.Mkdir(data, 0o700); err != nil {
		return nil, err
	}
	logf, err := os.Create(filepath.Join(dir, "app.log"))
	if err != nil {
		return nil, err
	}
	defer logf.Close()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), qa.EnvSocket+"="+sock, qa.EnvData+"="+data)
	cmd.Stdout, cmd.Stderr = logf, logf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	s := &session{cmd: cmd, exited: make(chan struct{}), dir: dir, startAt: time.Now()}
	go func() { cmd.Wait(); close(s.exited) }()
	deadline := time.Now().Add(launchTimeout)
	for {
		select {
		case <-s.exited:
			return nil, fmt.Errorf("%s exited before answering ping (%v)\n%s", bin, cmd.ProcessState, s.logTail())
		default:
		}
		if s.conn == nil {
			if c, token, err := dial(sock); err == nil {
				s.conn, s.token, s.dec = c, token, json.NewDecoder(c)
			}
		}
		if s.conn != nil {
			if _, err := s.call("ping", nil, 5*time.Second); err == nil {
				return s, nil
			}
		}
		if time.Now().After(deadline) {
			s.stop()
			return nil, fmt.Errorf("%s did not answer ping in %v\n%s", bin, launchTimeout, s.logTail())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// dial connects to the unix socket, or to the loopback port a Windows app wrote next to it.
func dial(sock string) (net.Conn, string, error) {
	if b, err := os.ReadFile(sock + ".tcp"); err == nil {
		f := strings.Fields(string(b))
		if len(f) != 2 {
			return nil, "", errors.New("bad " + sock + ".tcp")
		}
		c, err := net.Dial("tcp", f[0])
		return c, f[1], err
	}
	c, err := net.Dial("unix", sock)
	return c, "", err
}

// call sends one command and waits for its answer. A timeout or a broken connection poisons the
// session: the next call would read the late answer to this one.
func (s *session) call(cmd string, args any, timeout time.Duration) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.broken != nil {
		return nil, s.broken
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if args == nil {
		raw = nil
	}
	s.nextID++
	req := map[string]any{"id": s.nextID, "cmd": cmd, "token": s.token}
	if raw != nil {
		req["args"] = json.RawMessage(raw)
	}
	start := time.Now()
	s.conn.SetDeadline(start.Add(timeout))
	var resp response
	err = json.NewEncoder(s.conn).Encode(req)
	if err == nil {
		err = s.dec.Decode(&resp)
	}
	took := time.Since(start).Round(time.Millisecond)
	if err != nil {
		s.broken = fmt.Errorf("%s: connection to the app lost: %w", cmd, err)
		s.cmdLog = append(s.cmdLog, fmt.Sprintf("%s %s %s: %v", start.Format("15:04:05.000"), cmd, raw, err))
		return nil, s.broken
	}
	status := "ok"
	if !resp.OK {
		status = "error: " + resp.Error
	}
	s.cmdLog = append(s.cmdLog, fmt.Sprintf("%s %s %s -> %s (%v)", start.Format("15:04:05.000"), cmd, raw, status, took))
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}

func (s *session) commandLog() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.cmdLog, "\n") + "\n"
}

func (s *session) logTail() string {
	b, _ := os.ReadFile(filepath.Join(s.dir, "app.log"))
	if len(b) > 4096 {
		b = b[len(b)-4096:]
	}
	return string(b)
}

// stop asks the app to quit, kills it after a grace period and removes its directory unless
// GOWT_QA_KEEP is set.
func (s *session) stop() {
	if s.conn != nil && s.broken == nil {
		s.call("quit", nil, 2*time.Second)
	}
	select {
	case <-s.exited:
	case <-time.After(5 * time.Second):
		s.cmd.Process.Kill()
		<-s.exited
	}
	if s.conn != nil {
		s.conn.Close()
	}
	if os.Getenv("GOWT_QA_KEEP") == "" {
		os.RemoveAll(s.dir)
	}
}
