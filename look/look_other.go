//go:build !darwin && !windows

package look

import "github.com/haiodo/gowt/swt"

func setBackdrop(*swt.Shell, Backdrop) {}

func setRoundedCorners(*swt.Shell, bool) {}

func classic() {}

func setFullSizeContent(*swt.Shell, bool) {}

func setGlass(*swt.Composite, bool) {}

func glassButton(*swt.Control) {}

func setDarkContent(*swt.Display, bool) {}
