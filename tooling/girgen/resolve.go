package main

import (
	"regexp"
	"strings"
)

// Sym is one call-site name resolved against the GIR.
type Sym struct {
	Go      string
	Kind    string // func, const, str, type, size, sizefn, unresolved
	CName   string
	Fn      *Callable
	Gtk4    bool // only in the GTK 4 GIR: compile-only stub
	Value   string
	Rec     *Record
	TypeCT  string
	Derived bool
}

var classPrefixes = []string{"GTK3", "GTK4", "GTK", "GDK", "Cairo", "Graphene", "OS", "C", "Converter"}

var sizeofRe = regexp.MustCompile(`^(.+?)(?:Sizeof|_sizeof)$`)

func lcfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

type Resolver struct {
	ix3, ix4 *Index
	shapes   *Shapes
	called   map[string]bool
}

func (r *Resolver) record(name string) (*Record, string) {
	for _, ix := range []*Index{r.ix3, r.ix4} {
		for _, c := range []string{name, lcfirst(name)} {
			if rec, ok := ix.Types[c]; ok {
				return rec, c
			}
			if t, ok := ix.Aliases[c]; ok {
				if rec, ok := ix.Types[t]; ok {
					return rec, t
				}
			}
		}
	}
	return nil, ""
}

var memmoveRe = regexp.MustCompile(`^(?:GTK3|GTK4|GTK|GDK|Cairo|OS|C)Memmove(?:Overload\d+)?$`)

func (r *Resolver) resolve(name string) *Sym {
	s := &Sym{Go: name}
	if e, ok := r.shapes.Func[name]; ok {
		s.Kind, s.CName = "explicit", e.CSym
		return s
	}
	if v, ok := r.shapes.Str[name]; ok {
		s.Kind, s.Value = "str", v
		return s
	}
	if v, ok := r.shapes.Const[name]; ok {
		s.Kind, s.Value = "const", v
		return s
	}
	if r.shapes.Skip[name] {
		s.Kind = "skip"
		return s
	}
	if memmoveRe.MatchString(name) {
		s.Kind = "memmove"
		return s
	}
	if m := sizeofRe.FindStringSubmatch(name); m != nil {
		base := m[1]
		kind := "size"
		if strings.HasSuffix(name, "_sizeof") {
			kind = "sizefn"
			for _, p := range classPrefixes {
				if strings.HasPrefix(base, p) {
					base = base[len(p):]
					break
				}
			}
		}
		if rec, ct := r.record(base); rec != nil {
			s.Kind, s.Rec, s.TypeCT = kind, rec, ct
			return s
		}
	}
	if rec, ct := r.record(name); rec != nil {
		s.Kind, s.Rec, s.TypeCT = "type", rec, ct
		return s
	}
	for _, p := range classPrefixes {
		if !strings.HasPrefix(name, p) {
			continue
		}
		rest := name[len(p):]
		if rest == "" {
			continue
		}
		order := []*Index{r.ix3, r.ix4}
		if p == "GTK4" {
			order = []*Index{r.ix4, r.ix3}
		}
		for i, ix := range order {
			gtk4 := (ix == r.ix4) && (p != "GTK4" || i == 0)
			if fn, ok := ix.Funcs[lcfirst(rest)]; ok {
				s.Kind, s.CName, s.Fn, s.Gtk4 = "func", lcfirst(rest), fn, ix == r.ix4
				_ = gtk4
				return s
			}
			cands := []string{rest}
			if strings.HasPrefix(rest, "GDK_") {
				cands = append(cands, "GDK_KEY_"+rest[4:])
			}
			for _, c := range cands {
				if v, ok := ix.Consts[c]; ok {
					s.Kind, s.CName, s.Value = "const", c, v
					if ix.StrConsts[c] {
						s.Kind = "str"
					}
					return s
				}
			}
			if p == "OS" || p == "GTK" || p == "GTK3" {
				hy := strings.ToLower(strings.ReplaceAll(rest, "_", "-"))
				if ix.Props[hy] {
					s.Kind, s.CName, s.Value = "str", hy, hy
					return s
				}
			}
		}
	}
	if m := typeMacroRe.FindStringSubmatch(name); m != nil {
		s.Kind, s.CName = "typefn", strings.ToLower(m[1])+"_"+strings.ToLower(m[3])+"_get_type"
		if m[2] == "IS" {
			s.Kind = "isfn"
		}
		return s
	}
	if m := getTypeRe.FindStringSubmatch(name); m != nil {
		s.Kind, s.CName = "typefn", lcfirst(m[1])
		return s
	}
	// Remaining OS<lower_snake> names are GObject signal and GTK style property names; GIR does not
	// list style properties, so they are derived by the same hyphen rule (and reported by -report).
	if m := snakeRe.FindStringSubmatch(name); m != nil && !r.called[name] {
		v := strings.ToLower(strings.ReplaceAll(m[1], "_", "-"))
		if strings.HasPrefix(v, "notify-") {
			v = "notify::" + v[len("notify-"):]
		}
		s.Kind, s.Value, s.Derived = "str", v, true
		return s
	}
	s.Kind = "unresolved"
	return s
}

var getTypeRe = regexp.MustCompile(`^(?:GTK3|GTK4|GTK|GDK|OS)((?:Gtk|Gdk|Pango)_[a-z0-9_]+_get_type)$`)

var snakeRe = regexp.MustCompile(`^OS([A-Z][a-z0-9]*(?:_[a-z0-9]+)*)$`)

// GTK_TYPE_WIDGET and GTK_IS_WIDGET are macros over <x>_<y>_get_type().
var typeMacroRe = regexp.MustCompile(`^(?:GTK3|GTK4|GTK|GDK|OS)(GTK|GDK|PANGO)_(TYPE|IS)_([A-Z0-9_]+)$`)
