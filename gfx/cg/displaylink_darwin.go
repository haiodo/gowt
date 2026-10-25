//go:build darwin

package cg

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/haiodo/gowt/gfx"
)

var (
	dlOnce       sync.Once
	dlErr        error
	dlCreate     func(out *uintptr) int32
	dlSetOut     func(link, cb, ctx uintptr) int32
	dlStart      func(link uintptr) int32
	dlStop       func(link uintptr) int32
	dlRelease    func(link uintptr)
	dlCallback   uintptr
	dlLinks      sync.Map // ctx id -> *DisplayLink
	dlNextID     atomic.Uintptr
	errNoDisplay = errors.New("gfx/cg: no display for a CVDisplayLink")
)

// DisplayLink calls fn once per display refresh, on a thread of Core Video's, with the time the
// frame will be shown. fn must be short: hand the work to the UI thread and return.
type DisplayLink struct {
	ref     uintptr
	id      uintptr
	fn      func(gfx.FrameInfo)
	seq     atomic.Uint64
	running bool
}

var (
	machAbsolute func() uint64
	machTimebase func(info *[2]uint32) int32
	tbNumer      uint64 = 1
	tbDenom      uint64 = 1
)

// hostNow is mach_absolute_time in nanoseconds: the clock of CVTimeStamp.hostTime and, in seconds,
// of NSEvent.timestamp (time since system startup).
func hostNow() time.Duration { return hostDuration(machAbsolute()) }

func hostDuration(ticks uint64) time.Duration {
	return time.Duration(ticks/tbDenom*tbNumer + ticks%tbDenom*tbNumer/tbDenom)
}

// CVTimeStamp offsets: videoTimeScale 4, videoTime 8, hostTime 16, videoRefreshPeriod 32, flags 64
// (bit 1: hostTime valid). Time is the host time, the clock of input events, else the video time.
func readStamp(p uintptr) (t, period time.Duration) {
	base := *(*unsafe.Pointer)(unsafe.Pointer(&p))
	scale := *(*int32)(unsafe.Add(base, 4))
	vt := *(*int64)(unsafe.Add(base, 8))
	host := *(*uint64)(unsafe.Add(base, 16))
	rp := *(*int64)(unsafe.Add(base, 32))
	flags := *(*uint64)(unsafe.Add(base, 64))
	if scale <= 0 {
		return 0, 0
	}
	t = time.Duration(float64(vt) / float64(scale) * float64(time.Second))
	if flags&2 != 0 && host != 0 {
		t = hostDuration(host)
	}
	return t, time.Duration(float64(rp) / float64(scale) * float64(time.Second))
}

func dlTrampoline(link, now, out, flagsIn, flagsOut, ctx uintptr) uintptr {
	if v, ok := dlLinks.Load(ctx); ok {
		d := v.(*DisplayLink)
		t, period := readStamp(out)
		d.fn(gfx.FrameInfo{Time: t, Interval: period, Seq: d.seq.Add(1)})
	}
	return 0
}

func initDL() error {
	dlOnce.Do(func() {
		const cv = "/System/Library/Frameworks/CoreVideo.framework/CoreVideo"
		lib, err := purego.Dlopen(cv, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			dlErr = err
			return
		}
		purego.RegisterLibFunc(&dlCreate, lib, "CVDisplayLinkCreateWithActiveCGDisplays")
		purego.RegisterLibFunc(&dlSetOut, lib, "CVDisplayLinkSetOutputCallback")
		purego.RegisterLibFunc(&dlStart, lib, "CVDisplayLinkStart")
		purego.RegisterLibFunc(&dlStop, lib, "CVDisplayLinkStop")
		purego.RegisterLibFunc(&dlRelease, lib, "CVDisplayLinkRelease")
		purego.RegisterLibFunc(&machAbsolute, purego.RTLD_DEFAULT, "mach_absolute_time")
		purego.RegisterLibFunc(&machTimebase, purego.RTLD_DEFAULT, "mach_timebase_info")
		var tb [2]uint32
		if machTimebase(&tb) == 0 && tb[1] != 0 {
			tbNumer, tbDenom = uint64(tb[0]), uint64(tb[1])
		}
		dlCallback = purego.NewCallback(dlTrampoline)
	})
	return dlErr
}

// NewDisplayLink makes a stopped link on the active displays.
func NewDisplayLink(fn func(gfx.FrameInfo)) (*DisplayLink, error) {
	if err := initDL(); err != nil {
		return nil, err
	}
	d := &DisplayLink{fn: fn, id: dlNextID.Add(1)}
	if rc := dlCreate(&d.ref); rc != 0 || d.ref == 0 {
		return nil, fmt.Errorf("%w (CVReturn %d)", errNoDisplay, rc)
	}
	dlLinks.Store(d.id, d)
	if rc := dlSetOut(d.ref, dlCallback, d.id); rc != 0 {
		d.Close()
		return nil, fmt.Errorf("gfx/cg: CVDisplayLinkSetOutputCallback: %d", rc)
	}
	return d, nil
}

func (d *DisplayLink) Start() error {
	if d.running {
		return nil
	}
	if rc := dlStart(d.ref); rc != 0 {
		return fmt.Errorf("gfx/cg: CVDisplayLinkStart: %d", rc)
	}
	d.running = true
	return nil
}

func (d *DisplayLink) Stop() {
	if d.running {
		dlStop(d.ref)
		d.running = false
	}
}

// Close stops the link; fn is not called after it returns.
func (d *DisplayLink) Close() {
	d.Stop()
	dlLinks.Delete(d.id)
	if d.ref != 0 {
		dlRelease(d.ref)
		d.ref = 0
	}
}
