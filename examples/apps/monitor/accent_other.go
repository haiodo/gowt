//go:build !darwin && !windows && !linux

package main

func systemAccent() (r, g, b uint8, ok bool) { return 0, 0, 0, false }
