package gowt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// TestQuitExitsCleanly wants exit 0 and a disposed display after a quit; a child process, since a
// crash kills it. "menu" is Cmd+Q, "terminate" the Dock and logout.
func TestQuitExitsCleanly(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, path := range []string{"menu", "menu-async", "terminate"} {
		t.Run(path, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run", "^TestQuitExitChild$")
			mark := filepath.Join(t.TempDir(), "mark")
			cmd.Env = append(os.Environ(), "GOWT_QUIT_CHILD="+path, "GOWT_QUIT_MARK="+mark)
			done := make(chan error, 1)
			var out []byte
			go func() {
				var err error
				out, err = cmd.CombinedOutput()
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("the app did not exit with 0: %v\n%s", err, out)
				}
				got, _ := os.ReadFile(mark)
				if string(got) != "closed\ndisposed\n" {
					t.Fatalf("the child left %q, want OnClose, then the display dispose\n%s", got, out)
				}
			case <-time.After(40 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatal("the app did not exit")
			}
		})
	}
}

func TestQuitExitChild(t *testing.T) {
	path := os.Getenv("GOWT_QUIT_CHILD")
	if path == "" {
		t.Skip("run by TestQuitExitsCleanly")
	}
	mark := func(s string) {
		f, err := os.OpenFile(os.Getenv("GOWT_QUIT_MARK"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(s + "\n")
			f.Close()
		}
	}
	fellBack := false
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.Unwrap().AddListener(swt.Dispose, quitListener(func() bool { mark("disposed"); return true }))
			w := app.Window("Untitled *")
			w.OnClose(func() bool {
				// The box is dismissed by a timer; the answer does not matter, the close goes on.
				app.After(500*time.Millisecond, func() { cocoa.NSApplicationSharedApplication().StopModal() })
				w.MessageBox(Message{Title: "Save changes?", Text: "Unsaved.", Icon: IconQuestion, Buttons: ButtonsYesNoCancel})
				mark("closed")
				return true
			})
			w.Show()
			app.After(500*time.Millisecond, func() {
				if path == "menu-async" {
					app.Async(func() {
						cocoa.NSApplicationSharedApplication().SendAction(cocoa.OSSel_applicationShouldTerminate_, nil, nil)
					})
					return
				}
				if path == "terminate" {
					app := cocoa.NSApplicationSharedApplication()
					d := cocoa.OSObjc_msgSend(app.Id, cocoa.OSSel_registerName("delegate"))
					resp := cocoa.OSObjc_msgSendOverload44(d, cocoa.OSSel_registerName("respondsToSelector:"),
						cocoa.OSSel_registerName("applicationWillTerminate:"))
					fmt.Fprintf(os.Stderr, "diag: class_addMethod=%v delegate responds to applicationWillTerminate:=%v\n", willEndAdded, resp&0xff != 0)
					cocoa.NSApplicationSharedApplication().Terminate(nil)
					return
				}
				cocoa.NSApplicationSharedApplication().SendAction(cocoa.OSSel_applicationShouldTerminate_, nil, nil)
			})
			app.After(15*time.Second, func() { fellBack = true; app.Quit() })
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if fellBack {
		t.Fatal("the quit did not end Run before the fallback timer")
	}
}
