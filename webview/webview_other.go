//go:build !darwin && !linux && !windows

package webview

import "github.com/haiodo/gowt/swt"

func newEngine(*WebView, *swt.Composite, Options) (engine, error) { return nil, errNotImplemented }

func cookies(url string, done func([]Cookie, bool)) { done(nil, false) }
func setCookie(url, header string, done func(bool)) { done(false) }
func clearSessionCookies(done func())               { done() }
