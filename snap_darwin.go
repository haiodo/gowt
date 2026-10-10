package gowt

import "github.com/haiodo/gowt/swt"

// shotHandle is what internal/shot needs to find the window: the shell's NSView.
func shotHandle(s *swt.Shell) int64 { return s.View.Id }
