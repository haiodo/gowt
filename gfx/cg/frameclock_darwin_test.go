//go:build darwin

package cg

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

type fakeLink struct {
	fn     func(gfx.FrameInfo)
	closed atomic.Bool
	stop   chan struct{}
	wg     sync.WaitGroup
	ticks  atomic.Int64
}

func newFakeLink(fn func(gfx.FrameInfo)) *fakeLink {
	return &fakeLink{fn: fn, stop: make(chan struct{})}
}

func (l *fakeLink) Start() error {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for {
			select {
			case <-l.stop:
				return
			case <-time.After(2 * time.Millisecond):
				l.ticks.Add(1)
				l.fn(gfx.FrameInfo{})
			}
		}
	}()
	return nil
}

func (l *fakeLink) Close() {
	if l.closed.CompareAndSwap(false, true) {
		close(l.stop)
	}
	l.wg.Wait() // like CVDisplayLinkStop: no callback runs after it returns
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		if ok() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("timeout: " + what)
}

// A link that cannot be made (displays asleep) must not stop frames;
// the link takes over once it works.
func TestFrameClockFallbackAndSwitch(t *testing.T) {
	var ticks, attempts atomic.Int64
	var made atomic.Pointer[fakeLink]
	c := NewFrameClock(func(gfx.FrameInfo) { ticks.Add(1) })
	c.period, c.retry = 2*time.Millisecond, 20*time.Millisecond
	c.newLink = func(fn func(gfx.FrameInfo)) (tickLink, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("CVReturn -6661")
		}
		l := newFakeLink(fn)
		made.Store(l)
		return l, nil
	}
	c.Start()
	waitFor(t, "timer ticks without a link", func() bool { return ticks.Load() >= 5 })
	waitFor(t, "link made", func() bool { return made.Load() != nil })
	// the clock settles on a link that has ticked and is still open
	var l *fakeLink
	waitFor(t, "link took over", func() bool {
		l = made.Load()
		return l.ticks.Load() > 0 && !l.closed.Load()
	})
	c.Close()
	if !made.Load().closed.Load() {
		t.Error("link not closed")
	}
	a := ticks.Load()
	time.Sleep(20 * time.Millisecond)
	if ticks.Load() != a {
		t.Error("ticks after Close")
	}
}

// A link that goes silent later (sleep) is replaced by the timer.
func TestFrameClockSilentLink(t *testing.T) {
	var ticks atomic.Int64
	var first atomic.Pointer[fakeLink]
	c := NewFrameClock(func(gfx.FrameInfo) { ticks.Add(1) })
	c.period, c.retry = 2*time.Millisecond, 10*time.Millisecond
	c.newLink = func(fn func(gfx.FrameInfo)) (tickLink, error) {
		if first.Load() != nil {
			return nil, errors.New("asleep")
		}
		l := newFakeLink(fn)
		first.Store(l)
		return l, nil
	}
	c.Start()
	defer c.Close()
	waitFor(t, "first link ticks", func() bool { return ticks.Load() >= 3 })
	first.Load().Close()
	n := ticks.Load()
	waitFor(t, "timer after silence", func() bool { return ticks.Load() > n+10 })
}

// Close waits for a tick that is inside fn, so fn never runs after Close returns.
func TestFrameClockCloseWaitsForTick(t *testing.T) {
	for _, viaLink := range []bool{false, true} {
		var calls atomic.Int64
		entered, release := make(chan struct{}, 1), make(chan struct{})
		c := NewFrameClock(func(gfx.FrameInfo) {
			calls.Add(1)
			select {
			case entered <- struct{}{}:
				<-release
			default:
			}
		})
		c.period, c.retry = 2*time.Millisecond, 20*time.Millisecond
		c.newLink = func(fn func(gfx.FrameInfo)) (tickLink, error) {
			if !viaLink {
				return nil, errors.New("asleep")
			}
			return newFakeLink(fn), nil
		}
		c.Start()
		<-entered
		closed := make(chan struct{})
		go func() { c.Close(); close(closed) }()
		select {
		case <-closed:
			t.Fatalf("viaLink=%v: Close returned while fn was running", viaLink)
		case <-time.After(20 * time.Millisecond):
		}
		close(release)
		<-closed
		n := calls.Load()
		time.Sleep(20 * time.Millisecond)
		if calls.Load() != n {
			t.Errorf("viaLink=%v: fn ran after Close", viaLink)
		}
	}
}

// Subscribers share one clock: three subscribers, at most one display link, the clock keeps
// ticking while any subscriber stays, and it stops with the last one.
func TestSharedClockOneLinkManySubscribers(t *testing.T) {
	var a, b, c, d atomic.Int64
	links := dlMade.Load()
	stopA := SubscribeFrames(func(gfx.FrameInfo) { a.Add(1) })
	stopB := SubscribeFrames(func(gfx.FrameInfo) { b.Add(1) })
	stopC := SubscribeFrames(func(gfx.FrameInfo) { c.Add(1) })
	waitFor(t, "every subscriber ticks", func() bool { return a.Load() > 0 && b.Load() > 0 && c.Load() > 0 })
	if n := dlMade.Load() - links; n > 1 {
		t.Errorf("%d display links made for 3 subscribers, want at most 1", n)
	}
	stopA()
	waitFor(t, "the remaining subscribers keep ticking", func() bool { return b.Load() > 5 && c.Load() > 5 })
	stopB()
	stopC()
	stopD := SubscribeFrames(func(gfx.FrameInfo) { d.Add(1) })
	waitFor(t, "the clock restarts for a new subscriber", func() bool { return d.Load() > 0 })
	stopD()
}

// Stopping the last subscriber and resubscribing right away must start a fresh clock: the old
// clock's late tick carries its own high Seq, so a first tick of Seq near 1 proves the new
// subscriber is called by the new clock only.
func TestSharedClockResubscribeStartsFreshClock(t *testing.T) {
	var a, b, bSeq atomic.Int64
	stopA := SubscribeFrames(func(gfx.FrameInfo) { a.Add(1) })
	waitFor(t, "the first clock ticking", func() bool { return a.Load() > 40 })
	stopA()
	stopB := SubscribeFrames(func(fi gfx.FrameInfo) { bSeq.Store(int64(fi.Seq)); b.Add(1) })
	waitFor(t, "the new subscriber ticking", func() bool { return b.Load() > 0 })
	if s := bSeq.Load(); s > 10 {
		t.Errorf("the new subscriber's first tick has Seq %d: the old clock called it", s)
	}
	stopB()
}
