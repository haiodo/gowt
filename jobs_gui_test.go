package gowt

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
)

// jobsRun runs an app with one window; check runs in a goroutine and ends the app when it returns.
func jobsRun(t *testing.T, check func(app *App)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.Window("jobs").Show()
			app.After(20*time.Second, app.Quit) // a stuck check must not hang the run
			go func() {
				defer app.Quit()
				check(app)
			}()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// onUI runs f on the UI thread and returns its result.
func onUI[T any](app *App, f func() T) (v T) {
	app.Sync(func() { v = f() })
	return v
}

func pendingNow(app *App) int {
	return onUI(app, func() int { app.Flush(); return app.Pending() })
}

func TestJobThenAndWait(t *testing.T) {
	jobsRun(t, func(app *App) {
		boom := errors.New("boom")
		thenErr := make(chan error, 2)
		onUIThread := make(chan bool, 2)
		j := app.Job("work", func(ctx context.Context, p *JobProgress) error {
			time.Sleep(30 * time.Millisecond)
			return boom
		}).Then(func(err error) {
			onUIThread <- jrt.Goid() == app.jobs.uiID
			thenErr <- err
		})
		if err := j.Wait(); err != boom {
			t.Errorf("Wait: %v", err)
		}
		// Wait returns after the Then callbacks ran.
		if err := <-thenErr; err != boom || !<-onUIThread {
			t.Errorf("Then err %v", err)
		}
		// Then after completion still runs, on the UI thread.
		j.Then(func(err error) {
			onUIThread <- jrt.Goid() == app.jobs.uiID
			thenErr <- err
		})
		select {
		case err := <-thenErr:
			if err != boom || !<-onUIThread {
				t.Errorf("late Then err %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("a Then added after completion did not run")
		}
		if n := pendingNow(app); n != 0 {
			t.Errorf("Pending after the job: %d", n)
		}
	})
}

func TestJobCancel(t *testing.T) {
	jobsRun(t, func(app *App) {
		started := make(chan struct{})
		j := app.Job("blocked", func(ctx context.Context, p *JobProgress) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
		<-started
		if n := pendingNow(app); n != 1 {
			t.Errorf("Pending with a running job: %d", n)
		}
		j.Cancel()
		if err := j.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait after Cancel: %v", err)
		}
	})
}

func TestJobPanicGoesToOnError(t *testing.T) {
	jobsRun(t, func(app *App) {
		var mu sync.Mutex
		var reported []error
		app.Sync(func() {
			app.OnError(func(err error) { mu.Lock(); reported = append(reported, err); mu.Unlock() })
		})
		var thenErr error
		j := app.Job("bad", func(ctx context.Context, p *JobProgress) error { panic("kaboom") })
		j.Then(func(err error) { thenErr = err })
		err := j.Wait()
		mu.Lock()
		defer mu.Unlock()
		if err == nil || !strings.Contains(err.Error(), "kaboom") || thenErr == nil {
			t.Errorf("Wait %v, Then %v", err, thenErr)
		}
		if len(reported) != 1 || !strings.Contains(reported[0].Error(), "kaboom") {
			t.Errorf("OnError got %v", reported)
		}
	})
}

func TestJobWaitOnUIThreadPanics(t *testing.T) {
	jobsRun(t, func(app *App) {
		j := app.Job("quick", func(ctx context.Context, p *JobProgress) error { return nil })
		got := onUI(app, func() (r any) {
			defer func() { r = recover() }()
			j.Wait()
			return nil
		})
		if s, _ := got.(string); !strings.Contains(s, "UI thread") {
			t.Errorf("Wait on the UI thread: %v", got)
		}
		j.Wait()
	})
}

func TestPendingCounts(t *testing.T) {
	jobsRun(t, func(app *App) {
		if n := pendingNow(app); n != 0 {
			t.Fatalf("Pending at rest: %d", n)
		}
		release := make(chan struct{})
		app.Job("slow", func(ctx context.Context, p *JobProgress) error { <-release; return nil })
		stopPoll := app.Job("poll", func(ctx context.Context, p *JobProgress) error { <-ctx.Done(); return nil },
			BackgroundJob())
		done := app.Track("send")
		var stop func()
		app.Sync(func() { // After and Every are UI-thread calls
			app.After(500*time.Millisecond, func() {})
			app.After(5*time.Second, func() {})
			stop = app.Every(50*time.Millisecond, func() {})
		})

		if n := pendingNow(app); n != 3 {
			t.Errorf("Pending: %d, want the job, the track and the short timer", n)
		}
		what := onUI(app, app.waitingOn)
		for _, want := range []string{`job "slow"`, `track "send"`, "1 timer"} {
			if !strings.Contains(what, want) {
				t.Errorf("waitingOn %q lacks %q", what, want)
			}
		}
		if strings.Contains(what, "poll") {
			t.Errorf("waitingOn %q names the background job", what)
		}
		if got := app.Jobs(); len(got) != 3 || !got[1].Background || !got[2].Track {
			t.Errorf("Jobs: %+v", got)
		}
		done()
		done()
		if n := pendingNow(app); n != 2 {
			t.Errorf("Pending after done: %d", n)
		}
		close(release)
		time.Sleep(700 * time.Millisecond)
		if n := pendingNow(app); n != 0 {
			t.Errorf("Pending after the job and the short timer: %d (%s)", n, onUI(app, app.waitingOn))
		}
		stop()
		stopPoll.Cancel()
		stopPoll.Wait()
	})
}

func TestOnJobsCoalesces(t *testing.T) {
	jobsRun(t, func(app *App) {
		var mu sync.Mutex
		var calls [][]JobInfo
		app.OnJobs(func(js []JobInfo) { mu.Lock(); calls = append(calls, js); mu.Unlock() })
		// The UI thread is blocked while the job reports 50 steps: they make one call.
		var j *Job
		app.Sync(func() {
			j = app.Job("steps", func(ctx context.Context, p *JobProgress) error {
				p.Total(50)
				for range 50 {
					p.Step(1)
				}
				p.Text("end")
				return nil
			})
			<-j.ran
		})
		j.Wait()
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		if len(calls) == 0 || len(calls) > 3 {
			t.Fatalf("%d OnJobs calls for 52 changes", len(calls))
		}
		first, last := calls[0], calls[len(calls)-1]
		if len(first) != 1 || first[0].Total != 50 || first[0].Done != 50 || first[0].Text != "end" {
			t.Errorf("first call: %+v", first)
		}
		if len(last) != 0 {
			t.Errorf("last call: %+v, want no jobs", last)
		}
	})
}

func TestAfterCancel(t *testing.T) {
	jobsRun(t, func(app *App) {
		var cancelled, kept, twice atomic.Int32
		var cancel, cancelKept func()
		app.Sync(func() {
			cancel = app.After(200*time.Millisecond, func() { cancelled.Add(1) })
			cancelKept = app.After(100*time.Millisecond, func() { kept.Add(1) })
			app.After(150*time.Millisecond, func() { twice.Add(1) })()
		})
		if n := pendingNow(app); n != 2 {
			t.Errorf("Pending with two live short timers: %d", n)
		}
		cancel()
		cancel()
		if n := pendingNow(app); n != 1 {
			t.Errorf("Pending after cancel: %d, want 1", n)
		}
		time.Sleep(500 * time.Millisecond)
		if cancelled.Load() != 0 || twice.Load() != 0 || kept.Load() != 1 {
			t.Errorf("ran: cancelled %d, cancelled at once %d, kept %d, want 0, 0, 1", cancelled.Load(), twice.Load(), kept.Load())
		}
		cancelKept() // after it ran: no effect
		if n := pendingNow(app); n != 0 {
			t.Errorf("Pending at rest: %d", n)
		}
	})
}
