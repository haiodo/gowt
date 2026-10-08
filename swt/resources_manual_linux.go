package swt

import (
	"embed"
	"io/fs"
	"path"
	"strings"

	"github.com/haiodo/gowt/internal/jrt"
)

// Device.overrideThemeValues loads /org/eclipse/swt/internal/gtk/*.css by resource name; gtkres/ holds our own
// stand-ins: swt_functional_gtk_3_20.css carries the accent rules, the others are empty (stock theme).
//
//go:embed gtkres/*.css
var gtkResources embed.FS

type gtkResourceFS struct{}

const gtkResourceDir = "org/eclipse/swt/internal/gtk/"

func (gtkResourceFS) Open(name string) (fs.File, error) {
	if !strings.HasPrefix(name, gtkResourceDir) {
		return nil, fs.ErrNotExist
	}
	return gtkResources.Open("gtkres/" + path.Base(name))
}

func init() { jrt.RegisterResources(gtkResourceFS{}) }
