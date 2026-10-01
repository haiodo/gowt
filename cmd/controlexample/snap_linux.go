package main

import (
	"fmt"
	"os"

	"github.com/haiodo/gowt/swt"
)

// The -snap tab driver is macOS only for now.
func snapAll(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string) {
	fmt.Fprintln(os.Stderr, "controlexample -snap: not implemented on linux")
}
