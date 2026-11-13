//go:build !darwin

package main

import "github.com/haiodo/gowt"

// contentTopInset is the macOS full-size-content title bar strip; elsewhere the sidebar
// has no such strip.
func contentTopInset(*gowt.Window) int { return 0 }
