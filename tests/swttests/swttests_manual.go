// Package swttests is SWT's JUnit tests (org.eclipse.swt.tests.junit) translated by j2go; each
// test class registers itself with internal/junit, cmd/swttest runs them on the main thread.
package swttests

import (
	"embed"
	"io/fs"

	"github.com/haiodo/gowt/internal/jrt"
)

// The test images next to the Java sources, copied by tooling/port.sh.
//
//go:embed testdata
var testdata embed.FS

// A var initializer: runs before the generated init() functions, which may already load images.
var _ = func() bool {
	sub, _ := fs.Sub(testdata, "testdata")
	jrt.RegisterResources(sub)
	return true
}()
