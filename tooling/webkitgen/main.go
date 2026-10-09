// webkitgen emits internal/webkit/gen_funcs.go: purego prototypes for the C functions named in
// names.txt, with parameter and return types taken from the GIR files of WebKitGTK 4.1 and the
// libraries it sits on. Run it in the Linux stand: go run ./tooling/webkitgen
package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var girs = []string{"WebKit2-4.1", "JavaScriptCore-4.1", "Gtk-3.0", "GObject-2.0", "GLib-2.0", "Gio-2.0", "Soup-3.0"}

type typ struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
}

type param struct {
	Name      string    `xml:"name,attr"`
	Direction string    `xml:"direction,attr"`
	Nullable  string    `xml:"nullable,attr"`
	AllowNone string    `xml:"allow-none,attr"`
	Type      *typ      `xml:"type"`
	Array     *struct{} `xml:"array"`
	VarArgs   *struct{} `xml:"varargs"`
}

type fn struct {
	Kind   string
	Throws string  `xml:"throws,attr"`
	Ident  string  `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Ret    param   `xml:"return-value"`
	Inst   *param  `xml:"parameters>instance-parameter"`
	Params []param `xml:"parameters>parameter"`
	ns     string
}

type repo struct {
	funcs map[string]*fn
	kinds map[string]string // type name -> enum | callback
}

func load(dir, name string) (*repo, error) {
	f, err := os.Open(filepath.Join(dir, name+".gir"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := &repo{funcs: map[string]*fn{}, kinds: map[string]string{}}
	d := xml.NewDecoder(f)
	for {
		t, err := d.Token()
		if err != nil {
			break
		}
		se, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "function", "method", "constructor":
			var x fn
			if err := d.DecodeElement(&x, &se); err != nil {
				return nil, err
			}
			x.Kind = se.Name.Local
			if x.Ident != "" {
				r.funcs[x.Ident] = &x
			}
		case "enumeration", "bitfield", "callback":
			for _, a := range se.Attr {
				if a.Name.Local == "name" {
					k := "enum"
					if se.Name.Local == "callback" {
						k = "callback"
					}
					r.kinds[a.Value] = k
				}
			}
		}
	}
	return r, nil
}

var scalars = map[string]string{
	"gboolean": "int32", "gint": "int32", "gint32": "int32", "guint": "uint32", "guint32": "uint32",
	"gint64": "int64", "guint64": "uint64", "gsize": "uintptr", "gssize": "int", "glong": "int", "gulong": "uintptr",
	"gdouble": "float64", "gfloat": "float32", "GType": "uintptr", "gpointer": "uintptr", "gconstpointer": "uintptr",
	"guint8": "uint8", "gint8": "int8", "gint16": "int16", "guint16": "uint16", "GQuark": "uint32",
}

func (g *gen) goType(p param, ret bool) (string, error) {
	if p.VarArgs != nil {
		return "", fmt.Errorf("varargs")
	}
	if p.Array != nil || p.Type == nil {
		return "uintptr", nil
	}
	n := p.Type.Name
	if n == "none" {
		return "", nil
	}
	if p.Direction == "out" || p.Direction == "inout" {
		return "uintptr", nil
	}
	if s, ok := scalars[n]; ok && !strings.Contains(p.Type.CType, "*") {
		return s, nil
	}
	if (n == "utf8" || n == "filename") && !ret && p.Nullable != "1" && p.AllowNone != "1" {
		return "string", nil
	}
	if !strings.Contains(p.Type.CType, "*") {
		bare := n[strings.LastIndex(n, ".")+1:]
		for _, r := range g.repos {
			switch r.kinds[bare] {
			case "enum":
				return "int32", nil
			case "callback":
				return "uintptr", nil
			}
		}
		if s, ok := scalars[n]; ok {
			return s, nil
		}
		if s, ok := scalars[p.Type.CType]; ok {
			return s, nil
		}
		return "", fmt.Errorf("unknown by-value type %s", n)
	}
	return "uintptr", nil
}

type gen struct{ repos []*repo }

func main() {
	dir := flag.String("gir", "/usr/share/gir-1.0", "GIR directory")
	names := flag.String("names", "tooling/webkitgen/names.txt", "C functions to bind")
	out := flag.String("out", "internal/webkit/gen_funcs.go", "output file")
	flag.Parse()
	g := &gen{}
	for _, n := range girs {
		r, err := load(*dir, n)
		if err != nil {
			fmt.Fprintln(os.Stderr, "webkitgen:", err)
			os.Exit(1)
		}
		g.repos = append(g.repos, r)
	}
	nf, err := os.Open(*names)
	if err != nil {
		fmt.Fprintln(os.Stderr, "webkitgen:", err)
		os.Exit(1)
	}
	var idents []string
	sc := bufio.NewScanner(nf)
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" && !strings.HasPrefix(s, "#") {
			idents = append(idents, s)
		}
	}
	sort.Strings(idents)
	var vars, binds strings.Builder
	failed := false
	for _, id := range idents {
		var f *fn
		for _, r := range g.repos {
			if x := r.funcs[id]; x != nil {
				f = x
				break
			}
		}
		if f == nil {
			fmt.Fprintln(os.Stderr, "webkitgen: not in GIR:", id)
			failed = true
			continue
		}
		var ps []string
		if f.Inst != nil {
			ps = append(ps, "self uintptr")
		}
		for i, p := range f.Params {
			t, err := g.goType(p, false)
			if err != nil || t == "" {
				fmt.Fprintln(os.Stderr, "webkitgen:", id, "param", i, err)
				failed = true
				continue
			}
			ps = append(ps, fmt.Sprintf("a%d %s", i, t))
		}
		if f.Throws == "1" {
			ps = append(ps, "gerror uintptr")
		}
		rt, err := g.goType(f.Ret, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "webkitgen:", id, "return", err)
			failed = true
		}
		fmt.Fprintf(&vars, "\t%s func(%s) %s\n", id, strings.Join(ps, ", "), rt)
		fmt.Fprintf(&binds, "\tbindOne(&%s, %q, &missing)\n", id, id)
	}
	if failed {
		os.Exit(1)
	}
	src := "// Code generated by webkitgen from GIR files. DO NOT EDIT.\n//go:build linux\n\npackage webkit\n\nvar (\n" + vars.String() + ")\n\nfunc bindAll() (missing []string) {\n" + binds.String() + "\treturn\n}\n"
	b, err := format.Source([]byte(src))
	if err != nil {
		fmt.Fprintln(os.Stderr, "webkitgen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "webkitgen:", err)
		os.Exit(1)
	}
}
