//go:build !linux

package gowt

import "github.com/haiodo/gowt/swt"

// repaintShell is needed on GTK only, see repaint_linux.go.
func (a *App) repaintShell(*swt.Control) {}
