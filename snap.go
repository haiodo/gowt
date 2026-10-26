package gowt

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/internal/shot"
)

// snapFrameWait bounds the wait for the gfx views of the window; a view that animates all the time
// never gets quiet.
const snapFrameWait = 3 * time.Second

// armSnap implements GOWT_SNAP=<file.png>: after GOWT_SNAP_DELAY milliseconds (default 800) of the
// event loop, and once every web view has finished loading, it writes the first window as a PNG and quits. It returns the tick Run calls between
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
	// A web view paints after its load finishes: wait for that, not for a longer fixed delay.
	limit := due.Add(30 * time.Second)
	var settledAt time.Time
	var frames func() bool // gfx views have presented a frame since the capture was due
	var framesBy time.Time
	go func() {
		defer func() { recover() }() // Wake panics once the display is disposed
		for !a.display.IsDisposed() {
			time.Sleep(100 * time.Millisecond)
			a.display.Wake()
		}
	}()
	var clearedAt time.Time
	moves, placed := 0, false
	debug := os.Getenv("GOWT_SNAP_DEBUG") == "1"
	why := ""
	wait := func(reason string) { // GOWT_SNAP_DEBUG=1 prints each change of what the capture waits for
		if debug && reason != why {
			fmt.Fprintf(os.Stderr, "gowt: snap waits: %s\n", reason)
		}
		why = reason
	}
	// Whatever else is waited for, the capture is taken this long after it was due.
	hard := due.Add(40 * time.Second)
	return func() {
		now := time.Now()
		forced := now.After(hard)
		// The window can leave the Retina screen when it opens: look again once the capture is due.
		if shells := a.display.GetShells(); len(shells) > 0 && !placed && !now.Before(due) {
			if moves < 3 && shot.PlaceOnRetina(shotHandle(shells[0])) {
				moves++
				clearedAt = time.Time{}
				due = now.Add(time.Second)
			} else {
				placed = true
			}
		}
		if a.quit.Load() || now.Before(due) {
			wait("delay")
			return
		}
		if now.Before(limit) && !forced {
			if !shot.Settled() {
				wait("web view")
				return
			}
			if settledAt.IsZero() {
				settledAt = now
			}
			hold := 500 * time.Millisecond
			if shot.HasWebView() {
				hold = 2 * time.Second
			}
			if now.Sub(settledAt) < hold {
				wait("web view paint")
				return
			}
		}
		shells := a.display.GetShells()
		if frames == nil && len(shells) > 0 {
			frames, framesBy = qaview.Request(shells[0]), now.Add(snapFrameWait)
		}
		if len(shells) > 0 && clearedAt.IsZero() {
			shot.ClearFocus(shotHandle(shells[0]))
			clearedAt = now
		}
		if frames != nil && !frames() && now.Before(framesBy) && !forced {
			wait("gfx frame")
			return
		}
		if now.Sub(clearedAt) < 500*time.Millisecond && !forced {
			wait("focus clear")
			return
		}
		if debug {
			fmt.Fprintf(os.Stderr, "gowt: snap captures (forced %v)\n", forced)
		}
		if len(shells) == 0 {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: no window")
		} else if err := shot.WindowPNG(shotHandle(shells[0]), path); err != nil {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: %w", err)
		} else if err := qaview.OverlayPNG(shells[0], path); err != nil {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: %w", err)
		} else if err := shot.Finish(path); err != nil {
			*errp = fmt.Errorf("gowt: GOWT_SNAP: %w", err)
		}
		a.quit.Store(true)
	}
}
