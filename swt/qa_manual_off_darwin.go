//go:build !gowtqa

package swt

import "github.com/haiodo/gowt/internal/cocoa"

func qaNoActivate() bool { return false }

func (this *Display) keyWindowQA() *cocoa.NSWindow { return this.application.KeyWindow() }

func QAFocus(c *Control) bool { return c.SetFocus() }
