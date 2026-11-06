package swttests

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/haiodo/gowt/internal/junit"
	"github.com/haiodo/gowt/swt"
)

// Hand-written parts of the clipboard tests (manual.txt).

// peerRequest and peerReply mirror the protocol of tests/clipboardpeer.
type peerRequest struct {
	Cmd       string   `json:"cmd"`
	Value     string   `json:"value,omitempty"`
	Files     []string `json:"files,omitempty"`
	Clipboard int32    `json:"clipboard,omitempty"`
}

type peerReply struct {
	OK    bool     `json:"ok"`
	Value string   `json:"value,omitempty"`
	Files []string `json:"files,omitempty"`
	Error string   `json:"error,omitempty"`
}

var (
	peerBinaryOnce sync.Once
	peerBinary     string
	peerBinaryErr  error
)

// clipboardPeerBinary builds the peer process once per run and returns its
// path. GOWT_CLIPBOARD_PEER points at a prebuilt binary and skips the build.
func clipboardPeerBinary() (string, error) {
	peerBinaryOnce.Do(func() {
		if p := os.Getenv("GOWT_CLIPBOARD_PEER"); p != "" {
			if _, err := os.Stat(p); err != nil {
				peerBinaryErr = fmt.Errorf("GOWT_CLIPBOARD_PEER %s: %w", p, err)
				return
			}
			peerBinary = p
			return
		}
		root, err := repoRoot()
		if err != nil {
			peerBinaryErr = err
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		bin := filepath.Join(os.TempDir(), "gowt-clipboardpeer")
		cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./tests/clipboardpeer")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			peerBinaryErr = fmt.Errorf("go build ./tests/clipboardpeer: %v: %s", err, strings.TrimSpace(string(out)))
			return
		}
		peerBinary = bin
	})
	return peerBinary, peerBinaryErr
}

// repoRoot finds the checkout root by walking up from the working directory
// looking for gowt's go.mod (the swttest runner starts in the root).
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil &&
			strings.HasPrefix(string(b), "module github.com/haiodo/gowt\n") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod of github.com/haiodo/gowt not found above the working directory")
		}
		dir = parent
	}
}

// RemoteClipboard is the clipboard peer process of the cross-application
// clipboard tests: a second process (tests/clipboardpeer) with its own Display
// and Clipboard, driven over JSON lines on stdin/stdout. Start skips the
// calling test through a failed assumption when the peer cannot be built or
// launched, so a test that needs another clipboard owner is SKIP and the
// others run against the process's own clipboard.
type RemoteClipboard struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

func NewRemoteClipboard() *RemoteClipboard { return &RemoteClipboard{} }

func (r *RemoteClipboard) Start() {
	bin, err := clipboardPeerBinary()
	junit.AssumeTrue(err == nil, fmt.Sprint("clipboard peer: ", err))
	r.cmd = exec.Command(bin)
	r.stdin, err = r.cmd.StdinPipe()
	junit.AssumeTrue(err == nil, fmt.Sprint("clipboard peer: ", err))
	out, err := r.cmd.StdoutPipe()
	junit.AssumeTrue(err == nil, fmt.Sprint("clipboard peer: ", err))
	r.cmd.Stderr = os.Stderr
	err = r.cmd.Start()
	junit.AssumeTrue(err == nil, fmt.Sprint("clipboard peer: start: ", err))
	r.stdout = bufio.NewReader(out)
	var rep peerReply
	if err := r.call(&peerRequest{Cmd: "ping"}, &rep); err != nil || !rep.OK {
		r.Stop()
		junit.AssumeTrue(false, fmt.Sprintf("clipboard peer: ping: %v %s", err, rep.Error))
	}
}

func (r *RemoteClipboard) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil {
		return
	}
	if r.stdin != nil {
		b, _ := json.Marshal(peerRequest{Cmd: "quit"})
		fmt.Fprintf(r.stdin, "%s\n", b)
		r.stdin.Close()
	}
	done := make(chan error, 1)
	go func() { done <- r.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		r.cmd.Process.Kill()
		<-done
	}
	r.cmd = nil
}

