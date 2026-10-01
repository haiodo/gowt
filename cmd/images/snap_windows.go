package main

import "fmt"

// Window capture is per platform: main calls snapshot, each OS provides it. Not ported on win32.
func snapshot(path string) { fmt.Println("snapshot: not supported on windows") }
