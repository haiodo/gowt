//go:build !windows

package gowt

import "github.com/haiodo/gowt/swt"

const multilineScroll = swt.V_SCROLL

func fluentText(*TextView, bool) {}

func fluentBorder(*swt.Control) {}

func fluentTable(*swt.Table) {}

func fluentShell(*swt.Shell) {}
