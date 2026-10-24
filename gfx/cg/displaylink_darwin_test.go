//go:build darwin

package cg

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/cocoa"
)

// NSEvent.timestamp is on the clock of the frame time; NSProcessInfo.systemUptime is that clock.
func TestHostClockIsSystemUptime(t *testing.T) {
	initInput()
	if err := initDL(); err != nil {
		t.Skip(err)
	}
	pi := send0(uintptr(cocoa.OSObjc_lookUpClass("NSProcessInfo")), uintptr(cocoa.OSSel_registerName("processInfo")))
	up := time.Duration(msgF64Ret(pi, uintptr(cocoa.OSSel_registerName("systemUptime"))) * float64(time.Second))
	if d := up - hostNow(); d < -50*time.Millisecond || d > 50*time.Millisecond {
		t.Fatalf("systemUptime %v, mach time %v", up, hostNow())
	}
}

// A display link needs a display but no window.
func TestDisplayLinkTicks(t *testing.T) {
	var n atomic.Int64
	var last atomic.Int64
	var period atomic.Int64
	d, err := NewDisplayLink(func(fi gfx.FrameInfo) {
		n.Add(1)
		last.Store(int64(fi.Time - hostNow()))
		period.Store(int64(fi.Interval))
	})
	if err != nil {
		t.Skipf("no display link: %v", err)
	}
	if err := d.Start(); err != nil {
		t.Skip(err)
	}
	time.Sleep(400 * time.Millisecond)
	d.Close()
	got := n.Load()
	if got < 10 {
		t.Fatalf("%d ticks in 400 ms", got)
	}
	if d := time.Duration(last.Load()); d < -100*time.Millisecond || d > 100*time.Millisecond || period.Load() < int64(2*time.Millisecond) || period.Load() > int64(40*time.Millisecond) {
		t.Errorf("frame time %v, refresh period %v", time.Duration(last.Load()), time.Duration(period.Load()))
	}
	after := n.Load()
	time.Sleep(50 * time.Millisecond)
	if n.Load() != after {
		t.Error("ticks after Close")
	}
	t.Logf("%d ticks, refresh period %v", got, time.Duration(period.Load()))
}
