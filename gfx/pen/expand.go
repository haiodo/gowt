package pen

import (
	al "github.com/haiodo/gowt/gfx/pen/autolayout"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/gfx"
)

type paint struct {
	c     gfx.Color
	grad  []gfx.Stop // when set, a linear gradient
	angle float64    // degrees, CSS convention: 0 to the top, 90 to the right
}

type shadow struct {
	dx, dy, blur float64
	c            gfx.Color
}

// rn is a resolved node: refs expanded, overrides and variables applied, sizes parsed. The embedded
// Node holds what the layout needs and gives the result.
type rn struct {
	al.Node
	typ     string
	fill    *paint
	stroke  *paint
	sw      float64
	inner   bool
	radius  [4]float64 // top left, top right, bottom right, bottom left
	clip    bool
	shadow  *shadow
	opacity float64

	content, family string
	fsize, lh       float64
	weight          int
	growW           bool
	talign          string
	lines           []tline

	geom string
	vb   [4]float64

	kids []*rn
}

type ovmap map[string]map[string]any

func get[T any](m map[string]any, k string) (T, bool) {
	v, ok := m[k].(T)
	return v, ok
}

func clone(m map[string]any) map[string]any {
	c := make(map[string]any, len(m)+2)
	for k, v := range m {
		c[k] = v
	}
	return c
}

// mergeDesc overlays hi on lo, key by key; both are descendants maps.
func mergeDesc(lo, hi map[string]any) map[string]any {
	if len(lo) == 0 {
		return hi
	}
	if len(hi) == 0 {
		return lo
	}
	out := clone(lo)
	for k, v := range hi {
		hm, _ := v.(map[string]any)
		lm, _ := out[k].(map[string]any)
		if hm == nil || lm == nil {
			out[k] = v
			continue
		}
		out[k] = mergeProps(lm, hm)
	}
	return out
}

func mergeProps(lo, hi map[string]any) map[string]any {
	out := clone(lo)
	for k, v := range hi {
		if k == "descendants" {
			l, _ := out[k].(map[string]any)
			h, _ := v.(map[string]any)
			out[k] = mergeDesc(l, h)
			continue
		}
		out[k] = v
	}
	return out
}

func toOv(m map[string]any) ovmap {
	o := make(ovmap, len(m))
	for k, v := range m {
		if vm, ok := v.(map[string]any); ok {
			o[k] = vm
		}
	}
	return o
}

func mergeOv(lo, hi ovmap) ovmap {
	if len(lo) == 0 {
		return hi
	}
	if len(hi) == 0 {
		return lo
	}
	out := make(ovmap, len(lo)+len(hi))
	for k, v := range lo {
		out[k] = v
	}
	for k, v := range hi {
		if l, ok := out[k]; ok {
			out[k] = mergeProps(l, v)
		} else {
			out[k] = v
		}
	}
	return out
}

// override applies o to the node m: a property overlay, or a replacement when o names another
// type or component.
func override(m, o map[string]any) (res map[string]any, replaced bool) {
	if t, ok := o["type"]; ok && (t != m["type"] || o["ref"] != m["ref"]) {
		r := clone(o)
		for _, k := range [...]string{"x", "y"} {
			if _, has := r[k]; !has {
				if v, ok := m[k]; ok {
					r[k] = v
				}
			}
		}
		return r, true
	}
	r := mergeProps(m, o)
	delete(r, "id")
	r["id"] = m["id"]
	return r, false
}

type expander struct {
	d     *Doc
	nodes int
}

// expand resolves m. pfx is "F:" for the nodes of a library component, "" for the file's own.
func (e *expander) expand(m map[string]any, pfx string, ov ovmap, mode string, depth int) *rn {
	if depth > 60 {
		return nil
	}
	if id, _ := m["id"].(string); id != "" {
		if o := ov[pfx+id]; o != nil {
			var replaced bool
			m, replaced = override(m, o)
			if replaced {
				pfx = ""
			}
		}
	}
	if en, ok := m["enabled"].(bool); ok && !en {
		return nil
	}
	if th, ok := m["theme"].(map[string]any); ok {
		for k, v := range th {
			if strings.HasSuffix(k, "mode") {
				if s, ok := v.(string); ok {
					mode = s
				}
			}
		}
	}
	typ, _ := m["type"].(string)
	if typ == "ref" {
		return e.expandRef(m, pfx, ov, mode, depth)
	}
	n := e.newRN(m, typ, mode)
	if n == nil {
		return nil
	}
	for _, c := range kids(m) {
		if k := e.expand(c, pfx, ov, mode, depth+1); k != nil {
			n.kids = append(n.kids, k)
			n.Kids = append(n.Kids, &k.Node)
		}
	}
	return n
}

