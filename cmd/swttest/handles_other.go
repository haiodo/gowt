//go:build !windows

package main

func guiCounts() (gdi, user uintptr) { return }

func logHandles(string, uintptr, uintptr) {}
