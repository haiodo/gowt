//go:build !linux

package gowt

import "github.com/haiodo/gowt/swt"

func textScrollOnOverflow(*swt.Text) {}
