// Package rx is reactive state in the style of Svelte 5 runes. State is a writable signal,
// Derived a lazy memoized computation over other signals, Effect a side effect that runs again
// when something it read has changed. Dependencies are whatever a Derived or an Effect reads
// with Get during its last run; nothing is declared.
//
// Everything except State.Post belongs to one goroutine, the UI thread. gowt.Run attaches rx to
// its event loop: effects made stale by a Set run once per batch, after the current event
// handler, and Post hands values over from other goroutines with one hop per batch.
package rx

import (
	"fmt"
	"os"
	"runtime/debug"
	"slices"
)

const (
	clean uint8 = iota
	check       // a source upstream changed; recompute only if a direct source really changed
	dirty
)

const (
	kindState uint8 = iota
	kindDerived
	kindEffect
	kindScope
)

type node struct {
	kind      uint8
	state     uint8
	queued    bool
	disposed  bool
	running   bool // inside its own execute: a Derived that reads itself is a cycle
	stale     bool // its last run panicked; recompute on the next read whatever the state says
	untracked bool // the next run reads without subscribing (a Derived nobody observes)

	idx       int      // sources matched in order during the current run
	diverged  bool     // the current run read a source out of the old order
	sources   []*node  // what the last run read, in read order
	extra     []*node  // sources read after the first mismatch of the current run
	extraAt   []uint64 // epoch at which each extra source was read
	observers []*node

	exec     func()      // derived: recompute; effect: rerun
	catch    func(r any) // Catch: takes the panics of the effects below it
	parent   *node
	children []*node  // effects, deriveds and scopes created while this node ran
	cleanups []func() // OnCleanup funcs of the last run
	depth    int      // owners above it: a flush runs parents before the children they may dispose
	changed  uint64   // epoch of the last change of value (State, Derived)
	start    uint64   // epoch when the current run began
	verified uint64   // Derived read with nobody observing: epoch its value was computed at
}

var (
	observer   *node // the derived or effect whose reads are being tracked
	owner      *node // the node that owns nodes created now
	queue      []*node
	spare      []*node
	flushing   bool
	batchDepth int
	scheduled  bool // a tick is on its way to the UI thread
	inEffect   int  // effects running now; Flush inside one would re-enter it
	epoch      uint64
)

func track(src *node) {
	o := observer
	if o == nil {
		return
	}
	if !o.diverged && o.idx < len(o.sources) && o.sources[o.idx] == src {
		o.idx++
		return
	}
	// Repeated reads of one source in a run must not count as a new dependency.
	if slices.Contains(o.sources[:o.idx], src) || slices.Contains(o.extra, src) {
		return
	}
	o.diverged = true
	o.extra = append(o.extra, src)
	o.extraAt = append(o.extraAt, epoch)
}

// execute tracks what fn reads as n's sources. After a panic n keeps old and new sources, so it
// still hears every change that could fix it.
func (n *node) execute(fn func()) {
	prevObs, prevOwn := observer, owner
	observer, owner = n, n
	if n.untracked {
		observer, n.untracked = nil, false
	}
	n.idx, n.diverged, n.extra, n.extraAt, n.running, n.start = 0, false, n.extra[:0], n.extraAt[:0], true, epoch
	if n.kind == kindEffect {
		inEffect++
	}
	ok := false
	defer func() {
		observer, owner = prevObs, prevOwn
		n.running = false
		if n.kind == kindEffect {
			inEffect--
		}
		if !ok {
			n.keepSources()
			n.stale = n.kind == kindDerived
		}
	}()
	fn()
	n.stale = false
	n.commit()
	ok = true
}

// commit makes the sources read in this run the subscription set. A run that read the same
// sources in the same order (the steady state) allocates nothing.
func (n *node) commit() {
	if n.disposed {
		n.extra = n.extra[:0]
		return
	}
	if !n.diverged && n.idx == len(n.sources) {
		return
	}
	old := n.sources
	next := make([]*node, 0, n.idx+len(n.extra))
	next = append(next, old[:n.idx]...)
	next = append(next, n.extra...)
	missed := false
	for i, s := range n.extra {
		if !slices.Contains(old, s) {
			missed = n.subscribe(s, n.extraAt[i]) || missed
		}
	}
	n.sources, n.extra, n.diverged = next, n.extra[:0], false
	for _, s := range old[n.idx:] {
		if !slices.Contains(next, s) {
			s.removeObserver(n)
		}
	}
	if missed {
		n.mark(dirty)
	}
}

// subscribe adds n to s's observers and reports whether s changed after n read it at epoch
// readAt: a Set in between marked observers that did not include n yet.
func (n *node) subscribe(s *node, readAt uint64) bool {
	s.addObserver(n)
	return s.changed > readAt || (s.kind == kindDerived && s.state != clean)
}

func (n *node) keepSources() {
	missed := false
	if !n.disposed {
		for i, s := range n.extra {
			if !slices.Contains(n.sources, s) {
				n.sources = append(n.sources, s)
				missed = n.subscribe(s, n.extraAt[i]) || missed
			}
		}
	}
	n.extra, n.diverged = n.extra[:0], false
	if missed {
		n.mark(dirty)
	}
}

