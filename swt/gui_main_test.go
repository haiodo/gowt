package swt

import (
	"os"
	"testing"
)

// uiRun runs f where the toolkit may be used. macOS needs the main thread; its TestMain replaces
// this in widgets_viewtable_gui_darwin_test.go.
var uiRun = func(f func()) { f() }

// uiT is the t of a test body that runs on the UI thread, where FailNow must not end the goroutine.
type uiT struct {
	*testing.T
	cleanups []func()
}

type uiAbort struct{}

func (t *uiT) Fatalf(format string, args ...any) {
	t.Helper()
	t.Errorf(format, args...)
	panic(uiAbort{})
}

func (t *uiT) Cleanup(f func()) { t.cleanups = append(t.cleanups, f) }

// onUI runs body on the UI thread when GOWT_GUI_TEST is set; otherwise the test is skipped.
func onUI(t *testing.T, body func(t *uiT)) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var crash any
	uiRun(func() {
		u := &uiT{T: t}
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(uiAbort); !ok {
					crash = r
				}
			}
			for i := len(u.cleanups) - 1; i >= 0; i-- {
				u.cleanups[i]()
			}
		}()
		body(u)
	})
	if crash != nil {
		panic(crash)
	}
}
