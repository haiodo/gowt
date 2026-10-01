//go:build windows

// Command winprobe is a console check: every DLL proc the Win32 bindings call must resolve. Opens no window.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt" // package init (static initialisers) runs too: any failure prints "deferred init"
)

func init() { runtime.LockOSThread() }

func main() {
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
