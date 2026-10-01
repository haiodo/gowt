package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Shapes are the corrections to GIR-derived signatures read from shapes.txt and shapes_fit.txt;
// the directives (arg, ret, func, str, const, extra, wide, bitbool, need) are in tooling/j2go/README.md.
type Shapes struct {
	Arg     map[string]map[int]string
	Ret     map[string]string
	Func    map[string]ExplicitFn
	Extra   map[string][]GoParam // Go-only trailing fields of generated structs
	Wide    map[string]bool      // structs whose 8/16-bit fields SWT reads as int32
	BitBool map[string]bool      // structs whose bitfields SWT reads as bool, not int32
	Need    []string             // structs only the glue uses, no call site names them
	Skip    map[string]bool
	Str     map[string]string // NUL-terminated []int8 values the name rules do not derive
	Const   map[string]string // plain constants GIR does not carry (X11 headers, macros)
}

type ExplicitFn struct {
	CSym string
	Sig  Sig
}

func loadShapes(path string) (*Shapes, error) {
	s := &Shapes{Arg: map[string]map[int]string{}, Ret: map[string]string{}, Func: map[string]ExplicitFn{}, Skip: map[string]bool{}, Extra: map[string][]GoParam{}, Wide: map[string]bool{}, BitBool: map[string]bool{}, Str: map[string]string{}, Const: map[string]string{}}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" || line[0] == '#' {
			continue
		}
		fs := strings.Fields(line)
		switch fs[0] {
		case "arg":
			i, _ := strconv.Atoi(fs[2])
			if s.Arg[fs[1]] == nil {
				s.Arg[fs[1]] = map[int]string{}
			}
			s.Arg[fs[1]][i] = fs[3]
		case "ret":
			if fs[2] == "-" {
				fs[2] = ""
			}
			s.Ret[fs[1]] = fs[2]
		case "str":
			s.Str[fs[1]] = strings.Join(fs[2:], " ")
		case "const":
			s.Const[fs[1]] = fs[2]
		case "extra":
			s.Extra[fs[1]] = append(s.Extra[fs[1]], GoParam{Name: fs[2], Type: fs[3]})
		case "wide":
			for _, n := range fs[1:] {
				s.Wide[n] = true
			}
		case "bitbool":
			for _, n := range fs[1:] {
				s.BitBool[n] = true
			}
		case "need":
			s.Need = append(s.Need, fs[1:]...)
		case "func":
			rest := strings.TrimSpace(strings.TrimPrefix(line, "func "))
			name, rest, _ := strings.Cut(rest, " ")
			csym, rest, _ := strings.Cut(strings.TrimSpace(rest), "(")
			params, ret, _ := strings.Cut(rest, ")")
			sig := Sig{Ret: strings.TrimSpace(ret)}
			for _, p := range strings.Split(params, ",") {
				pf := strings.Fields(p)
				if len(pf) == 2 {
					sig.Params = append(sig.Params, GoParam{Name: pf[0], Type: pf[1]})
				}
			}
			s.Func[name] = ExplicitFn{strings.TrimSpace(csym), sig}
		}
	}
	return s, sc.Err()
}

// merge lets the hand-written file win over the learned one.
func (s *Shapes) merge(o *Shapes) {
	for k := range o.Wide {
		s.Wide[k] = true
	}
	for k := range o.BitBool {
		s.BitBool[k] = true
	}
	for k, v := range o.Extra {
		s.Extra[k] = append(s.Extra[k], v...)
	}
	s.Need = append(s.Need, o.Need...)
	for n, m := range o.Arg {
		if s.Arg[n] == nil {
			s.Arg[n] = map[int]string{}
		}
		for i, t := range m {
			s.Arg[n][i] = t
		}
	}
	for k, v := range o.Ret {
		s.Ret[k] = v
	}
	for k, v := range o.Func {
		s.Func[k] = v
	}
	for k := range o.Skip {
		s.Skip[k] = true
	}
	for k, v := range o.Str {
		s.Str[k] = v
	}
	for k, v := range o.Const {
		s.Const[k] = v
	}
}
