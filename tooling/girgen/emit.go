package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type gen struct {
	res      *Resolver
	shapes   *Shapes
	structs  map[string]bool
	layouts  map[string]*Layout
	warnings []string
	syms     []*Sym
	emitted  map[string]bool
	unpacks  map[string]bool // generated structs with an unpack() to run after C filled them
}

func (g *gen) warn(f string, a ...any) { g.warnings = append(g.warnings, fmt.Sprintf(f, a...)) }

func (g *gen) sigFor(s *Sym) (Sig, string) {
	if e, ok := g.shapes.Func[s.Go]; ok {
		return g.overrides(s.Go, e.Sig), e.CSym
	}
	ix := g.res.ix3
	if s.Gtk4 {
		ix = g.res.ix4
	}
	sig := g.overrides(s.Go, g.defaultSig(ix, s.Fn, g.res.ix3.FuncNS[s.CName] == "cairo.h"))
	if sig.Warn != "" {
		g.warn("%s: %s", s.Go, sig.Warn)
	}
	return sig, s.CName
}

func (g *gen) overrides(name string, sig Sig) Sig {
	for i, t := range g.shapes.Arg[name] {
		if i < len(sig.Params) {
			sig.Params[i].Type = t
		}
	}
	if t, ok := g.shapes.Ret[name]; ok {
		sig.Ret = t
	}
	return sig
}

func (g *gen) emitFunc(w *strings.Builder, name, csym string, sig Sig) {
	var raws, params, args, post []string
	for i, p := range sig.Params {
		if goKeywords[p.Name] {
			p.Name = fmt.Sprintf("a%d", i)
		}
		raws = append(raws, rawType(p))
		params = append(params, p.Name+" "+p.Type)
		args = append(args, rawConv(p))
		if strings.HasPrefix(p.Type, "*") && g.unpacks[p.Type[1:]] {
			post = append(post, fmt.Sprintf("if %s != nil {\n\t\t%s.unpack()\n\t}", p.Name, p.Name))
		}
	}
	rr := rawRet(sig)
	rt := ""
	if sig.Ret != "" {
		rt = " " + rr
	}
	fmt.Fprintf(w, "var r_%s = lz[func(%s)%s]{sym: %q}\n", name, strings.Join(raws, ", "), rt, csym)
	ret := ""
	if sig.Ret != "" {
		ret = " " + sig.Ret
	}
	call := fmt.Sprintf("r_%s.get()(%s)", name, strings.Join(args, ", "))
	fmt.Fprintf(w, "func %s(%s)%s {\n", name, strings.Join(params, ", "), ret)
	switch {
	case sig.Ret == "":
		fmt.Fprintf(w, "\t%s\n", call)
	case len(post) > 0 && sig.Ret == "bool":
		fmt.Fprintf(w, "\tr := %s != 0\n", call)
	case len(post) > 0:
		fmt.Fprintf(w, "\tr := %s(%s)\n", sig.Ret, call)
	case sig.Ret == "bool":
		fmt.Fprintf(w, "\treturn %s != 0\n", call)
	case sig.Ret == rr:
		fmt.Fprintf(w, "\treturn %s\n", call)
	default:
		fmt.Fprintf(w, "\treturn %s(%s)\n", sig.Ret, call)
	}
	for _, p := range post {
		fmt.Fprintf(w, "\t%s\n", p)
	}
	if len(post) > 0 && sig.Ret != "" {
		w.WriteString("\treturn r\n")
	}
	w.WriteString("}\n\n")
}

func constLit(v string) string {
	if n, ok := parseInt(v); ok {
		if n > 0x7fffffff && n <= 0xffffffff {
			n = int64(int32(uint32(n)))
		}
		return strconv.FormatInt(n, 10)
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v
	}
	return strconv.Quote(v)
}

func (g *gen) emitAll() (funcs, consts, types string) {
	g.collectUnpacks()
	var fw, cw, tw strings.Builder
	syms := append([]*Sym(nil), g.syms...)
	sort.Slice(syms, func(i, j int) bool { return syms[i].Go < syms[j].Go })
	recs := map[string]*Record{}
	var order []string
	need := func(ct string, r *Record) {
		if _, ok := recs[ct]; !ok {
			recs[ct] = r
			order = append(order, ct)
		}
	}
	for _, s := range syms {
		switch s.Kind {
		case "func":
			sig, csym := g.sigFor(s)
			g.emitFunc(&fw, s.Go, csym, sig)
		case "memmove":
			fmt.Fprintf(&fw, "func %s(dst, src any, n ...int64) { memmove(dst, src, n...) }\n\n", s.Go)
		case "explicit":
			e := g.shapes.Func[s.Go]
			g.emitFunc(&fw, s.Go, e.CSym, g.overrides(s.Go, e.Sig))
		case "typefn":
			fmt.Fprintf(&fw, "func %s() int64 { return gtypeOf(%q) }\n\n", s.Go, s.CName)
		case "isfn":
			fmt.Fprintf(&fw, "func %s(obj int64) bool { return isA(obj, gtypeOf(%q)) }\n\n", s.Go, s.CName)
		case "const":
			fmt.Fprintf(&cw, "const %s = %s\n", s.Go, constLit(s.Value))
		case "str":
			fmt.Fprintf(&cw, "var %s = cs(%q)\n", s.Go, s.Value)
		case "type", "size", "sizefn":
			need(s.TypeCT, s.Rec)
		}
	}
	for i := 0; i < len(order); i++ { // layouts may add nested records
		ct := order[i]
		l := g.layoutOf(g.res.ix3, recs[ct])
		_ = l
	}
	sort.Strings(order)
	for _, ct := range order {
		l := g.layoutOf(g.res.ix3, recs[ct])
		g.emitted[exportName(ct)] = true
		g.emitStruct(&tw, exportName(ct), l, g.shapes.Extra[ct], g.shapes.Wide[ct], g.shapes.BitBool[ct])
	}
	for _, s := range syms {
		switch s.Kind {
		case "type":
			if exportName(s.TypeCT) != s.Go {
				fmt.Fprintf(&tw, "type %s = %s\n\n", s.Go, exportName(s.TypeCT))
			}
		case "size":
			fmt.Fprintf(&cw, "const %s = %d\n", s.Go, g.layoutOf(g.res.ix3, s.Rec).Size)
		case "sizefn":
			fmt.Fprintf(&fw, "func %s() int32 { return %d }\n\n", s.Go, g.layoutOf(g.res.ix3, s.Rec).Size)
		}
	}
	return fw.String(), cw.String(), tw.String()
}

// collectUnpacks notes which structs get an unpack(): the wrappers call it after C filled one in.
func (g *gen) collectUnpacks() {
	g.unpacks = map[string]bool{}
	for _, s := range g.syms {
		if s.Kind != "type" && s.Kind != "size" && s.Kind != "sizefn" {
			continue
		}
		l := g.layoutOf(g.res.ix3, s.Rec)
		if len(l.Bits) > 0 || g.shapes.Wide[s.TypeCT] {
			g.unpacks[exportName(s.TypeCT)] = true
			g.unpacks[s.Go] = true
		}
	}
}
