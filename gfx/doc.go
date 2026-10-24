// Package gfx is an experimental retained 2D scene for gowt: vector paths with curve
// geometry, recorded Pictures, and a grid-indexed Scene with hit-testing.
//
// The package is pure Go and builds on every OS. Rasterizers live in sub-packages
// (gfx/cg: CoreGraphics + CoreText on macOS). The API is a spike for the G-0 budget
// measurements (TSK-2026-10-24-gfx-tenniarb-01, -15) and may change without notice.
package gfx
