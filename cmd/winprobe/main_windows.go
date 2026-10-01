//go:build windows

// Command winprobe is a console check: every DLL proc the Win32 bindings call must resolve. Opens no window.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt" // package init (static initialisers) runs too: any failure prints "deferred init"
)

func init() { runtime.LockOSThread() }

func main() {
	link := flag.Bool("link", false, "InitCommonControlsEx + GetClassInfoEx of SysLink and a few other classes (no window)")
	display := flag.Bool("display", false, "also create a Display (hidden message window only, no Shell) and print fonts, DPI, monitors")
	flag.Parse()
	total, missing, optional := win32.Probe()
	fmt.Printf("procs: %d, missing: %d, optional absent: %d\n", total, len(missing), len(optional))
	for _, m := range missing {
		fmt.Println("MISSING", m)
	}
	for _, m := range optional {
		fmt.Println("optional", m)
	}
	if *link {
		icex := [2]uint32{8, 0xffff} // all classes
		r, _, e := syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&icex)))
		fmt.Println("InitCommonControlsEx:", r, e)
		for _, c := range []string{"SysLink", "SysListView32", "SysTreeView32", "msctls_progress32", "ComboBox"} {
			var wc [80]byte
			*(*uint32)(unsafe.Pointer(&wc)) = 80
			n, _ := syscall.UTF16PtrFromString(c)
			r, _, e := syscall.NewLazyDLL("user32.dll").NewProc("GetClassInfoExW").Call(0, uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(&wc)))
			fmt.Println("class", c, r, e)
		}
		hm, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("comctl32.dll"))))
		var buf [260]uint16
		syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleFileNameW").Call(hm, uintptr(unsafe.Pointer(&buf[0])), 260)
		fmt.Println("comctl32 module:", syscall.UTF16ToString(buf[:]))
		vi := [5]uint32{20}
		v, _, _ := syscall.NewLazyDLL("comctl32.dll").NewProc("DllGetVersion").Call(uintptr(unsafe.Pointer(&vi)))
		fmt.Println("comctl32 version", vi[1], vi[2], "hr", v)
	}
	if *display {
		d := swt.NewDisplay()
		f := d.GetSystemFont()
		fmt.Println("system font:", f.GetFontData()[0].GetName(), f.GetFontData()[0].GetHeight())
		fmt.Println("dpi:", d.GetDPI().X, d.GetDPI().Y)
		hm := win32.OSMonitorFromWindow(0, win32.OSMONITOR_DEFAULTTOPRIMARY)
		mi := win32.NewMONITORINFO()
		mi.CbSize = win32.MONITORINFOSizeof
		ok := win32.OSGetMonitorInfo(hm, mi)
		fmt.Println("primary", hm, ok, mi.CbSize, mi.RcMonitor_right, mi.RcMonitor_bottom)
		for i, m := range d.GetMonitors() {
			b := m.GetBounds()
			fmt.Println("monitor", i, b.X, b.Y, b.Width, b.Height)
		}
		d.Dispose()
	}
	if len(missing) > 0 {
		os.Exit(1)
	}
}
