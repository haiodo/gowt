//go:build !darwin

package main

import "github.com/haiodo/gowt"

// titleInset is the macOS title bar strip; elsewhere the sidebar has none.
func titleInset(*gowt.Window) int { return 0 }
