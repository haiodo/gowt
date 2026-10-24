// Package pen loads pen.dev design files (.pen, JSON) into gfx scene subtrees. It is a tool for
// the demos and benchmarks, not a public API: the files it reads are never part of the repository.
//
// Supported: frame, rectangle, ellipse, path (SVG geometry), text, ref (component instances with
// descendants overrides, library imports, theme variables), a subset of auto-layout, fills with
// colors and linear gradients, strokes, outer shadows. Everything else is skipped silently.
package pen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/haiodo/gowt/gfx"
)

// Options configure Open.
type Options struct {
	// Measurer measures text as the renderer draws it; gfx.DefaultMeasurer when nil.
	Measurer gfx.TextMeasurer
	// FontName maps a family and a weight (400, 500, 600, 700) to the name the renderer takes. The
	// default uses IBM Plex Sans when it is installed and Helvetica Neue otherwise.
	FontName func(family string, weight int) string
	// Mode picks the variable values: "light" (default) or "dark".
	Mode string
	// FirstID is the id of the first node built; ids grow from there.
	FirstID int
}

// Doc is an opened .pen file with its libraries.
type Doc struct {
	o     Options
	top   []map[string]any
	index map[string]map[string]map[string]any // alias -> id -> node; alias "" is the file itself
	vars  map[string]map[string]any
	next  int
	font  map[fontKey]string
	// Skipped counts what the loader ignored, by feature.
	Skipped map[string]int
}

type fontKey struct {
	family string
	weight int
}

// Open reads the file and the libraries it imports.
func Open(path string, o Options) (*Doc, error) {
	root, err := readJSON(path)
	if err != nil {
		return nil, err
	}
	d := &Doc{o: o, index: map[string]map[string]map[string]any{}, vars: map[string]map[string]any{}, next: o.FirstID,
		font: map[fontKey]string{}, Skipped: map[string]int{}}
	if d.o.Measurer == nil {
		d.o.Measurer = gfx.DefaultMeasurer
	}
	if d.o.FontName == nil {
		d.o.FontName = systemFont()
	}
	if d.o.Mode == "" {
		d.o.Mode = "light"
	}
	if d.next == 0 {
		d.next = 1
	}
	if imp, ok := root["imports"].(map[string]any); ok {
		for alias, rel := range imp {
			lib, err := readJSON(filepath.Join(filepath.Dir(path), fmt.Sprint(rel)))
			if err != nil {
				return nil, fmt.Errorf("pen: import %s: %w", alias, err)
			}
			d.addLib(alias, lib)
		}
	}
	d.addLib("", root)
	d.top = kids(root)
	return d, nil
}

func readJSON(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("pen: %s: %w", path, err)
	}
	return m, nil
}

func (d *Doc) addLib(alias string, root map[string]any) {
	idx := map[string]map[string]any{}
	var walk func(m map[string]any)
	walk = func(m map[string]any) {
		if id, ok := m["id"].(string); ok {
			idx[id] = m
		}
		for _, c := range kids(m) {
			walk(c)
		}
	}
	for _, c := range kids(root) {
		walk(c)
	}
	d.index[alias] = idx
	if v, ok := root["variables"].(map[string]any); ok {
		for k, x := range v {
			if xm, ok := x.(map[string]any); ok {
				d.vars[k] = xm
			}
		}
	}
}

func kids(m map[string]any) []map[string]any {
	l, _ := m["children"].([]any)
	out := make([]map[string]any, 0, len(l))
	for _, c := range l {
		if cm, ok := c.(map[string]any); ok {
			out = append(out, cm)
		}
	}
	return out
}

// NextID returns the id the next built node takes.
func (d *Doc) NextID() int { return d.next }

func (d *Doc) newID() int {
	d.next++
	return d.next - 1
}

func splitRef(s string) (alias, id string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return "", s
}

// Info describes one top level node that is big enough to be a screen.
type Info struct {
	Index int
	Name  string
	W, H  float64
}

// Screens lists the top level frames and instances of at least 400 x 300 units; the smaller ones are
// page titles and notes.
func (d *Doc) Screens() []Info {
	var out []Info
	for i, m := range d.top {
		w, h := d.topSize(m)
		if w >= 400 && h >= 300 {
			name, _ := m["name"].(string)
			out = append(out, Info{i, name, w, h})
		}
	}
	return out
}

func (d *Doc) topSize(m map[string]any) (w, h float64) {
	for depth := 0; depth < 8; depth++ {
		if v, ok := m["width"].(float64); ok && w == 0 {
			w = v
		}
		if v, ok := m["height"].(float64); ok && h == 0 {
			h = v
		}
		if m["type"] != "ref" {
			break
		}
		s, _ := m["ref"].(string)
		alias, id := splitRef(s)
		c := d.index[alias][id]
		if c == nil {
			break
		}
		m = c
	}
	return
}