func (e *expander) expandRef(m map[string]any, pfx string, ov ovmap, mode string, depth int) *rn {
	s, _ := m["ref"].(string)
	alias, id := splitRef(s)
	if !strings.Contains(s, ":") && pfx != "" {
		alias = strings.TrimSuffix(pfx, ":")
	}
	comp := e.d.index[alias][id]
	if comp == nil {
		e.d.Skipped["unresolved ref "+s]++
		return nil
	}
	merged := clone(comp)
	for k, v := range m {
		switch k {
		case "id", "type", "ref", "name", "descendants", "children":
		default:
			merged[k] = v
		}
	}
	desc, _ := m["descendants"].(map[string]any)
	childOv := mergeOv(toOv(desc), ov)
	if comp["type"] == "ref" {
		// a component that is itself an instance: its own overrides sit under ours
		cd, _ := comp["descendants"].(map[string]any)
		merged["descendants"] = cd
		merged["ref"] = comp["ref"]
		return e.expandRef(merged, alias+":", childOv, mode, depth+1)
	}
	typ, _ := merged["type"].(string)
	n := e.newRN(merged, typ, mode)
	if n == nil {
		return nil
	}
	cpfx := alias + ":"
	src := kids(comp)
	if own := kids(m); len(own) > 0 {
		src, cpfx = own, ""
	}
	for _, c := range src {
		if k := e.expand(c, cpfx, childOv, mode, depth+1); k != nil {
			n.kids = append(n.kids, k)
			n.Kids = append(n.Kids, &k.Node)
		}
	}
	return n
}

func (e *expander) newRN(m map[string]any, typ, mode string) *rn {
	switch typ {
	case "frame", "rectangle", "ellipse", "path", "text":
	default:
		e.d.Skipped["type "+typ]++
		return nil
	}
	e.nodes++
	n := &rn{typ: typ, opacity: 1, lh: 0}
	n.W, n.H = parseSize(m["width"]), parseSize(m["height"])
	if x, ok := get[float64](m, "x"); ok {
		n.X = x
	}
	if y, ok := get[float64](m, "y"); ok {
		n.Y = y
	}
	if v, ok := get[float64](m, "opacity"); ok {
		n.opacity = v
	}
	switch m["layout"] {
	case "vertical":
		n.Dir = al.Column
	case "none":
		n.Dir = al.Abs
	}
	if typ != "frame" {
		n.Dir = al.Abs
	}
	n.Gap, _ = get[float64](m, "gap")
	n.Pad = parsePad(m["padding"])
	n.Align = parseAlign(m["alignItems"])
	n.Justify = parseAlign(m["justifyContent"])
	n.clip, _ = get[bool](m, "clip")
	n.fill = e.parsePaint(m["fill"], mode)
	n.stroke = e.parsePaint(m["stroke"], mode)
	n.sw, _ = get[float64](m, "strokeWidth")
	if n.stroke != nil && n.sw == 0 {
		n.sw = 1
	}
	n.inner = m["strokeAlignment"] == "inner"
	switch r := m["cornerRadius"].(type) {
	case float64:
		n.radius = [4]float64{r, r, r, r}
	case []any:
		if len(r) == 4 {
			for i := range r {
				n.radius[i], _ = r[i].(float64)
			}
		}
	}
	n.shadow = e.parseShadow(m["effect"], mode)
	switch typ {
	case "text":
		n.content, _ = get[string](m, "content")
		n.family, _ = get[string](m, "fontFamily")
		n.fsize, _ = get[float64](m, "fontSize")
		if n.fsize == 0 {
			n.fsize = 14
		}
		n.lh, _ = get[float64](m, "lineHeight")
		n.weight = parseWeight(m["fontWeight"])
		g, _ := get[string](m, "textGrowth")
		n.growW = g == "fixed-width" || g == "fixed-width-height"
		n.talign, _ = get[string](m, "textAlign")
	case "path":
		n.geom, _ = get[string](m, "geometry")
		n.vb = [4]float64{0, 0, 0, 0}
		if vb, ok := m["viewBox"].([]any); ok && len(vb) == 4 {
			for i := range vb {
				n.vb[i], _ = vb[i].(float64)
			}
		}
	}
	return n
}

func parseSize(v any) al.Size {
	switch x := v.(type) {
	case float64:
		return al.Size{Kind: al.Fixed, V: x}
	case string:
		if strings.HasPrefix(x, "fill_container") {
			return al.Size{Kind: al.Fill}
		}
	}
	return al.Size{}
}

func parseAlign(v any) al.Align {
	switch v {
	case "center":
		return al.Center
	case "end":
		return al.End
	case "space_between":
		return al.SpaceBetween
	}
	return al.Start
}

func parsePad(v any) [4]float64 {
	switch p := v.(type) {
	case float64:
		return [4]float64{p, p, p, p}
	case []any:
		f := func(i int) float64 { x, _ := p[i].(float64); return x }
		switch len(p) {
		case 2:
			return [4]float64{f(0), f(1), f(0), f(1)}
		case 4:
			return [4]float64{f(0), f(1), f(2), f(3)}
		}
	}
	return [4]float64{}
}

func parseWeight(v any) int {
	switch x := v.(type) {
	case string:
		if n, err := strconv.Atoi(x); err == nil {
			return n
		}
		if x == "bold" {
			return 700
		}
	case float64:
		return int(x)
	}
	return 400
}

