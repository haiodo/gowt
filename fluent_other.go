//go:build !windows

package gowt

import "github.com/haiodo/gowt/swt"

const multilineScroll = swt.V_SCROLL

func fluentText(*TextView, bool) {}

func fluentBorder(*swt.Control) {}

func fluentCard(*swt.Display) *swt.Color { return nil }

func fluentScroll(*swt.Composite) {}

func fluentTable(*swt.Table) {}

func fluentList(*swt.List) {}

func fluentTree(*swt.Tree) {}

func fluentStyledText(*swt.StyledText) {}

func fluentShell(*swt.Shell) {}

func controlHeight(opts []Option) []Option { return opts }

func fluentRange(opts []Option) []Option { return opts }
