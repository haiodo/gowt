package swttests

import (
	"os"

	"github.com/haiodo/gowt/internal/junit"
	"github.com/haiodo/gowt/swt"
)

// Hand-written parts of the clipboard tests (manual.txt).

// RemoteClipboard is the Swing peer process of SWT's clipboard tests, driven over RMI. There is no
// such peer here: Start skips the test, so a test that needs another clipboard owner is SKIP and
// the others run against the process's own clipboard.
type RemoteClipboard struct{}

func NewRemoteClipboard() *RemoteClipboard { return &RemoteClipboard{} }

func (this *RemoteClipboard) Start() {
	junit.AssumeTrue(false, "no remote clipboard peer: the Swing/RMI process of the SWT tests is not ported")
}

func (this *RemoteClipboard) Stop()                                    {}
func (this *RemoteClipboard) SetContents(s string, clipboard ...int32) {}
func (this *RemoteClipboard) GetStringContents(clipboard ...int32) string {
	return ""
}
func (this *RemoteClipboard) SetRtfContents(s string)        {}
func (this *RemoteClipboard) GetRtfContents() string         { return "" }
func (this *RemoteClipboard) SetHtmlContents(s string)       {}
func (this *RemoteClipboard) GetHtmlContents() string        { return "" }
func (this *RemoteClipboard) SetUrlContents(b []int8)        {}
func (this *RemoteClipboard) GetUrlContents() []int8         { return nil }
func (this *RemoteClipboard) SetImageContents(b []int8)      {}
func (this *RemoteClipboard) GetImageContents() []int8       { return nil }
func (this *RemoteClipboard) SetFileListContents(l []string) {}
func (this *RemoteClipboard) GetFileListContents() []string  { return nil }
func (this *RemoteClipboard) SetMyTypeContents(b []int8)     {}
func (this *RemoteClipboard) GetMyTypeContents() []int8      { return nil }

// openAndFocusShell without the Wayland branch that waits for a button press (a null Boolean in Java):
// no port runs on Wayland.
func (this *ClipboardBase) OpenAndFocusShell(forSetContents bool) {
	junit.AssertNull(this.shell)
	this.shell = swt.NewShellDisplay(this.display)
	SwtTestUtilOpenShell(this.shell)
}

// CapturedOutput is CapturedOutput.java: what Go code writes to os.Stdout and os.Stderr while it is open
// (temp files, so the content is complete at once). Like the Java class it does not see C-level output.
type CapturedOutput struct {
	origOut, origErr *os.File
	out, err         *os.File
}

func NewCapturedOutput() *CapturedOutput {
	c := &CapturedOutput{origOut: os.Stdout, origErr: os.Stderr}
	var e error
	if c.out, e = os.CreateTemp("", "swt-out"); e != nil {
		panic(e)
	}
	if c.err, e = os.CreateTemp("", "swt-err"); e != nil {
		panic(e)
	}
	os.Stdout, os.Stderr = c.out, c.err
	return c
}

func read(f *os.File) string {
	b, _ := os.ReadFile(f.Name())
	return string(b)
}

func (c *CapturedOutput) GetOutContent() string { return read(c.out) }
func (c *CapturedOutput) GetErrContent() string { return read(c.err) }

func (c *CapturedOutput) AssertNoOutput() {
	junit.AssertEquals("", c.GetOutContent())
	junit.AssertEquals("", c.GetErrContent())
}

func (c *CapturedOutput) Close() {
	os.Stdout, os.Stderr = c.origOut, c.origErr
	for _, f := range []*os.File{c.out, c.err} {
		f.Close()
		os.Remove(f.Name())
	}
}
