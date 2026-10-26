//go:build windows

package win32

import (
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// A callback grows the stack while the native holds a pointer into the caller's frame; the native then
// writes through the old address. Each test calls on a fresh goroutine, so its stack is small.

var (
	user32Test         = syscall.NewLazyDLL("user32.dll")
	procRegisterClass  = user32Test.NewProc("RegisterClassExW")
	procCreateWindow   = user32Test.NewProc("CreateWindowExW")
	procDefWindowProc  = user32Test.NewProc("DefWindowProcW")
	procSendMessageT   = user32Test.NewProc("SendMessageW")
	procPostMessageT   = user32Test.NewProc("PostMessageW")
	procDestroyWindowT = user32Test.NewProc("DestroyWindow")
	procEndMenu        = user32Test.NewProc("EndMenu")
	procKillTimer      = user32Test.NewProc("KillTimer")
)

const (
	testWMUser    = 0x400
	testWMCommand = 0x111
	testIDOK      = 1
	cfEnableHook  = 0x8
)

//go:noinline
func growStack(n int) byte {
	var pad [4096]byte
	pad[n%len(pad)] = byte(n)
	if n > 0 {
		pad[0] += growStack(n - 1)
	}
	return pad[0] + pad[n%len(pad)]
}

var growSink byte

func grow() { growSink = growStack(64) }

var classSeq atomic.Int32

// testWindow makes a window of a new class on this thread; its procedure asks onMsg first.
func testWindow(t *testing.T, parent uintptr, onMsg func(msg uint32, wp, lp uintptr) (uintptr, bool)) uintptr {
	t.Helper()
	proc := syscall.NewCallback(func(hwnd, msg, wp, lp uintptr) uintptr {
		if r, ok := onMsg(uint32(msg), wp, lp); ok {
			return r
		}
		r, _, _ := procDefWindowProc.Call(hwnd, msg, wp, lp)
		return r
	})
	name, _ := syscall.UTF16PtrFromString("gowtEscape" + string(rune('A'+classSeq.Add(1)%26)) + time.Now().Format("150405.000000"))
	wc := wndClassEx{size: uint32(unsafe.Sizeof(wndClassEx{})), proc: proc, class: uintptr(unsafe.Pointer(name))}
	if r, _, err := procRegisterClass.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		t.Fatalf("RegisterClassEx: %v", err)
	}
	hwnd, _, err := procCreateWindow.Call(0, uintptr(unsafe.Pointer(name)), 0, 0, 0, 0, 0, 0, parent, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("CreateWindowEx: %v", err)
	}
	return hwnd
}

type wndClassEx struct {
	size, style uint32
	proc        uintptr
	cls, wnd    int32
	inst, icon  uintptr
	cursor, bg  uintptr
	menu, class uintptr
	small       uintptr
}

const hwndMessage = ^uintptr(2)

// retrieveRound: another thread sends to this thread's window while retrieve is inside GetMessage or
// PeekMessage; the posted message queued before it must come back intact.
func retrieveRound(t *testing.T, postFirst bool, retrieve func(hwnd int64) (MSG, bool)) {
	t.Helper()
	ready := make(chan uintptr, 1)
	done := make(chan struct{})
	var got MSG
	var ok bool
	var sent atomic.Bool
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		hwnd := testWindow(t, hwndMessage, func(msg uint32, wp, lp uintptr) (uintptr, bool) {
			if msg == testWMUser {
				sent.Store(true)
				grow()
				return 7, true
			}
			return 0, false
		})
		defer procDestroyWindowT.Call(hwnd)
		ready <- hwnd
		got, ok = retrieve(int64(hwnd))
	}()
	hwnd := <-ready
	// PeekMessage polls: the post is queued first, so the call that dispatches the send returns it too.
	// GetMessage blocks: the send comes first and the post wakes it.
	if postFirst {
		procPostMessageT.Call(hwnd, testWMUser+1, 11, 22)
	}
	time.Sleep(50 * time.Millisecond)
	if r, _, _ := procSendMessageT.Call(hwnd, testWMUser, 0, 0); r != 7 {
		t.Errorf("SendMessage = %d, want 7 from the window procedure", r)
	}
	if !postFirst {
		procPostMessageT.Call(hwnd, testWMUser+1, 11, 22)
	}
	<-done
	if !sent.Load() {
		t.Fatal("the window procedure was not called")
	}
	if !ok || got.Message != testWMUser+1 || got.WParam != 11 || got.LParam != 22 || got.Hwnd != int64(hwnd) {
		t.Errorf("message = %+v ok=%v, want WM_USER+1 (11, 22) for %#x", got, ok, hwnd)
	}
}

func TestGetMessageSurvivesStackGrowth(t *testing.T) {
	for range 5 {
		retrieveRound(t, false, func(hwnd int64) (MSG, bool) {
			var msg MSG
			r := OSGetMessage(&msg, hwnd, 0, 0)
			return msg, r
		})
	}
}

