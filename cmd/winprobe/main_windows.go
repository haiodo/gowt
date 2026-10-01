//go:build windows

// Command winprobe is a console check: every DLL proc the Win32 bindings call must resolve. Opens no window.
package main

import (
	"fmt"
	"os"

	"github.com/haiodo/gowt/internal/win32"
	_ "github.com/haiodo/gowt/swt" // its package init (static initialisers) runs too: any failure prints "deferred init"
)

func main() {
	total, missing, optional := win32.Probe()
	fmt.Printf("procs: %d, missing: %d, optional absent: %d\n", total, len(missing), len(optional))
	for _, m := range missing {
		fmt.Println("MISSING", m)
	}
	for _, m := range optional {
		fmt.Println("optional", m)
	}
	if len(missing) > 0 {
		os.Exit(1)
	}
}
