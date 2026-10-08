package gowt

import "github.com/haiodo/gowt/internal/cocoa"

func (w *Window) setBackdrop(b Backdrop) {
	if had := cocoa.InstallBackdrop(w.shell.View, int(b), true); b != BackdropNone || had {
		w.clear(b != BackdropNone)
	}
}
