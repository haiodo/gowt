package gowt

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt/internal/shot"
)

// armSnap implements GOWT_SNAP=<file.png>: after GOWT_SNAP_DELAY milliseconds (default 800) of the
// event loop it writes the first window as a PNG and quits. It returns the tick Run calls between
// events; the capture runs there, not in a timer callback, because Wine blocked on a capture taken
// from WM_TIMER.
func (a *App) armSnap(path string, errp *error) func() {
	delay := 800 * time.Millisecond
	if s := os.Getenv("GOWT_SNAP_DELAY"); s != "" {
		if ms, err := strconv.Atoi(s); err == nil {
			delay = time.Duration(ms) * time.Millisecond
		}
	}
	due := time.Now().Add(delay)
	go func() {
		defer func() { recover() }() // Wake panics once the display is disposed
		for !a.display.IsDisposed() {
			time.Sleep(100 * time.Millisecond)
			a.display.Wake()
		}
	}()
	return func() {
		if a.quit.Load() || time.Now().Before(due) {
			return
		}
		shells := a.display.GetShells()
		if len(shells) == 0 {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: no window")
		} else if err := shot.WindowPNG(shotHandle(shells[0]), path); err != nil {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: %w", err)
		}
		a.quit.Store(true)
	}
}
