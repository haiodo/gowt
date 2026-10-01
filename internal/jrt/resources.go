package jrt

import (
	"bytes"
	"io/fs"
	"strings"
)

// resourceFSs back Class.getResourceAsStream: each package registers its embed.FS once via
// RegisterResources, and GetResourceAsStream looks names up in all of them.
var resourceFSs []fs.FS

func RegisterResources(f fs.FS) { resourceFSs = append(resourceFSs, f) }

// GetResourceAsStream is Class.getResourceAsStream(name): nil if no FS was registered or name
// isn't in it, matching Java's own "resource not found" return.
func GetResourceAsStream(name string) InputStream {
	if data, ok := readResource(name); ok {
		return NewInputStream(bytes.NewReader(data))
	}
	return nil
}

func readResource(name string) ([]byte, bool) {
	for _, f := range resourceFSs {
		if data, err := fs.ReadFile(f, strings.TrimPrefix(name, "/")); err == nil {
			return data, true
		}
	}
	return nil, false
}
