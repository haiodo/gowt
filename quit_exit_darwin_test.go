package gowt

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

// TestQuitExitsCleanly quits a dirty window the way Cmd+Q does, after a message box from OnClose, and
// checks that the process ends with status 0. The child runs in its own process because a crash on
// the way out kills it. GOWT_QUIT_PATH picks "menu" (the menu item's action, a direct call of
// applicationShouldTerminate:) or "terminate" (NSApp terminate:, the Dock and logout).
func TestQuitExitsCleanly(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, path := range []string{"menu", "menu-async", "terminate"} {
		t.Run(path, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run", "^TestQuitExitChild$")
			cmd.Env = append(os.Environ(), "GOWT_QUIT_CHILD="+path)
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
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("Untitled *")
			w.OnClose(func() bool {
				// The box is dismissed by a timer; the answer does not matter, the close goes on.
				app.After(500*time.Millisecond, func() { cocoa.NSApplicationSharedApplication().StopModal() })
				w.MessageBox(Message{Title: "Save changes?", Text: "Unsaved.", Icon: IconQuestion, Buttons: ButtonsYesNoCancel})
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
					cocoa.NSApplicationSharedApplication().Terminate(nil)
					return
				}
				cocoa.NSApplicationSharedApplication().SendAction(cocoa.OSSel_applicationShouldTerminate_, nil, nil)
			})
			app.After(30*time.Second, app.Quit)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	// Reaching here means Run returned; the exit status is the test's.
}