func TestPeekMessageSurvivesStackGrowth(t *testing.T) {
	for range 5 {
		retrieveRound(t, true, func(hwnd int64) (MSG, bool) {
			var msg MSG
			time.Sleep(100 * time.Millisecond)
			for range 100 {
				if OSPeekMessage(&msg, hwnd, 0, 0, OSPM_REMOVE) {
					return msg, true
				}
				time.Sleep(10 * time.Millisecond)
			}
			return msg, false
		})
	}
}

func needGUI(t *testing.T) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("opens a window: set GOWT_GUI_TEST=1")
	}
}

// dialogHook is the hook of a common dialog: it grows the stack, runs setup with the struct pointer
// the dialog holds, and presses OK.
func dialogHook(setup func(lp int64)) int64 {
	return NewCallbackN(4, func(a []int64) int64 {
		if int32(a[1]) == OSWM_INITDIALOG {
			grow()
			if setup != nil {
				setup(a[3])
			}
			procPostMessageT.Call(uintptr(a[0]), testWMCommand, testIDOK, 0)
		}
		return 0
	})
}

func onFreshThread(f func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		f()
	}()
	<-done
}

func TestChooseColorSurvivesStackGrowth(t *testing.T) {
	needGUI(t)
	const want = 0x123456
	for range 3 {
		onFreshThread(func() {
			custom := OSHeapAlloc(OSGetProcessHeap(), OSHEAP_ZERO_MEMORY, 64)
			defer OSHeapFree(OSGetProcessHeap(), 0, custom)
			// lCustData is at byte 48: what the hook stores there through the dialog's pointer must be in cc.
			cc := &CHOOSECOLOR{LStructSize: 72, Flags: OSCC_ENABLEHOOK | OSCC_RGBINIT, LpCustColors: custom,
				LpfnHook: dialogHook(func(lp int64) { *(*int64)(unsafe.Add(unsafe.Pointer(nil), uintptr(lp)+48)) = want })}
			if !OSChooseColor(cc) {
				t.Errorf("ChooseColor = false")
				return
			}
			if cc.LCustData != want {
				t.Errorf("LCustData = %#x, want %#x: the hook wrote through a stale address", cc.LCustData, want)
			}
		})
	}
}

func TestChooseFontSurvivesStackGrowth(t *testing.T) {
	needGUI(t)
	for range 3 {
		onFreshThread(func() {
			heap := OSGetProcessHeap()
			lf := OSHeapAlloc(heap, OSHEAP_ZERO_MEMORY, 92)
			defer OSHeapFree(heap, 0, lf)
			// LOGFONT as words: lfHeight -16 at 0, lfWeight 400 at byte 16, lfFaceName at byte 28.
			buf := make([]uint16, 46)
			buf[0], buf[1], buf[8] = 0xFFF0, 0xFFFF, 400
			copy(buf[14:], syscall.StringToUTF16("Segoe UI"))
			OSMoveMemoryOverload12(lf, buf, 92)
			cf := &CHOOSEFONT{LStructSize: 104, LpLogFont: lf, Flags: cfEnableHook | OSCF_INITTOLOGFONTSTRUCT | OSCF_SCREENFONTS,
				LpfnHook: dialogHook(nil)}
			if !OSChooseFont(cf) {
				t.Errorf("ChooseFont = false")
				return
			}
			if cf.IPointSize == 0 {
				t.Errorf("IPointSize = 0: the dialog wrote through a stale address")
			}
		})
	}
}

// The timer procedure runs inside the menu's modal loop, grows the stack and closes the menu.
func TestTrackPopupMenuSurvivesStackGrowth(t *testing.T) {
	needGUI(t)
	onFreshThread(func() {
		hwnd := testWindow(t, 0, func(uint32, uintptr, uintptr) (uintptr, bool) { return 0, false })
		defer procDestroyWindowT.Call(hwnd)
		menu := OSCreatePopupMenu()
		defer OSDestroyMenu(menu)
		item := syscall.StringToUTF16("item")
		user32Test.NewProc("AppendMenuW").Call(uintptr(menu), 0, 1, uintptr(unsafe.Pointer(&item[0])))
		var fired atomic.Bool
		timer := syscall.NewCallback(func(_, _, id, _ uintptr) uintptr {
			fired.Store(true)
			grow()
			procEndMenu.Call()
			return 0
		})
		id := OSSetTimer(0, 0, 200, int64(timer))
		defer procKillTimer.Call(0, uintptr(id))
		rc := RECT{Left: 0, Top: 0, Right: 100, Bottom: 100}
		OSTrackPopupMenu(menu, 0x100, 50, 50, 0, int64(hwnd), &rc)
		if !fired.Load() {
			t.Error("the timer procedure did not run inside the menu loop")
		}
		if rc != (RECT{Left: 0, Top: 0, Right: 100, Bottom: 100}) {
			t.Errorf("rect = %+v changed", rc)
		}
	})
}