// call sends one request and waits for its reply, at most 30 s.
func (r *RemoteClipboard) call(req *peerRequest, rep *peerReply) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil {
		return errors.New("the peer process is not running")
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.stdin, "%s\n", b); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		line, err := r.stdout.ReadBytes('\n')
		if err != nil {
			done <- err
			return
		}
		done <- json.Unmarshal(line, rep)
	}()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			display := swt.DisplayGetCurrent()
			if display == nil || display.IsDisposed() {
				continue
			}
			for display.ReadAndDispatch() {
			}
		case <-timer.C:
			return errors.New("no reply in 30s")
		}
	}
}

// mustCall runs one command and returns the reply; a peer failure fails the test.
func (r *RemoteClipboard) mustCall(req *peerRequest) peerReply {
	var rep peerReply
	if err := r.call(req, &rep); err != nil {
		panic("clipboard peer " + req.Cmd + ": " + err.Error())
	}
	if !rep.OK {
		panic("clipboard peer " + req.Cmd + ": " + rep.Error)
	}
	return rep
}

func clipboardArg(clipboard []int32) int32 {
	if len(clipboard) > 0 {
		return clipboard[0]
	}
	return swt.DNDCLIPBOARD
}

func (this *RemoteClipboard) SetContents(s string, clipboard ...int32) {
	this.mustCall(&peerRequest{Cmd: "set_text", Value: s, Clipboard: clipboardArg(clipboard)})
}

func (this *RemoteClipboard) GetStringContents(clipboard ...int32) string {
	return this.mustCall(&peerRequest{Cmd: "get_text", Clipboard: clipboardArg(clipboard)}).Value
}

func (this *RemoteClipboard) ClearContents(clipboard ...int32) {
	this.mustCall(&peerRequest{Cmd: "clear", Clipboard: clipboardArg(clipboard)})
}

func (this *RemoteClipboard) SetRtfContents(s string) {
	this.mustCall(&peerRequest{Cmd: "set_rtf", Value: s})
}

func (this *RemoteClipboard) GetRtfContents() string {
	return this.mustCall(&peerRequest{Cmd: "get_rtf"}).Value
}

func (this *RemoteClipboard) SetHtmlContents(s string) {
	this.mustCall(&peerRequest{Cmd: "set_html", Value: s})
}

func (this *RemoteClipboard) GetHtmlContents() string {
	return this.mustCall(&peerRequest{Cmd: "get_html"}).Value
}

func (this *RemoteClipboard) SetUrlContents(b []int8) {
	this.mustCall(&peerRequest{Cmd: "set_url", Value: string(int8bytes(b))})
}

func (this *RemoteClipboard) GetUrlContents() []int8 {
	s := this.mustCall(&peerRequest{Cmd: "get_url"}).Value
	if s == "" {
		return nil
	}
	return fromBytes([]byte(s))
}

func (this *RemoteClipboard) SetImageContents(b []int8) {
	this.mustCall(&peerRequest{Cmd: "set_image", Value: base64.StdEncoding.EncodeToString(int8bytes(b))})
}

func (this *RemoteClipboard) GetImageContents() []int8 {
	v := this.mustCall(&peerRequest{Cmd: "get_image"}).Value
	if v == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		panic("clipboard peer get_image: " + err.Error())
	}
	return fromBytes(b)
}

func (this *RemoteClipboard) SetFileListContents(l []string) {
	this.mustCall(&peerRequest{Cmd: "set_files", Files: l})
}

func (this *RemoteClipboard) GetFileListContents() []string {
	return this.mustCall(&peerRequest{Cmd: "get_files"}).Files
}

func (this *RemoteClipboard) SetMyTypeContents(b []int8) {
	this.mustCall(&peerRequest{Cmd: "set_mine", Value: base64.StdEncoding.EncodeToString(int8bytes(b))})
}

func (this *RemoteClipboard) GetMyTypeContents() []int8 {
	v := this.mustCall(&peerRequest{Cmd: "get_mine"}).Value
	if v == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		panic("clipboard peer get_mine: " + err.Error())
	}
	return fromBytes(b)
}

func int8bytes(s []int8) []byte {
	b := make([]byte, len(s))
	for i, v := range s {
		b[i] = byte(v)
	}
	return b
}

func fromBytes(b []byte) []int8 {
	s := make([]int8, len(b))
	for i, v := range b {
		s[i] = int8(v)
	}
	return s
}

