package gowt

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
)

// shortTimer is the limit under which App.After counts as pending work for App.Pending and the QA
// driver's settle; a longer After and every Every are not waited for.
const shortTimer = time.Second

// jobSet is the work an App knows about: Jobs, Tracks and short timers.
type jobSet struct {
	mu     sync.Mutex
	active []*Job // in start order, until the UI thread has run the Then callbacks
	onJobs func([]JobInfo)
	armed  bool // a call of onJobs is queued
	timers atomic.Int32
	ctx    context.Context // ends with Run
	end    context.CancelFunc
	ended  chan struct{}
	uiID   uint64
}

func (s *jobSet) init() {
	s.ctx, s.end = context.WithCancel(context.Background())
	s.ended = make(chan struct{})
	s.uiID = jrt.Goid()
}

func (s *jobSet) close() {
	s.end()
	close(s.ended)
}

// Job is work App.Job runs in a goroutine.
type Job struct {
	app        *App
	name       string
	background bool
	track      bool
	started    time.Time
	cancel     context.CancelFunc
	ran        chan struct{} // closed when fn has returned
	fin        chan struct{} // closed when the Then callbacks have run

	// guarded by app.jobs.mu
	done, total int
	text        string
	err         error
	panicked    bool
	finished    bool
	thens       []func(error)
}

// JobOption changes how App.Job treats a job.
type JobOption func(*Job)

// BackgroundJob marks perpetual work, a poll or a sync loop: it shows in Jobs but App.Pending and the
// QA driver's settle do not wait for it. (Background is the colour option.)
func BackgroundJob() JobOption { return func(j *Job) { j.background = true } }

// JobProgress is how a Job reports itself to Jobs and OnJobs; its methods are safe from the job's
// goroutine.
type JobProgress struct{ j *Job }

// Total sets the number of steps the job has; 0 means unknown.
func (p *JobProgress) Total(n int) { p.update(func(j *Job) { j.total = n }) }

// Step adds n finished steps.
func (p *JobProgress) Step(n int) { p.update(func(j *Job) { j.done += n }) }

// Text sets what the job is doing now.
func (p *JobProgress) Text(s string) { p.update(func(j *Job) { j.text = s }) }

func (p *JobProgress) update(f func(*Job)) {
	a := p.j.app
	a.jobs.mu.Lock()
	f(p.j)
	a.jobs.mu.Unlock()
	a.jobsChanged()
}

// JobInfo is one running Job or Track as App.Jobs and App.OnJobs report it.
type JobInfo struct {
	Name       string
	Background bool
	Track      bool // work from App.Track, not a Job
	Started    time.Time
	Done       int // JobProgress.Step so far
	Total      int // JobProgress.Total; 0 unknown
	Text       string
}

// Job runs fn in a goroutine. ctx is cancelled by Job.Cancel and when Run ends; fn should return
// then. A panic in fn goes to OnError and counts as fn's error. While fn runs, and until its Then
// callbacks have run on the UI thread, App.Pending counts the job, unless it is a BackgroundJob.
// Safe from any goroutine.
func (a *App) Job(name string, fn func(ctx context.Context, p *JobProgress) error, opts ...JobOption) *Job {
	ctx, cancel := context.WithCancel(a.jobs.ctx)
	j := &Job{app: a, name: name, started: time.Now(), cancel: cancel,
		ran: make(chan struct{}), fin: make(chan struct{})}
	for _, o := range opts {
		o(j)
	}
	a.addJob(j)
	go func() {
		err := j.run(ctx, fn)
		cancel()
		a.jobs.mu.Lock()
		j.err = err
		a.jobs.mu.Unlock()
		close(j.ran)
		a.Async(func() { a.finishJob(j) })
	}()
	return j
}

func (j *Job) run(ctx context.Context, fn func(context.Context, *JobProgress) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w (job %q)\n%s", panicError(r), j.name, debug.Stack())
			j.app.jobs.mu.Lock()
			j.panicked = true
			j.app.jobs.mu.Unlock()
		}
	}()
	return fn(ctx, &JobProgress{j})
}

// finishJob runs on the UI thread once fn has returned.
func (a *App) finishJob(j *Job) {
	a.jobs.mu.Lock()
	j.finished = true
	thens, err, panicked := j.thens, j.err, j.panicked
	j.thens = nil
	a.jobs.mu.Unlock()
	if panicked {
		a.report(err)
	}
	for _, f := range thens {
		a.callThen(f, err)
	}
	a.removeJob(j)
	close(j.fin)
}

func (a *App) callThen(f func(error), err error) {
	defer func() {
		if r := recover(); r != nil {
			a.fail(r)
		}
	}()
	f(err)
}

