//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

func systemDark() bool { return swt.DisplayIsSystemDarkTheme() }
