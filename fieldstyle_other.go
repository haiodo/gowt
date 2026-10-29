//go:build !darwin && !windows && !linux

package gowt

import "github.com/haiodo/gowt/swt"

func setFieldFill(*swt.Text, int)                          {}
func setFieldStroke(*swt.Text, uint8, uint8, uint8, uint8) {}
func setFieldInset(*swt.Text, int)                         {}
