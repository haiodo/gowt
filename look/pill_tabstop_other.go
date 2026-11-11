//go:build !windows

package look

import "github.com/haiodo/gowt/swt"

// pillTabStop does nothing off Windows: a Control is already a tab group there through its default
// traversal code, so the canvas joins the shell's Tab order on its own.
func pillTabStop(*swt.Canvas) {}
