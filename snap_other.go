//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

func shotHandle(s *swt.Shell) int64 { return s.Handle }