func (n *node) addObserver(o *node) { n.observers = append(n.observers, o) }

// removeObserver unsubscribes o. A Derived nobody observes any more lets go of its own sources,
// so a long-lived State does not keep the deriveds of disposed bindings alive.
func (n *node) removeObserver(o *node) {
	i := slices.Index(n.observers, o)
	if i < 0 {
		return
	}
	last := len(n.observers) - 1
	n.observers[i] = n.observers[last]
	n.observers[last] = nil
	n.observers = n.observers[:last]
	if last == 0 && n.kind == kindDerived && !n.running {
		n.unlink()
	}
}

// unlink drops the subscriptions; the next read recomputes and subscribes again.
func (n *node) unlink() {
	srcs := n.sources
	n.sources = nil
	n.state = dirty
	for _, s := range srcs {
		s.removeObserver(n)
	}
}

func (n *node) mark(st uint8) {
	prev := n.state
	if prev >= st {
		return
	}
	n.state = st
	if prev != clean {
		return // already propagated when it left clean
	}
	if n.kind == kindEffect {
		n.enqueue()
		return
	}
	for _, o := range n.observers {
		o.mark(check)
	}
}

func (n *node) enqueue() {
	if n.queued {
		return
	}
	n.queued = true
	queue = append(queue, n)
	if !scheduled && !flushing && batchDepth == 0 {
		scheduled = wakeLoop()
	}
}

// update brings n up to date: a check node asks its derived sources first and reruns only when
// one of them produced a different value.
func (n *node) update() {
	if n.state == check {
		for _, s := range n.sources {
			if s.kind == kindDerived {
				s.update()
				if n.state == dirty {
					break
				}
			}
		}
	}
	if n.state == dirty || n.stale {
		n.state = clean // before exec, so a write from inside the run marks it again
		n.exec()
		return
	}
	n.state = clean
}

// settle repairs the marks a panic left upstream of n: clean and stale means the next read
// recomputes and the next Set propagates again.
func (n *node) settle() {
	for _, s := range n.sources {
		if s.kind == kindDerived && s.state != clean {
			s.state, s.stale = clean, true
			s.settle()
		}
	}
	n.state = clean
}

func (n *node) adopt() {
	if owner != nil {
		n.parent = owner
		n.depth = owner.depth + 1
		owner.children = append(owner.children, n)
	}
}

// release disposes what the last run created and runs its cleanups. A panic in one of them does
// not skip the rest; the first one is raised again at the end.
func (n *node) release() {
	if len(n.children) == 0 && len(n.cleanups) == 0 {
		return
	}
	var first any
	failed := false
	kids := n.children
	n.children = nil
	for _, c := range kids {
		c.parent = nil // the whole list is dropped; no per-child search
		if r, ok := try(c.dispose); !ok && !failed {
			first, failed = r, true
		}
	}
	clear(kids)
	if n.children == nil {
		n.children = kids[:0]
	}
	fs := n.cleanups
	n.cleanups = nil
	for _, f := range fs {
		if r, ok := try(f); !ok && !failed {
			first, failed = r, true
		}
	}
	if failed {
		panic(first)
	}
}

func try(f func()) (r any, ok bool) {
	defer func() {
		if !ok {
			r = recover()
		}
	}()
	f()
	return nil, true
}

func (n *node) dispose() {
	if n.disposed {
		return
	}
	n.disposed = true
	// A long-lived owner (a Scope) must not keep its dead children.
	if p := n.parent; p != nil {
		if i := slices.Index(p.children, n); i >= 0 {
			last := len(p.children) - 1
			p.children[i] = p.children[last]
			p.children[last] = nil
			p.children = p.children[:last]
		}
		n.parent = nil
	}
	srcs := n.sources
	n.sources = nil
	for _, s := range srcs {
		s.removeObserver(n)
	}
	n.release()
}

// Effect runs fn now, so a bound widget has its value before the first layout, then once per
// Flush after a source it read changed. What fn creates, and the OnCleanup funcs it registers,
// end before the next run and on dispose. Inside an Effect or a Scope the new effect is owned by
// it. A panic in the first run disposes the effect and reaches the caller.
func Effect(fn func()) (dispose func()) {
	mustUI("Effect")
	n := &node{kind: kindEffect}
	n.exec = func() {
		n.release()
		n.execute(fn)
		if n.disposed {
			n.release() // fn disposed its own effect: what it registered after that has no later run to wait for
		}
	}
	n.adopt()
	ok := false
	defer func() {
		if !ok {
			n.dispose()
		}
	}()
	n.exec()
	ok = true
	return n.dispose
}

// Scope runs fn untracked and owns what fn creates: dispose disposes it all, for effects that
// must die together, such as those of one window. A Scope made inside an Effect belongs to that
// run and is disposed with it.
func Scope(fn func()) (dispose func()) {
	mustUI("Scope")
	n := &node{kind: kindScope}
	n.adopt()
	prevObs, prevOwn := observer, owner
	observer, owner = nil, n
	defer func() { observer, owner = prevObs, prevOwn }()
	fn()
	return n.dispose
}

