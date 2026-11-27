//go:build darwin

package gowt

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

// TestCloseButtonQuits clicks the real AppKit close button, which differs from the QA Shell.Close.
func TestCloseButtonQuits(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, scenario := range []string{"single", "two-windows", "two-windows-hidden"} {
		t.Run(scenario, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run", "^TestCloseButtonChild$")
			mark := filepath.Join(t.TempDir(), "mark")
			cmd.Env = append(os.Environ(), "GOWT_CLOSE_CHILD="+scenario, "GOWT_CLOSE_MARK="+mark)
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
				if got, _ := os.ReadFile(mark); string(got) != "returned\n" {
					t.Fatalf("Run did not return within 5 s: %q\n%s", got, out)
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("the app process did not finish\n%s", out)
			}
		})
	}
}

func TestCloseButtonChild(t *testing.T) {
	scenario := os.Getenv("GOWT_CLOSE_CHILD")
	if scenario == "" {
		t.Skip("run by TestCloseButtonQuits")
	}
	mark := func(s string) {
		f, err := os.OpenFile(os.Getenv("GOWT_CLOSE_MARK"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(s + "\n")
			f.Close()
		}
	}
	timedOut := false
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("Rex")
			var second *Window
			quitting := false
			if scenario != "single" {
				second = app.Window("Settings")
				second.OnClose(func() bool {
					if quitting {
						return true
					}
					second.SetVisible(false)
					return false
				})
				w.OnClose(func() bool {
					quitting = true
					second.Close()
					return true
				})
			}
			w.Show()
			if scenario == "two-windows-hidden" {
				second.Show()
			}
			cocoa.NSApplicationSharedApplication().ActivateIgnoringOtherApps(true)
			click := func() {
				clickCloseButton(w)
				app.After(5*time.Second, func() {
					timedOut = true
					app.Quit()
				})
			}
			if scenario == "two-windows-hidden" {
				app.After(300*time.Millisecond, func() {
					clickCloseButton(second)
					app.After(400*time.Millisecond, click)
				})
			} else {
				app.After(300*time.Millisecond, click)
			}
		})
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if timedOut {
		mark("timeout")
		return
	}
	mark("returned")
}

func clickCloseButton(w *Window) {
	window := w.Unwrap().View.Window()
	if window == nil {
		return
	}
	button := cocoa.OSObjc_msgSendOverload44(window.Id, cocoa.OSSel_registerName("standardWindowButton:"), 0)
	if button != 0 {
		cocoa.OSObjc_msgSendOverload44(button, cocoa.OSSel_registerName("performClick:"), 0)
	}
}
