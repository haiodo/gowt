//go:build darwin

package cg

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

type fakeLink struct {
	fn     func(gfx.FrameInfo)
	closed atomic.Bool
	stop   chan struct{}
}

func newFakeLink(fn func(gfx.FrameInfo)) *fakeLink {
	return &fakeLink{fn: fn, stop: make(chan struct{})}
}

func (l *fakeLink) Start() error {
	go func() {
		for {
			select {
			case <-l.stop:
				return
			case <-time.After(2 * time.Millisecond):
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
	l := made.Load()
	waitFor(t, "link only, no more attempts", func() bool {
		a := ticks.Load()
		time.Sleep(10 * time.Millisecond)
		return ticks.Load() > a && attempts.Load() == 3
	})
	c.Close()
	if !l.closed.Load() {
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
