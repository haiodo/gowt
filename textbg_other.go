//go:build !linux

package gowt

import "github.com/haiodo/gowt/swt"

func textThemeBackground(*swt.Text) {}
