//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

var (
	procGetGuiResources = syscall.NewLazyDLL("user32.dll").NewProc("GetGuiResources")
	procSelf            = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcess")
)

// guiCounts is the GDI and USER object count of this process.
func guiCounts() (gdi, user uintptr) {
	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	h, _, _ := procSelf.Call()
	gdi, _, _ = procGetGuiResources.Call(h, 0)
	user, _, _ = procGetGuiResources.Call(h, 1)
	return
}

func logHandles(class string, g0, u0 uintptr) {
	if os.Getenv("GOWT_HANDLE_LOG") == "" {
		return
	}
	g1, u1 := guiCounts()
	fmt.Printf("HANDLES %s gdi %d->%d (%+d) user %d->%d (%+d)\n", class, g0, g1, int(g1)-int(g0), u0, u1, int(u1)-int(u0))
}
