package main

import (
	"os"
	"regexp"
	"strings"
)

var (
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cairoProto   = regexp.MustCompile(`cairo_public\s+([^;()]+?)\s*\b(cairo_\w+)\s*\(([^;]*?)\)\s*(?:cairo_private\s*)?;`)
)

// addCairoProtos reads function prototypes from the cairo headers: the cairo GIR carries types
// only. Parameters have no direction there; the generator's pointer rules pick the Go shape.
func addCairoProtos(ix *Index, files []string) {
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		src := blockComment.ReplaceAllString(string(b), "")
		for _, m := range cairoProto.FindAllStringSubmatch(src, -1) {
			c := &Callable{Name: m[2], CIdent: m[2]}
			c.Ret.Type = &Type{CType: strings.Join(strings.Fields(m[1]), " ")}
			params := strings.TrimSpace(m[3])
			if params != "void" && params != "" {
				for _, p := range strings.Split(params, ",") {
					c.Params = append(c.Params, protoParam(p))
				}
			}
			ix.Funcs[c.CIdent], ix.FuncNS[c.CIdent] = c, "cairo.h" // the headers beat the near-empty cairo GIR
		}
	}
}

// protoParam splits "const double *dashes" into the type text and drops the name.
func protoParam(p string) Param {
	p = strings.Join(strings.Fields(p), " ")
	p = strings.ReplaceAll(p, "* ", "*")
	toks := strings.Split(p, " ")
	name := ""
	if len(toks) > 1 {
		last := toks[len(toks)-1]
		stars := strings.TrimLeft(last, "*")
		if len(toks) > 1 && !isTypeWord(stars) {
			name = stars
			toks[len(toks)-1] = last[:len(last)-len(stars)]
			if toks[len(toks)-1] == "" {
				toks = toks[:len(toks)-1]
			}
		}
	}
	ct := strings.Join(toks, " ")
	if i := strings.Index(ct, "[]"); i >= 0 {
		ct = ct[:i] + "*"
	}
	pr := Param{Name: name}
	pr.Type = &Type{CType: ct}
	return pr
}

func isTypeWord(s string) bool {
	switch s {
	case "int", "double", "char", "long", "unsigned", "float", "void", "short", "const":
		return true
	}
	return strings.HasSuffix(s, "_t") && strings.HasPrefix(s, "cairo_") && !strings.Contains(s, "_t_")
}
