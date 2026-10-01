package main

import (
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// Window capture is per platform; the snapshot gate is not ported to win32.
func snapAll(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string) {
	fmt.Println("snapAll: not supported on windows")
}
