//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

func rowLinesOnly(*swt.Control, bool) bool { return false }
