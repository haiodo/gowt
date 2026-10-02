//go:build !darwin

package webview

import "github.com/haiodo/gowt/swt"

func newEngine(*WebView, *swt.Composite, Options) (engine, error) { return nil, errNotImplemented }
