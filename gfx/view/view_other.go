//go:build !darwin && !linux

// Package view puts a gfx scene into a gowt window as a widget.
//
// Only macOS has a backend so far; elsewhere New makes a plain canvas and never calls init, and
// a View cannot be obtained.
package view

import (
	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// View is not available on this OS.
type View struct{}

// New makes a plain canvas; init is never called.
func New(scene *gfx.Scene, init func(*View), opts ...g.Option) g.Option {
	return g.Canvas(nil, opts...)
}
