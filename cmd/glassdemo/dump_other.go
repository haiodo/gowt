//go:build !darwin

package main

import "github.com/haiodo/gowt"

func dumpViews(*gowt.Window) string { return "view dump is macOS only\n" }
