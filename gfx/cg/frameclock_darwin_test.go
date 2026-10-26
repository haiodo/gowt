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
