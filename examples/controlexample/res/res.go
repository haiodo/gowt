// Package res embeds the ControlExample resources. It is a separate package so that its registration
// runs before any package-level initializer of controlexample that reads a resource bundle.
package res

import (
	"embed"

	"github.com/haiodo/gowt/internal/jrt"
)

//go:embed *.png *.gif *.bmp examples_control.properties
var resources embed.FS

func init() { jrt.RegisterResources(resources) }
