//go:build windows

package d2d

import (
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procCreateWindow = user32.NewProc("CreateWindowExW")
	procDestroyWin   = user32.NewProc("DestroyWindow")
)

// testWindow makes a window that is never shown: a swap chain needs an HWND, not a desktop.
func testWindow(t *testing.T, w, h int) uintptr {
	t.Helper()
	hwnd, _, err := procCreateWindow.Call(0, uintptr(unsafe.Pointer(utf16Z("STATIC"))), 0, 0x80000000, 0, 0, uintptr(w), uintptr(h), 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("CreateWindowEx: %v", err)
	}
	t.Cleanup(func() { procDestroyWin.Call(hwnd) })
	return hwnd
}

func TestHostPresents(t *testing.T) {
	hwnd := testWindow(t, 320, 200)
	wpx, hpx, err := ClientSize(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHost(shared, hwnd, wpx, hpx, 1)
	if err != nil {
		if strings.Contains(err.Error(), "0x887a0022") {
			t.Skip("no desktop session: a swap chain needs one (run through vm.sh gui)")
		}
		t.Fatal(err)
	}
	defer h.Close()
	tg := shared.NewTarget(64, 64)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, gfx.Color{})
	tg.Draw([]gfx.Item{{Pic: record(func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 64, 64), red) }), M: gfx.Identity(), Alpha: 1}})
	img := tg.Snapshot(64, 64)
	l := h.NewLayer(gfx.GroupTiles)
	l.SetImage(img)
	gfx.Release(img)
	l.Set(gfx.R(10, 10, 64, 64), gfx.Point{}, gfx.Identity())
	h.Begin()
	h.Commit()
	t.Logf("draw %v present %v", h.LastDraw, h.LastPresent)
	if err := h.Resize(400, 300); err != nil {
		t.Fatal(err)
	}
	h.Begin()
	h.Commit()
	l.Remove()
	h.Commit()
}

func TestFrameClock(t *testing.T) {
	n := 0
	done := make(chan struct{})
	c, err := NewFrameClock(func(fi gfx.FrameInfo) {
		n++
		if n == 20 {
			close(done)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(); err != nil {
		t.Skipf("no DWM composition here: %v", err)
	}
	<-done
	c.Close()
	t.Logf("20 ticks")
}
