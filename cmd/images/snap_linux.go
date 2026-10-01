package main

import (
	"fmt"
	"os"
)

// Window capture on linux is not ported yet: use the container's scrot or an X11 screenshot.
func snapshot(path string) { fmt.Fprintln(os.Stderr, "snapshot: not implemented on linux:", path) }
