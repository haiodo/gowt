//go:build !gowt_nomanifest

package gowt

// Facade apps get visual styles and per-monitor DPI without their own manifest; apps with one use swt directly
// or build with -tags gowt_nomanifest and put the manifest into their own resource object (docs/packaging.md).
import _ "github.com/haiodo/gowt/winmanifest"
