package gowt

import "github.com/haiodo/gowt/internal/cocoa"

func (w *Window) setBackdrop(b Backdrop) { cocoa.InstallBackdrop(w.shell.View, int(b), true) }
