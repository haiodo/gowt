package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// jstubs writes Java declarations of internal/gtk's exported API (types, constants, functions as
// static natives) for j2go to resolve the widget code against: the type information comes from our
// generated binding, not from SWT's PI sources. A Go name <Class><member> maps to Java class
// <Class>, member <member>, the way j2go forms call-site names.

// javaBases are Java names (overloaded natives) the Go glue spells differently.
var javaBases = map[string]string{"OSCallProc": "Call", "OSCallProcII": "Call", "OSCallFunctionArg0Arg1Arg2Arg3": "call", "OSCallFunctionArg0Arg1Arg2Arg3Arg4Arg5": "call"}

var jClasses = []string{"GTK3", "GTK4", "GTK", "GDK", "Cairo", "Graphene", "OS", "Converter", "C"}

var jPrims = map[string]string{"int64": "long", "int32": "int", "int16": "short", "int8": "byte", "uint16": "char",
	"bool": "boolean", "float32": "float", "float64": "double", "string": "String", "any": "Object", "uint32": "int", "uint8": "byte"}

type jType struct {
	name   string
	fields []jField
	meths  []*ast.FuncDecl
	iface  *ast.InterfaceType
	alias  string
	sizeof string
}

type jField struct{ name, typ string }

func (g *jgen) jtype(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		if j, ok := jPrims[t.Name]; ok {
			return j
		}
		return javaName(t.Name)
	case *ast.StarExpr:
		return g.jtype(t.X)
	case *ast.ArrayType:
		return g.jtype(t.Elt) + "[]"
	case *ast.Ellipsis:
		return g.jtype(t.Elt) + "[]"
	}
	return "Object"
}

type jFunc struct {
	goName, member string
	d              *ast.FuncDecl
}

type jgen struct {
	scope  *types.Scope        // the type-checked internal/gtk, for the Java type of its vars
	ctypes map[string][]string // Java types of call-site arguments, by Go name
	spell  map[string]string   // Class.Capitalized -> the spelling EPL code uses
	csyms  map[string]string   // Go name -> C symbol of explicit shapes (the Java base name)
	groups map[string][]jFunc
	over   map[string]string // erasure key -> Go member name
	fset   *token.FileSet
	types  map[string]*jType
	consts map[string][2]string // name -> java type, value
	vars   map[string]ast.Expr  // name -> initializer or type expr
	funcs  []*ast.FuncDecl
	arity  map[string]int // call-site argument count of variadic functions
}

// extraJava holds members j2go special-cases by their Java key (see manual.txt).
var extraJava = map[string]string{"OS": "\t// Converter.java (EPL) calls this overload; the Go glue has no use for it.\n\tpublic static native long g_utf8_to_utf16(byte[] p0, long p1, long[] p2, long[] p3, long[] p4);\n", "GdkRectangle": "\tpublic native org.eclipse.swt.graphics.Rectangle toRectangle();\n"}

func pkgOf(name string) string {
	switch {
	case name == "GTK3" || strings.HasPrefix(name, "GdkEvent") || name == "GdkGeometry" || name == "GdkWindowAttr" || name == "GtkTargetEntry":
		return "gtk3"
	case name == "GTK4":
		return "gtk4"
	case name == "Cairo" || strings.HasPrefix(name, "Cairo_") || strings.HasPrefix(name, "cairo_"):
		return "cairo"
	case name == "C" || name == "Converter":
		return "org.eclipse.swt.internal"
	}
	return "gtk"
}

// javaName is how Java spells a generated type: cairo's C struct names stay lower case there.
func javaName(n string) string {
	if strings.HasPrefix(n, "Cairo_") {
		return "c" + n[1:]
	}
	return n
}

func fullPkg(name string) string {
	if p := pkgOf(name); strings.Contains(p, ".") {
		return p
	} else {
		return "org.eclipse.swt.internal." + p
	}
}

func jImports() string {
	return "import org.eclipse.swt.internal.gtk.*;\nimport org.eclipse.swt.internal.gtk3.*;\nimport org.eclipse.swt.internal.gtk4.*;\nimport org.eclipse.swt.internal.cairo.*;\n\n"
}

func splitClass(name string) (class, member string) {
	for _, c := range jClasses {
		if strings.HasPrefix(name, c) && len(name) > len(c) {
			return c, name[len(c):]
		}
	}
	return "", name
}

