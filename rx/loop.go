package rx

import (
	"sync"
	"sync/atomic"

	"github.com/haiodo/gowt/internal/jrt"
)

var (
	uiThread atomic.Uint64 // goroutine id of the attached loop; 0: not attached, nothing is checked

	// loopMu guards wake. Post calls wake outside postMu: the UI thread drains posts while it
	// holds the SWT monitor, and wake (Display.AsyncExec) takes that monitor.
	loopMu sync.RWMutex
	wake   func()

	postMu    sync.Mutex
	postList  []poster
	postDrain []poster
	postArmed bool // a tick is on its way, or will be sent when a loop attaches
)

// Attach binds rx to an event loop and makes the calling goroutine its UI thread. prepare gets
// the function that must run on that thread and returns wake, which queues one call of it and
// is safe from any goroutine. gowt.Run calls Attach; detach unbinds before the loop stops.
func Attach(prepare func(tick func()) (wake func())) (detach func()) {
	w := prepare(tick)
	loopMu.Lock()
	prevWake, prevThread := wake, uiThread.Load()
	wake = w
	uiThread.Store(jrt.Goid())
	loopMu.Unlock()
	postMu.Lock()
	pending := postArmed
	postMu.Unlock()
	if pending || len(queue) > 0 {
		w()
	}
	return func() {
		loopMu.Lock()
		wake = prevWake
		uiThread.Store(prevThread)
		loopMu.Unlock()
		scheduled = false
	}
}

// Pending counts the work waiting for a tick: stale effects and State.Post values not applied yet.
// For a test that waits for the UI to go quiet. UI thread only.
func Pending() int {
	mustUI("Pending")
	postMu.Lock()
	n := len(postList)
	postMu.Unlock()
	return n + len(queue)
}

// wakeLoop asks the attached loop for a tick; false when no loop is attached.
func wakeLoop() bool {
	loopMu.RLock()
	defer loopMu.RUnlock()
	if wake == nil {
		return false
	}
	wake()
	return true
}

func tick() {
	drainPosts()
	Flush()
}

func mustUI(op string) {
	if id := uiThread.Load(); id != 0 && id != jrt.Goid() {
		panic("rx: " + op + " called off the UI thread; other goroutines hand values over with State.Post")
	}
}

type poster interface{ apply() }

// Post stores v for the UI thread; it is the only call that is safe from any goroutine. Posts
// before the next tick keep only the last value per State and are applied in one batch. Before
// a loop is attached, and after it is detached, values wait for the next Attach.
func (s *State[T]) Post(v T) {
	postMu.Lock()
	s.pending = v
	if !s.posted {
		s.posted = true
		postList = append(postList, s)
	}
	arm := !postArmed
	postArmed = true
	postMu.Unlock()
	if arm {
		wakeLoop()
	}
}

func (s *State[T]) apply() {
	postMu.Lock()
	v := s.pending
	var zero T
	s.pending, s.posted = zero, false
	postMu.Unlock()
	s.Set(v)
}

func drainPosts() {
	postMu.Lock()
	postList, postDrain = postDrain[:0], postList
	postArmed = false
	postMu.Unlock()
	if len(postDrain) == 0 {
		return
	}
	batchDepth++
	defer func() {
		clear(postDrain)
		batchDepth--
	}()
	for _, p := range postDrain {
		p.apply()
	}
}
