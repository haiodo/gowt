//go:build jface

package main

// `go build -tags jface ./cmd/swttest` runs JFace's translated tests instead of SWT's (make test-jface).
import _ "github.com/haiodo/gowt/tests/jfacetests"

const pkg = "github.com/haiodo/gowt/tests/jfacetests"