func writeJStubs(srcDir, outDir string, calls map[string]int, spell map[string]string, csyms map[string]string) error {
	ct, scope := callTypes("swt")
	g := &jgen{ctypes: ct, scope: scope, spell: spell, csyms: csyms, groups: map[string][]jFunc{}, over: map[string]string{}, fset: token.NewFileSet(), types: map[string]*jType{}, consts: map[string][2]string{}, vars: map[string]ast.Expr{}, arity: calls}
	files, _ := filepath.Glob(filepath.Join(srcDir, "*.go"))
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(g.fset, p, nil, 0)
		if err != nil {
			return err
		}
		g.collect(f)
	}
	os.MkdirAll(outDir, 0o755)
	classes := map[string]*strings.Builder{}
	body := func(c string) *strings.Builder {
		if classes[c] == nil {
			classes[c] = &strings.Builder{}
		}
		return classes[c]
	}
	for n := range g.types {
		body(n)
	}
	for n, cv := range g.consts {
		g.addConst(body, n, cv)
	}
	for n, e := range g.vars {
		g.addVar(body, n, e)
	}
	for _, fd := range g.funcs {
		g.addFunc(body, fd)
	}
	g.emitGroups(body)
	for name, t := range g.types {
		b := body(name)
		for _, f := range t.fields {
			fmt.Fprintf(b, "\tpublic %s %s;\n", f.typ, lcfirst2(f.name))
		}
		for _, m := range t.meths {
			if t.iface == nil {
				fmt.Fprintf(b, "\tpublic native %s;\n", g.sig(m, 0))
			}
		}
		if t.iface != nil {
			for _, m := range t.iface.Methods.List {
				ft := m.Type.(*ast.FuncType)
				fmt.Fprintf(b, "\t%s;\n", g.sig(&ast.FuncDecl{Name: m.Names[0], Type: ft}, 0))
			}
		}
	}
	var ov []string
	for k, v := range g.over {
		ov = append(ov, k+"="+v)
	}
	sort.Strings(ov)
	if err := os.WriteFile(filepath.Join(outDir, "names.properties"), []byte(strings.Join(ov, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	var names []string
	for n := range classes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if n == "Converter" {
			continue // the EPL Converter.java is used as is
		}
		dir := filepath.Join(outDir, strings.ReplaceAll(fullPkg(n), ".", "/"))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		kind, ext := "class", ""
		if t := g.types[n]; t != nil {
			if t.iface != nil {
				kind = "interface"
			}
			if t.alias != "" {
				ext = " extends " + t.alias
			}
			if i := g.implements(t); i != "" {
				ext += " implements " + i
			}
		}
		src := fmt.Sprintf("// Generated by girgen from internal/gtk's Go declarations. DO NOT EDIT.\npackage %s;\n\n%spublic %s %s%s {\n%s%s}\n", fullPkg(n), jImports(), kind, javaName(n), ext, classes[n].String(), extraJava[n])
		if err := os.WriteFile(filepath.Join(dir, javaName(n)+".java"), []byte(src), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (g *jgen) collect(f *ast.File) {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			if d.Recv != nil {
				if tn := recvName(d.Recv); tn != "" {
					g.typ(tn).meths = append(g.typ(tn).meths, d)
				}
				continue
			}
			if strings.HasPrefix(d.Name.Name, "New") && g.types[d.Name.Name[3:]] != nil {
				continue
			}
			if hasFuncParam(d) {
				continue
			}
			g.funcs = append(g.funcs, d)
		case *ast.GenDecl:
			g.collectGen(d)
		}
	}
}

func hasFuncParam(d *ast.FuncDecl) bool {
	for _, p := range d.Type.Params.List {
		if _, ok := p.Type.(*ast.FuncType); ok {
			return true
		}
	}
	return false
}

func recvName(r *ast.FieldList) string {
	switch t := r.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func (g *jgen) typ(n string) *jType {
	if g.types[n] == nil {
		g.types[n] = &jType{name: n}
	}
	return g.types[n]
}

func (g *jgen) collectGen(d *ast.GenDecl) {
	for _, s := range d.Specs {
		switch s := s.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}
			t := g.typ(s.Name.Name)
			switch ty := s.Type.(type) {
			case *ast.StructType:
				for _, f := range ty.Fields.List {
					for _, n := range f.Names {
						if n.IsExported() {
							t.fields = append(t.fields, jField{n.Name, g.jtype(f.Type)})
						}
					}
				}
			case *ast.InterfaceType:
				t.iface = ty
			case *ast.Ident:
				if s.Assign.IsValid() {
					t.alias = ty.Name
				}
			}
		case *ast.ValueSpec:
			for i, n := range s.Names {
				if !n.IsExported() {
					continue
				}
				if d.Tok == token.CONST {
					var v ast.Expr
					if i < len(s.Values) {
						v = s.Values[i]
					}
					g.consts[n.Name] = [2]string{g.constType(s.Type, v), g.expr(v)}
				} else {
					if s.Type != nil {
						g.vars[n.Name] = &ast.CompositeLit{Type: s.Type}
					} else if i < len(s.Values) {
						g.vars[n.Name] = s.Values[i]
					}
				}
			}
		}
	}
}

func (g *jgen) expr(e ast.Expr) string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	printer.Fprint(&b, g.fset, e)
	return b.String()
}

