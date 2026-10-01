package main

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

// callTypes type-checks package swt (errors ignored) and records, for every gtk.<Name> call, the
// Java types of its arguments: the Go signatures of the memmove family take `any`, so only the call
// sites tell which Java overload each Go name stands for.
func callTypes(swtDir string) (map[string][]string, *types.Scope) {
	fset := token.NewFileSet()
	var files []*ast.File
	list, _ := filepath.Glob(filepath.Join(swtDir, "*.go"))
	for _, p := range list {
		base := filepath.Base(p)
		if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_darwin.go") || strings.HasSuffix(base, "_windows.go") {
			continue
		}
		if f, err := parser.ParseFile(fset, p, nil, 0); err == nil {
			files = append(files, f)
		}
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil), Error: func(e error) {
		if os.Getenv("GIRGEN_DEBUG") != "" {
			println(e.Error())
		}
	}}
	conf.Check("swt", fset, files, info)
	var scope *types.Scope
	if p, err := conf.Importer.Import("github.com/haiodo/gowt/internal/gtk"); err == nil {
		scope = p.Scope()
	}
	out := map[string][]string{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			se, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := se.X.(*ast.Ident); !ok || id.Name != "gtk" {
				return true
			}
			var ts []string
			for _, a := range ce.Args {
				ts = append(ts, argJava(info, a))
			}
			if old, seen := out[se.Sel.Name]; !seen || strings.Contains(strings.Join(old, ","), "Object") {
				out[se.Sel.Name] = ts
			}
			return true
		})
	}
	return out, scope
}

func javaOf(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		if j, ok := jPrims[t.Name()]; ok {
			return j
		}
		switch t.Kind() {
		case types.UntypedInt:
			return "int"
		case types.UntypedString:
			return "String"
		}
	case *types.Slice:
		return javaOf(t.Elem()) + "[]"
	case *types.Pointer:
		return javaOf(t.Elem())
	case *types.Named:
		return javaName(t.Obj().Name())
	case *types.Alias:
		return javaName(t.Obj().Name())
	}
	return "Object"
}

// argJava is the Java type of a call argument: a bare integer constant is an int (the Go type it
// converted to is the parameter's), a converted or computed value keeps its Go type.
func argJava(info *types.Info, a ast.Expr) string {
	tv := info.Types[a]
	if _, call := a.(*ast.CallExpr); !call && tv.Value != nil && tv.Value.Kind() == constant.Int {
		return "int"
	}
	return javaOf(tv.Type)
}