// color resolves a hex literal or a $variable.
func (e *expander) color(s string, mode string) (gfx.Color, bool) {
	if strings.HasPrefix(s, "$") {
		name := s[1:]
		if i := strings.IndexByte(name, ':'); i >= 0 {
			name = name[i+1:]
		}
		v := e.d.vars[name]
		if v == nil {
			e.d.Skipped["unknown variable"]++
			return gfx.Color{}, false
		}
		switch val := v["value"].(type) {
		case string:
			return e.color(val, mode)
		case []any:
			var pick string
			for _, it := range val {
				im, _ := it.(map[string]any)
				th, _ := im["theme"].(map[string]any)
				cs, _ := im["value"].(string)
				if pick == "" {
					pick = cs
				}
				for _, tv := range th {
					if tv == mode {
						pick = cs
					}
				}
			}
			return e.color(pick, mode)
		}
		return gfx.Color{}, false
	}
	return parseHex(s)
}

func parseHex(s string) (gfx.Color, bool) {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3, 4:
		var t strings.Builder
		for _, c := range s {
			t.WriteRune(c)
			t.WriteRune(c)
		}
		s = t.String()
	}
	if len(s) != 6 && len(s) != 8 {
		return gfx.Color{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return gfx.Color{}, false
	}
	if len(s) == 6 {
		v = v<<8 | 0xFF
	}
	return gfx.Color{R: float64(v>>24&0xFF) / 255, G: float64(v>>16&0xFF) / 255, B: float64(v>>8&0xFF) / 255, A: float64(v&0xFF) / 255}, true
}

func (e *expander) parsePaint(v any, mode string) *paint {
	switch x := v.(type) {
	case string:
		if c, ok := e.color(x, mode); ok {
			return &paint{c: c}
		}
	case map[string]any:
		if en, ok := x["enabled"].(bool); ok && !en {
			return nil
		}
		if x["type"] != "gradient" {
			e.d.Skipped["fill "+toStr(x["type"])]++
			return nil
		}
		cols, _ := x["colors"].([]any)
		p := &paint{}
		for _, ci := range cols {
			cm, _ := ci.(map[string]any)
			cs, _ := cm["color"].(string)
			c, ok := e.color(cs, mode)
			if !ok {
				continue
			}
			pos, _ := cm["position"].(float64)
			p.grad = append(p.grad, gfx.Stop{Offset: pos, Color: c})
		}
		if len(p.grad) == 0 {
			return nil
		}
		if x["gradientType"] != "linear" {
			e.d.Skipped["gradient "+toStr(x["gradientType"])+" drawn linear"]++
		}
		p.c = p.grad[0].Color
		rot, _ := x["rotation"].(float64)
		p.angle = rot
		return p
	}
	return nil
}

func toStr(v any) string {
	s, _ := v.(string)
	return s
}

func (e *expander) parseShadow(v any, mode string) *shadow {
	var m map[string]any
	switch x := v.(type) {
	case map[string]any:
		m = x
	case []any:
		for _, it := range x {
			if im, ok := it.(map[string]any); ok && im["type"] == "shadow" {
				m = im
				break
			}
		}
	}
	if m == nil {
		return nil
	}
	if m["type"] != "shadow" || m["shadowType"] == "inner" {
		e.d.Skipped["effect "+toStr(m["type"])+" "+toStr(m["shadowType"])]++
		return nil
	}
	s := &shadow{c: gfx.Color{A: 0.25}}
	if cs, ok := m["color"].(string); ok {
		if c, ok := e.color(cs, mode); ok {
			s.c = c
		}
	}
	if off, ok := m["offset"].(map[string]any); ok {
		s.dx, _ = off["x"].(float64)
		s.dy, _ = off["y"].(float64)
	}
	s.blur, _ = m["blur"].(float64)
	return s
}

// systemFont maps IBM Plex Sans to its PostScript names when the font is installed, and to
// Helvetica Neue otherwise.
func systemFont() func(string, int) string {
	have := false
	home, _ := os.UserHomeDir()
	for _, dir := range []string{"/Library/Fonts", "/System/Library/Fonts", filepath.Join(home, "Library/Fonts"), "/usr/share/fonts", filepath.Join(home, ".fonts")} {
		if fs, _ := filepath.Glob(filepath.Join(dir, "IBMPlexSans*")); len(fs) > 0 {
			have = true
			break
		}
	}
	return func(family string, weight int) string {
		if have && strings.HasPrefix(family, "IBM Plex Sans") {
			switch {
			case weight >= 700:
				return "IBMPlexSans-Bold"
			case weight >= 600:
				return "IBMPlexSans-SemiBold"
			case weight >= 500:
				return "IBMPlexSans-Medium"
			}
			return "IBMPlexSans"
		}
		switch {
		case weight >= 700:
			return "HelveticaNeue-Bold"
		case weight >= 500:
			return "HelveticaNeue-Medium"
		}
		return "HelveticaNeue"
	}
}

// FontFallback reports whether text falls back to the system font.
func (d *Doc) FontFallback() bool {
	return !strings.HasPrefix(d.o.FontName("IBM Plex Sans", 400), "IBMPlex")
}