// Catch runs fn as Scope does and owns what fn creates. A panic in an Effect it owns, at any depth,
// during a Flush goes to handle instead of to the caller of Flush; the nearest Catch above the effect
// takes it. handle runs on the UI thread while the stack still shows where the panic was raised, and
// should dispose the scope or the part that failed. A panic in handle goes to the next Catch above,
// or to Flush's caller. A panic in the first run of an Effect still reaches the code that made it.
func Catch(fn func(), handle func(r any)) (dispose func()) {
	mustUI("Catch")
	n := &node{kind: kindScope, catch: handle}
	n.adopt()
	prevObs, prevOwn := observer, owner
	observer, owner = nil, n
	defer func() { observer, owner = prevObs, prevOwn }()
	fn()
	return n.dispose
}

// handOver gives r to the Catch handlers above n, nearest first, until one returns; false when
// none is left. The parent is read before a handler runs: the handler may dispose its Catch.
func (n *node) handOver(r any) (any, bool) {
	for p := n.parent; p != nil; {
		if p.catch == nil {
			p = p.parent
			continue
		}
		h, up := p.catch, p.parent
		var ok bool
		if r, ok = try(func() { h(r) }); ok {
			return nil, true
		}
		p = up
	}
	return r, false
}

// OnCleanup registers f with the running Effect or Derived, to run before its next run and on
// dispose, or with the Scope being built, to run on dispose. Elsewhere f never runs.
func OnCleanup(f func()) {
	mustUI("OnCleanup")
	if owner != nil {
		owner.cleanups = append(owner.cleanups, f)
	}
}

// Untrack calls fn without subscribing the current Derived or Effect to what fn reads.
func Untrack(fn func()) {
	mustUI("Untrack")
	prev := observer
	observer = nil
	defer func() { observer = prev }()
	fn()
}

// Batch runs fn and then flushes, for code that needs the widgets updated when it returns;
// writes from event handlers are batched per event-loop turn anyway.
func Batch(fn func()) {
	mustUI("Batch")
	batchDepth++
	ok := false
	defer func() {
		batchDepth--
		if batchDepth > 0 {
			return
		}
		if ok {
			Flush()
		} else if len(queue) > 0 && !scheduled {
			scheduled = wakeLoop() // fn panicked: do not run effects on the way out, leave them to the loop
		}
	}()
	fn()
	ok = true
}

const maxFlushRounds = 100

// Flush runs every stale effect until none is left, parents before their children. A panic in
// one effect does not stop the others: Flush finishes, then panics with the first value (later
// ones go to stderr). Effects that keep each other stale for 100 rounds are a cycle and panic too.
// Inside an Effect, Flush and Batch only schedule: the running effect would be entered again.
func Flush() {
	mustUI("Flush")
	if flushing {
		return
	}
	if inEffect > 0 {
		if len(queue) > 0 && !scheduled {
			scheduled = wakeLoop()
		}
		return
	}
	flushing = true
	scheduled = false
	defer func() { flushing = false }()
	var first any
	failed := false
	for round := 0; len(queue) > 0; round++ {
		if round == maxFlushRounds {
			n := len(queue)
			dropQueue()
			panic(fmt.Sprintf("rx: effects did not settle after %d flush rounds (%d still stale): an effect writes a State that it, or an effect it triggers, reads", maxFlushRounds, n))
		}
		q := queue
		queue = spare[:0]
		if !slices.IsSortedFunc(q, byDepth) {
			slices.SortStableFunc(q, byDepth)
		}
		for _, e := range q {
			if e.disposed {
				e.queued = false
				continue
			}
			if e.ownerStale() {
				queue = append(queue, e) // after the owner, which may dispose it; stays queued
				continue
			}
			e.queued = false
			if r, stack, ok := runEffect(e); !ok {
				if !failed {
					first, failed = r, true
				} else {
					fmt.Fprintf(os.Stderr, "rx: effect panicked after another one in the same Flush: %v\n%s", r, stack)
				}
			}
		}
		clear(q)
		spare = q[:0]
	}
	if failed {
		panic(first)
	}
}

// ownerStale reports whether an Effect above n is waiting to rerun, which would dispose n. An
// owner made stale only in this round (by an effect writing its input) is in the next round.
func (n *node) ownerStale() bool {
	for p := n.parent; p != nil; p = p.parent {
		if p.kind == kindEffect && p.state != clean {
			return true
		}
	}
	return false
}

func byDepth(a, b *node) int { return a.depth - b.depth }

func runEffect(e *node) (r any, stack []byte, ok bool) {
	defer func() {
		if ok {
			return
		}
		r, stack = recover(), debug.Stack()
		e.settle()
		// Handed over here, so that a stack taken in a handler still shows where the panic was raised.
		r, ok = e.handOver(r)
	}()
	e.update()
	return nil, nil, true
}

func dropQueue() {
	for _, e := range queue {
		e.queued = false
		e.settle()
	}
	clear(queue)
	queue = queue[:0]
}