func (g *jgen) valueType(e ast.Expr) string {
	if c, ok := e.(*ast.CallExpr); ok {
		switch f := c.Fun.(type) {
		case *ast.Ident:
			if f.Name == "cs" {
				return "byte[]"
			}
			return "long"
		case *ast.SelectorExpr:
			if f.Sel.Name == "Getenv" {
				return "String"
			}
		}
	}
	if b, ok := e.(*ast.BinaryExpr); ok && (b.Op == token.NEQ || b.Op == token.EQL) {
		return "boolean"
	}
	return "long"
}

func (g *jgen) constType(t ast.Expr, v ast.Expr) string {
	if t != nil {
		return g.jtype(t)
	}
	if l, ok := v.(*ast.BasicLit); ok {
		switch l.Kind {
		case token.STRING:
			return "String"
		case token.FLOAT:
			return "double"
		}
	}
	if id, ok := v.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
		return "boolean"
	}
	return "int"
}

func lcfirst2(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// sig renders a method or function; arity > 0 replaces a variadic tail by that many fixed parameters.
func (g *jgen) sig(d *ast.FuncDecl, arity int) string {
	var ps []string
	for _, p := range d.Type.Params.List {
		n := max(len(p.Names), 1)
		if _, ok := p.Type.(*ast.Ellipsis); ok {
			for len(ps) < arity {
				ps = append(ps, fmt.Sprintf("%s p%d", g.argType(d.Name.Name, len(ps), "Object"), len(ps)))
			}
			continue
		}
		for i := 0; i < n; i++ {
			ps = append(ps, fmt.Sprintf("%s p%d", g.argType(d.Name.Name, len(ps), g.jtype(p.Type)), len(ps)))
		}
	}
	ret := "void"
	if r := d.Type.Results; r != nil && len(r.List) == 1 {
		ret = g.jtype(r.List[0].Type)
	}
	return fmt.Sprintf("%s %s(%s)", ret, lcfirst2(d.Name.Name), strings.Join(ps, ", "))
}

func (g *jgen) implements(t *jType) string {
	if t.iface != nil {
		return ""
	}
	have := map[string]bool{}
	for _, m := range t.meths {
		have[m.Name.Name] = true
	}
	for _, i := range g.types {
		if i.iface == nil || len(i.iface.Methods.List) == 0 {
			continue
		}
		ok := true
		for _, m := range i.iface.Methods.List {
			ok = ok && have[m.Names[0].Name]
		}
		if ok {
			return i.name
		}
	}
	return ""
}

func (g *jgen) member(name string) (class, member string, isType bool) {
	if strings.HasSuffix(name, "Sizeof") {
		base := strings.TrimSuffix(name, "Sizeof")
		g.typ(base)
		return base, "sizeof", true
	}
	c, m := splitClass(name)
	return c, m, false
}

func (g *jgen) addConst(body func(string) *strings.Builder, name string, cv [2]string) {
	c, m, size := g.member(name)
	if c == "" {
		return
	}
	m = g.spelledIf(c, m, size)
	typ, val := cv[0], cv[1]
	if size {
		typ = "int"
	}
	if val == "" || strings.ContainsAny(val, "()") && !strings.HasPrefix(val, "-") || strings.Contains(val, ".") && typ == "int" {
		fmt.Fprintf(body(c), "\tpublic static %s %s;\n", typ, m)
		return
	}
	if typ == "int" {
		var n int64
		if _, err := fmt.Sscan(val, &n); err == nil && (n > 2147483647 || n < -2147483648) {
			typ, val = "long", val+"L"
		}
	}
	fmt.Fprintf(body(c), "\tpublic static final %s %s = %s;\n", typ, m, val)
}

func (g *jgen) addVar(body func(string) *strings.Builder, name string, e ast.Expr) {
	c, m, _ := g.member(name)
	if c == "" {
		return
	}
	m = g.spelledIf(c, m, false)
	t := "long"
	if o := g.scope.Lookup(name); o != nil {
		t = javaOf(o.Type())
		fmt.Fprintf(body(c), "\tpublic static %s %s;\n", t, m)
		return
	}
	switch v := e.(type) {
	case *ast.CompositeLit:
		t = g.jtype(v.Type)
	case *ast.CallExpr:
		t = g.callType(v)
	case *ast.BinaryExpr:
		t = "boolean"
	}
	fmt.Fprintf(body(c), "\tpublic static %s %s;\n", t, m)
}

func (g *jgen) callType(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		if f.Name == "cs" {
			return "byte[]"
		}
		for _, d := range g.funcs {
			if d.Name.Name == f.Name && d.Type.Results != nil {
				return g.jtype(d.Type.Results.List[0].Type)
			}
		}
		if t, ok := jPrims[f.Name]; ok {
			return t
		}
		return "long"
	case *ast.SelectorExpr:
		if f.Sel.Name == "Getenv" {
			return "String"
		}
	}
	return "long"
}

