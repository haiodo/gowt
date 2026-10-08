// Package lucide embeds the subset of Lucide icons (https://lucide.dev, ISC; the Feather-derived
// ones MIT, see LICENSE) that gowt's icon dictionary names. Stroke icons on a 24x24 grid, drawn
// with currentColor.
package lucide

import "embed"

//go:embed *.svg
var files embed.FS

// SVG returns the icon source by its Lucide name ("settings", "folder-open").
func SVG(name string) ([]byte, bool) {
	b, err := files.ReadFile(name + ".svg")
	return b, err == nil
}

// Names lists the embedded icons.
func Names() []string {
	ents, _ := files.ReadDir(".")
	names := make([]string, len(ents))
	for i, e := range ents {
		names[i] = e.Name()[:len(e.Name())-len(".svg")]
	}
	return names
}
