package swttests

import (
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