// Then calls f on the UI thread with the error fn returned, once fn has returned. Called after that,
// it queues f at once. Several Then calls run in order. Returns j.
func (j *Job) Then(f func(err error)) *Job {
	a := j.app
	a.jobs.mu.Lock()
	if !j.finished {
		j.thens = append(j.thens, f)
		a.jobs.mu.Unlock()
		return j
	}
	err := j.err
	a.jobs.mu.Unlock()
	a.Async(func() { a.callThen(f, err) })
	return j
}

// Cancel cancels the ctx of fn and returns; it does not wait for fn.
func (j *Job) Cancel() { j.cancel() }

// Wait blocks until fn has returned and the Then callbacks have run, and returns fn's error. If Run
// ends first it waits for fn only. From any goroutine except the UI thread, which could never run
// the callbacks: there it panics.
func (j *Job) Wait() error {
	a := j.app
	if jrt.Goid() == a.jobs.uiID {
		panic("gowt: Job.Wait called on the UI thread; use Then")
	}
	select {
	case <-j.fin:
	case <-a.jobs.ended:
		<-j.ran
	}
	a.jobs.mu.Lock()
	defer a.jobs.mu.Unlock()
	return j.err
}

// Track counts work gowt does not start, a request through the app's own client or a callback of a
// core library, as outstanding for App.Pending and the QA driver's settle until done is called.
// done is safe from any goroutine and idempotent. Call it after the callback's result reached the
// UI: app.Async(func() { show(r); done() }).
func (a *App) Track(name string) (done func()) {
	j := &Job{app: a, name: name, track: true, started: time.Now()}
	a.addJob(j)
	var once sync.Once
	return func() {
		once.Do(func() { a.removeJob(j) })
	}
}

func (a *App) addJob(j *Job) {
	a.jobs.mu.Lock()
	a.jobs.active = append(a.jobs.active, j)
	a.jobs.mu.Unlock()
	a.jobsChanged()
}

func (a *App) removeJob(j *Job) {
	a.jobs.mu.Lock()
	for i, x := range a.jobs.active {
		if x == j {
			a.jobs.active = append(a.jobs.active[:i:i], a.jobs.active[i+1:]...)
			break
		}
	}
	a.jobs.mu.Unlock()
	a.jobsChanged()
}

// Jobs lists the running Jobs and Tracks, background ones included, oldest first. Safe from any
// goroutine.
func (a *App) Jobs() []JobInfo {
	a.jobs.mu.Lock()
	defer a.jobs.mu.Unlock()
	var out []JobInfo
	for _, j := range a.jobs.active {
		out = append(out, JobInfo{Name: j.name, Background: j.background, Track: j.track,
			Started: j.started, Done: j.done, Total: j.total, Text: j.text})
	}
	return out
}

// OnJobs sets f to be called on the UI thread with App.Jobs when a job or track starts or ends or
// a JobProgress changes, for a status spinner. Changes are coalesced: at most one call per
// event-loop turn. OnJobs(nil) removes it.
func (a *App) OnJobs(f func([]JobInfo)) {
	a.jobs.mu.Lock()
	a.jobs.onJobs = f
	a.jobs.mu.Unlock()
}

func (a *App) jobsChanged() {
	a.jobs.mu.Lock()
	if a.jobs.onJobs == nil || a.jobs.armed {
		a.jobs.mu.Unlock()
		return
	}
	a.jobs.armed = true
	a.jobs.mu.Unlock()
	a.Async(func() {
		a.jobs.mu.Lock()
		a.jobs.armed = false
		f := a.jobs.onJobs
		a.jobs.mu.Unlock()
		if f != nil {
			a.callJobs(f)
		}
	})
}

func (a *App) callJobs(f func([]JobInfo)) {
	defer func() {
		if r := recover(); r != nil {
			a.fail(r)
		}
	}()
	f(a.Jobs())
}

// outstanding counts what Pending waits for besides rx: non-background jobs, tracks and short timers.
func (a *App) outstanding() int {
	a.jobs.mu.Lock()
	defer a.jobs.mu.Unlock()
	n := int(a.jobs.timers.Load())
	for _, j := range a.jobs.active {
		if !j.background {
			n++
		}
	}
	return n
}

// waitingOn names what outstanding counts: `job "Sync" 4.2s`, `track "Send" 1.0s`, `1 timer`.
func (a *App) waitingOn() string {
	a.jobs.mu.Lock()
	defer a.jobs.mu.Unlock()
	var parts []string
	for _, j := range a.jobs.active {
		if j.background {
			continue
		}
		kind := "job"
		if j.track {
			kind = "track"
		}
		parts = append(parts, fmt.Sprintf("%s %q %.1fs", kind, j.name, time.Since(j.started).Seconds()))
	}
	switch n := a.jobs.timers.Load(); n {
	case 0:
	case 1:
		parts = append(parts, "1 timer")
	default:
		parts = append(parts, fmt.Sprintf("%d timers", n))
	}
	return strings.Join(parts, ", ")
}
