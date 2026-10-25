package main

import (
	"testing"

	"github.com/ebitengine/purego"
)

// TestAccent reads the accent colour without a window: AppKit only has to be loaded.
func TestAccent(t *testing.T) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		t.Skip(err)
	}
	r, g, b, ok := systemAccent()
	if !ok {
		t.Fatal("no accent colour")
	}
	t.Logf("accent %d %d %d", r, g, b)
	if r == 0 && g == 0 && b == 0 {
		t.Error("accent is black")
	}
}
