package jrt

import (
	"bytes"
	"io/fs"
)

// resourceFS backs Class.getResourceAsStream: a package registers its embed.FS once via
// RegisterResources, and GetResourceAsStream looks names up there.
var resourceFS fs.FS

func RegisterResources(f fs.FS) { resourceFS = f }

// GetResourceAsStream is Class.getResourceAsStream(name): nil if no FS was registered or name
// isn't in it, matching Java's own "resource not found" return.
func GetResourceAsStream(name string) InputStream {
	if resourceFS == nil {
		return nil
	}
	data, err := fs.ReadFile(resourceFS, name)
	if err != nil {
		return nil
	}
	return NewInputStream(bytes.NewReader(data))
}
