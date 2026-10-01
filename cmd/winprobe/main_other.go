//go:build !windows

package main

import "fmt"

func main() { fmt.Println("winprobe checks the Windows bindings: GOOS=windows go build ./cmd/winprobe") }
