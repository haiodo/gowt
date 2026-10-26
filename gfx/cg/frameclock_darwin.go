//go:build darwin

package cg

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/haiodo/gowt/gfx"
)

type tickLink interface {
	Start() error
	Close()
}

// FrameClock is a DisplayLink that does not die with the display: while no link can be made or
// started (displays asleep: CVReturn -6661) or a running link goes silent, a 60 Hz timer calls fn
// instead, and the link is tried again every second. fn has the DisplayLink contract: short, any
// thread, decides itself whether a frame is needed.
type FrameClock struct {
	fn      func(gfx.FrameInfo)
	newLink func(func(gfx.FrameInfo)) (tickLink, error)
	period  time.Duration // timer mode tick
	retry   time.Duration // link retry and silence check
	seq     atomic.Uint64
	lastDL  atomic.Int64 // hostNow of the last link tick, 0 before the first
	done    chan struct{}
	wg      sync.WaitGroup
}

var clockStart = time.Now()

// clockNow is the frame time clock; the mach clock when Core Video loaded, else a monotonic stand-in.
func clockNow() time.Duration {
	if machAbsolute != nil {
		return hostNow()
	}
	return time.Since(clockStart)
}

func NewFrameClock(fn func(gfx.FrameInfo)) *FrameClock {
	_ = initDL() // hostNow needs its symbols; a failure leaves clockNow on the stand-in
	return &FrameClock{
		fn: fn,
		newLink: func(f func(gfx.FrameInfo)) (tickLink, error) {
			d, err := NewDisplayLink(f)
			if err != nil {
				return nil, err
			}
			return d, nil
		},
		period: time.Second / 60,
		retry:  time.Second,
		done:   make(chan struct{}),
	}
}

// Start never fails: without a working link the timer runs.
func (c *FrameClock) Start() {
	c.wg.Add(1)
	go c.run()
}

func (c *FrameClock) Close() {
	close(c.done)
	c.wg.Wait()
}

func (c *FrameClock) emit(fi gfx.FrameInfo) {
	fi.Seq = c.seq.Add(1)
	c.fn(fi)
}

func (c *FrameClock) tryLink() tickLink {
	c.lastDL.Store(0)
	l, err := c.newLink(func(fi gfx.FrameInfo) {
		c.lastDL.Store(int64(clockNow()) | 1)
		c.emit(fi)
	})
	if err != nil {
		return nil
	}
	if l.Start() != nil {
		l.Close()
		return nil
	}
	return l
}

func (c *FrameClock) run() {
	defer c.wg.Done()
	link := c.tryLink()
	for {
		if link != nil {
			if c.watchLink(link) {
				return
			}
			link = nil
		}
		var closed bool
		if link, closed = c.timerMode(); closed {
			return
		}
	}
}

// watchLink closes l and reports true on Close, or closes it and reports false when it went silent.
func (c *FrameClock) watchLink(l tickLink) bool {
	t := time.NewTicker(c.retry)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			l.Close()
			return true
		case <-t.C:
			if last := c.lastDL.Load(); last != 0 && clockNow()-time.Duration(last) > 2*c.retry {
				l.Close()
				return false
			}
		}
	}
}

// timerMode ticks until a link that has produced a tick replaces it.
func (c *FrameClock) timerMode() (tickLink, bool) {
	t := time.NewTicker(c.period)
	defer t.Stop()
	every := int(c.retry / c.period)
	if every < 1 {
		every = 1
	}
	var cand tickLink
	for n := 1; ; n++ {
		select {
		case <-c.done:
			if cand != nil {
				cand.Close()
			}
			return nil, true
		case <-t.C:
			if cand != nil && c.lastDL.Load() != 0 {
				return cand, false
			}
			c.emit(gfx.FrameInfo{Time: clockNow(), Interval: c.period})
			if n%every == 0 {
				if cand != nil {
					cand.Close()
				}
				cand = c.tryLink()
			}
		}
	}
}