func (g *jgen) spelled(class, member string) string {
	if sp, ok := g.spell[class+"."+strings.ToUpper(member[:1])+member[1:]]; ok {
		return sp
	}
	return member
}

// addFunc groups the functions of one Java base name: several Go names on one base are Java
// overloads, whose Go names are pinned through the generated names.properties.
func (g *jgen) addFunc(body func(string) *strings.Builder, d *ast.FuncDecl) {
	c, m := splitClass(d.Name.Name)
	if c == "" {
		return
	}
	base := g.csyms[d.Name.Name]
	if b, ok := javaBases[d.Name.Name]; ok {
		base = b
	}
	if base == "" {
		base = g.spelled(c, overloadRe.ReplaceAllString(m, ""))
	}
	g.groups[c+"#"+base] = append(g.groups[c+"#"+base], jFunc{d.Name.Name, m, d})
}

var overloadRe = regexp.MustCompile(`Overload\d+$`)

func erasureOf(jt string) string {
	if strings.HasSuffix(jt, "[]") {
		e := strings.TrimSuffix(jt, "[]")
		switch e {
		case "byte":
			return "[B"
		case "int":
			return "[I"
		case "long":
			return "[J"
		case "short":
			return "[S"
		case "char":
			return "[C"
		case "double":
			return "[D"
		case "float":
			return "[F"
		case "boolean":
			return "[Z"
		}
		return "[L" + qualified(e) + ";"
	}
	return qualified(jt)
}

func qualified(t string) string {
	switch t {
	case "byte":
		return "B"
	case "int":
		return "I"
	case "long":
		return "J"
	case "short":
		return "S"
	case "char":
		return "C"
	case "double":
		return "D"
	case "float":
		return "F"
	case "boolean":
		return "Z"
	case "String":
		return "java.lang.String"
	case "Object":
		return "java.lang.Object"
	}
	return fullPkg(t) + "." + javaName(t)
}

func (g *jgen) emitGroups(body func(string) *strings.Builder) {
	for key, fs := range g.groups {
		class, base := key[:strings.Index(key, "#")], key[strings.Index(key, "#")+1:]
		for _, f := range fs {
			sig := g.sig(&ast.FuncDecl{Name: &ast.Ident{Name: f.goName}, Type: f.d.Type}, g.arity[f.goName])
			fmt.Fprintf(body(class), "\tpublic static native %s;\n", strings.Replace(sig, lcfirst2(f.goName)+"(", base+"(", 1))
			if len(fs) > 1 || strings.ToUpper(base[:1])+base[1:] != f.member {
				g.over[g.key(class, base, f.d)] = f.member
			}
		}
	}
}

// argType refines an Object parameter by what the call sites pass.
func (g *jgen) argType(goName string, i int, j string) string {
	if j == "Object" {
		if ts := g.ctypes[goName]; i < len(ts) {
			return ts[i]
		}
	}
	return j
}

func (g *jgen) key(class, base string, d *ast.FuncDecl) string {
	var ts []string
	n := g.arity[d.Name.Name]
	for _, p := range d.Type.Params.List {
		cnt := max(len(p.Names), 1)
		if _, ok := p.Type.(*ast.Ellipsis); ok {
			for len(ts) < n {
				ts = append(ts, erasureOf(g.argType(d.Name.Name, len(ts), "Object")))
			}
			continue
		}
		for i := 0; i < cnt; i++ {
			ts = append(ts, erasureOf(g.argType(d.Name.Name, len(ts), g.jtype(p.Type))))
		}
	}
	return fullPkg(class) + "." + class + "#" + base + "(" + strings.Join(ts, ",") + ")"
}

func (g *jgen) spelledIf(class, member string, size bool) string {
	if size {
		return member
	}
	return g.spelled(class, member)
}

// eplSpellings collects how the EPL Java sources spell members of the PI classes (g_object_set,
// VERSION, commit, ...): Go names only keep the capitalized form.
func eplSpellings(dirs []string) (map[string]string, error) {
	re := regexp.MustCompile(`\b(OS|GTK3|GTK4|GTK|GDK|Cairo|Graphene|Converter|C)\.([A-Za-z_]\w*)`)
	out := map[string]string{}
	for _, d := range dirs {
		err := filepath.WalkDir(d, func(p string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() && strings.Contains(e.Name(), "Eclipse SWT PI") {
				return filepath.SkipDir // the clean room: PI sources are never read
			}
			if e.IsDir() || !strings.HasSuffix(p, ".java") {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				out[m[1]+"."+strings.ToUpper(m[2][:1])+m[2][1:]] = m[2]
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
