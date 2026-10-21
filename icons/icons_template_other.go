//go:build !darwin

package icons

import "github.com/haiodo/gowt/swt"

// Only AppKit has template images.
func markTemplate(*swt.Image) {}