// Test_org_eclipse_swt_dnd_ClipboardPeer is the cross-application clipboard
// slice: every scenario goes through the peer process, never the process's own
// clipboard round trip. Clear and ownership loss have no upstream counterpart
// (the Swing peer's interface lacks them), so they live here.
type Test_org_eclipse_swt_dnd_ClipboardPeer struct {
	ClipboardBase
	textTransfer *swt.TextTransfer
}

func NewTest_org_eclipse_swt_dnd_ClipboardPeer() *Test_org_eclipse_swt_dnd_ClipboardPeer {
	this := &Test_org_eclipse_swt_dnd_ClipboardPeer{}
	this.impl = this
	return this
}

func (this *Test_org_eclipse_swt_dnd_ClipboardPeer) setUp_() {
	this.ClipboardBase.setUp_()
	this.textTransfer = swt.TextTransferGetInstance()
}

func (this *Test_org_eclipse_swt_dnd_ClipboardPeer) Test_PeerSetLocalGet() {
	this.OpenAndFocusRemote()
	hello := ClipboardBaseGetUniqueTestString()
	this.remote.SetContents(hello)
	this.OpenAndFocusShell(false)
	junit.AssertEquals(hello, this.clipboard.GetContents(upcastswtTextTransferToswtTransfer(this.textTransfer)))
}

func (this *Test_org_eclipse_swt_dnd_ClipboardPeer) Test_LocalSetPeerGet() {
	this.OpenAndFocusShell(true)
	hello := ClipboardBaseGetUniqueTestString()
	this.clipboard.SetContents([]any{hello}, []*swt.Transfer{upcastswtTextTransferToswtTransfer(this.textTransfer)})
	this.OpenAndFocusRemote()
	junit.AssertEquals(hello, this.remote.GetStringContents())
}

func (this *Test_org_eclipse_swt_dnd_ClipboardPeer) Test_PeerClear() {
	this.OpenAndFocusRemote()
	hello := ClipboardBaseGetUniqueTestString()
	this.remote.SetContents(hello)
	this.OpenAndFocusShell(false)
	junit.AssertEquals(hello, this.clipboard.GetContents(upcastswtTextTransferToswtTransfer(this.textTransfer)))
	this.remote.ClearContents()
	junit.AssertNull(this.clipboard.GetContents(upcastswtTextTransferToswtTransfer(this.textTransfer)))
}

func (this *Test_org_eclipse_swt_dnd_ClipboardPeer) Test_OwnershipChange() {
	this.OpenAndFocusShell(true)
	first := ClipboardBaseGetUniqueTestString()
	this.clipboard.SetContents([]any{first}, []*swt.Transfer{upcastswtTextTransferToswtTransfer(this.textTransfer)})
	this.OpenAndFocusRemote()
	junit.AssertEquals(first, this.remote.GetStringContents())
	second := ClipboardBaseGetUniqueTestString()
	this.remote.SetContents(second)
	junit.AssertEquals(second, this.clipboard.GetContents(upcastswtTextTransferToswtTransfer(this.textTransfer)))
}

func init() {
	junit.Register(&junit.Class{
		Name: "Test_org_eclipse_swt_dnd_ClipboardPeer",
		New:  func() any { return NewTest_org_eclipse_swt_dnd_ClipboardPeer() },
		BeforeEach: []func(any){
			func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).impl.setUp_() },
		},
		AfterEach: []func(any){
			func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).TearDown() },
		},
		Tests: []junit.Test{
			{Name: "test_PeerSetLocalGet", Tags: []string{"clipboard"}, Run: func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).Test_PeerSetLocalGet() }},
			{Name: "test_LocalSetPeerGet", Tags: []string{"clipboard"}, Run: func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).Test_LocalSetPeerGet() }},
			{Name: "test_PeerClear", Tags: []string{"clipboard"}, Run: func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).Test_PeerClear() }},
			{Name: "test_OwnershipChange", Tags: []string{"clipboard"}, Run: func(t any) { t.(*Test_org_eclipse_swt_dnd_ClipboardPeer).Test_OwnershipChange() }},
		},
	})
}

// openAndFocusShell without the Wayland branch that waits for a button press (a null Boolean in Java):
// no port runs on Wayland.
func (this *ClipboardBase) OpenAndFocusShell(forSetContents bool) {
	junit.AssertNull(this.shell)
	this.shell = swt.NewShellDisplay(this.display)
	SwtTestUtilOpenShell(this.shell)
}
