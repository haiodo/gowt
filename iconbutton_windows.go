package gowt

import "github.com/haiodo/gowt/swt"

// A Win32 push button always draws its frame and Accessible is a stub: the three stay as they are.
func setButtonBorderless(*swt.Button) {}

func setButtonCircle(*swt.Button) {}

func setAccessibleLabel(*swt.Control, string) {}
