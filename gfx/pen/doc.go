// Package pen loads pen.dev design files (.pen, JSON) into gfx scene subtrees. It is experimental
// and uses only the public gfx API and the standard library, so it can be copied to another module.
//
// Open reads a file and the libraries it imports; Screens lists the top level frames and instances
// that are big enough to be screens; Build lays one out and adds its nodes to a gfx.Tx, so the screen
// is a subtree that can be moved as a whole and whose elements can be hit-tested; Unsupported reports
// what was left out. The auto-layout is in gfx/layout.
//
// Supported: frame, rectangle, ellipse, path (SVG path data with a viewBox), text; ref instances of
// components from the file or from imported libraries, with descendants overrides (properties, whole
// node replacement, slot children); color variables with light and dark themes and a per-instance
// theme; solid fills and linear gradients; strokes with inner or centered alignment; outer shadows;
// uniform and per-corner radius; clip (kept only where it changes the picture); opacity; text with
// family, size, weight, line height, fixed-width wrapping and center or right alignment, measured by
// the Measurer given in Options. Text sizes follow pen.dev: a hug width is the measured width rounded up,
// a line box is lineHeight * fontSize rounded (the font's own ascent plus descent when none is given).
// Fonts that are not installed fall back to the system font; cg.RegisterFonts adds IBM Plex Sans
// to a process without installing it. Boxes lists the laid out rectangles under pen.dev's ids.
//
// Not supported: fill rules (paths are filled non-zero, evenodd icons render as solid blobs),
// inner shadows, blur effects, radial and angular gradients (drawn linear), image fills, per-side
// strokes, rotation, flip, text decoration and letter spacing, the guess for the gradient rotation
// (CSS convention), min and max sizes and wrap in layouts, refs to missing components (skipped),
// hug sizes of frames that hold fill children (the content size is used).
package pen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		d.o.FontName = systemFont(d.o.Measurer)
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
	ID    string
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
			id, _ := m["id"].(string)
			out = append(out, Info{i, id, name, w, h})
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

// Unsupported lists what the loader met and left out so far, one line per feature with a count,
// sorted. It grows as screens are built.
func (d *Doc) Unsupported() []string {
	out := make([]string, 0, len(d.Skipped))
	for k, n := range d.Skipped {
		out = append(out, fmt.Sprintf("%s (%d)", k, n))
	}
	sort.Strings(out)
	return out
}
