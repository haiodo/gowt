package main

import (
	"fmt"
	"strings"
)

// C is the scalar kind of the C parameter when the signature came from GIR ("" for explicit ones).
type GoParam struct{ Name, Type, C string }

type Sig struct {
	Params []GoParam
	Ret    string // "" for void
	CRet   string // scalar kind of the C result
	Warn   string
}

var goKeywords = map[string]bool{"type": true, "func": true, "var": true, "range": true, "map": true, "chan": true,
	"go": true, "select": true, "interface": true, "package": true, "import": true, "return": true, "string": true,
	"len": true, "cap": true, "new": true, "make": true, "default": true, "fallthrough": true, "struct": true}

func (p *Param) ctype() string {
	if p.Type != nil && p.Type.CType != "" {
		return p.Type.CType
	}
	if p.Array != nil {
		return p.Array.CType
	}
	return ""
}

func (g *gen) paramGo(ix *Index, ct, dir string, noDir bool) (string, string) {
	base, stars := normCType(ct)
	pr := ix.primOf(base)
	switch {
	case stars == 0:
		if pr != "" {
			return pr, ""
		}
		if ix.structOf(base) != nil && !ix.Cbs[base] {
			return "int64", "struct by value: " + ct
		}
		return "int64", ""
	case stars == 1 && (dir == "out" || dir == "inout" || noDir) && pr != "":
		if pr == "bool" {
			pr = "int32"
		}
		if noDir && strings.Contains(ct, "const") && base == "char" {
			return "[]int8", ""
		}
		return "[]" + pr, ""
	case stars == 1 && pr == "int8" && (base == "gchar" || base == "char"):
		return "[]int8", ""
	case stars == 1:
		if rec := g.valueStruct(ix, base); rec != "" {
			return "*" + rec, ""
		}
	case stars == 2 && (dir == "out" || dir == "inout"):
		return "[]int64", ""
	}
	return "int64", ""
}

// valueStruct names the Go struct type for a C struct base that SWT passes by pointer.
func (g *gen) valueStruct(ix *Index, base string) string {
	if g.structs[base] {
		return exportName(base)
	}
	return ""
}

func (g *gen) defaultSig(ix *Index, fn *Callable, noDir bool) Sig {
	var s Sig
	used := map[string]bool{}
	add := func(name, typ string) {
		if name == "" || goKeywords[name] || used[name] {
			name = fmt.Sprintf("a%d", len(s.Params))
		}
		used[name] = true
		s.Params = append(s.Params, GoParam{name, typ, cKind(typ)})
	}
	if fn.Inst != nil {
		add(fn.Inst.Name, "int64")
	}
	for i := range fn.Params {
		p := &fn.Params[i]
		if p.Varargs != nil {
			s.Warn = "varargs"
			continue
		}
		t, w := g.paramGo(ix, p.ctype(), p.Direction, noDir)
		if w != "" {
			s.Warn = w
		}
		add(p.Name, t)
	}
	if fn.Throws == "1" {
		add("error", "[]int64")
	}
	rt := fn.Ret.Type
	if rt != nil {
		base, stars := normCType(rt.CType)
		switch {
		case stars > 0:
			s.Ret = "int64"
		case base == "void" || base == "none":
		default:
			if pr := ix.primOf(base); pr != "" {
				s.Ret = pr
			} else {
				s.Ret = "int64"
			}
		}
	}
	if fn.Ret.Array != nil {
		s.Ret = "int64"
	}
	s.CRet = cKind(s.Ret)
	return s
}

// cKind maps the Go type derived from a C type to the width the C side has.
func cKind(t string) string {
	switch t {
	case "int32", "bool":
		return "int32"
	case "int16", "int8", "float32", "float64":
		return t
	case "":
		return ""
	}
	return "uintptr"
}

// rawType is the type the purego-registered function takes for a wrapper parameter.
func rawType(p GoParam) string {
	t := p.Type
	switch {
	case t == "any" || t == "string" || strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "*"):
		return "unsafe.Pointer"
	case p.C != "":
		return p.C
	case t == "int64":
		return "uintptr"
	case t == "bool":
		return "int32"
	}
	return t
}

func rawConv(p GoParam) string {
	t, raw := p.Type, rawType(p)
	switch {
	case t == "string":
		return "sp(utf8z(" + p.Name + ", true))"
	case t == "any":
		return "anyp(" + p.Name + ")"
	case strings.HasPrefix(t, "[]"):
		return "sp(" + p.Name + ")"
	case strings.HasPrefix(t, "*"):
		return "unsafe.Pointer(" + p.Name + ")"
	case t == "bool":
		return "b2i(" + p.Name + ")"
	case t == raw:
		return p.Name
	}
	return raw + "(" + p.Name + ")"
}

func rawRet(sig Sig) string {
	switch {
	case sig.CRet != "":
		return sig.CRet
	case sig.Ret == "int64":
		return "uintptr"
	case sig.Ret == "bool":
		return "int32"
	}
	return sig.Ret
}
