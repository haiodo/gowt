//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

func topOnFirstSize(*swt.Control, func()) {}
