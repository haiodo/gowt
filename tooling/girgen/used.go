package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const gtkImport = `"github.com/haiodo/gowt/internal/gtk"`

// usedNames lists every gtk.<Name> selector in the Go files under roots: the call sites are the
// interface spec, so only these names get generated.
func usedNames(roots []string) ([]string, map[string]bool, error) {
	set := map[string]bool{}
	called := map[string]bool{}
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
				return err
			}
			if strings.Contains(p, "/internal/gtk/") {
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return err
			}
			local := ""
			for _, im := range f.Imports {
				if im.Path.Value == gtkImport {
					local = "gtk"
					if im.Name != nil {
						local = im.Name.Name
					}
				}
			}
			if local == "" {
				return nil
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if ce, ok := n.(*ast.CallExpr); ok {
					if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
						if id, ok := se.X.(*ast.Ident); ok && id.Name == local && id.Obj == nil {
							called[se.Sel.Name] = true
						}
					}
				}
				if se, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := se.X.(*ast.Ident); ok && id.Name == local && id.Obj == nil {
						set[se.Sel.Name] = true
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, called, nil
}

// glueNames lists the top-level names the hand-written glue_*.go files declare; the generator
// leaves those alone.
func glueNames(dir string) (map[string]bool, error) {
	set := map[string]bool{}
	fset := token.NewFileSet()
	files, _ := filepath.Glob(filepath.Join(dir, "glue_*.go"))
	for _, p := range files {
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					set[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						set[sp.Name.Name] = true
					case *ast.ValueSpec:
						for _, n := range sp.Names {
							set[n.Name] = true
						}
					}
				}
			}
		}
	}
	return set, nil
}

// callArity maps each gtk.<Name> that is called to its argument count (variadic Go functions need it).
func callArity(roots []string) (map[string]int, error) {
	out := map[string]int{}
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.Contains(p, "/internal/gtk/") {
				return err
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if ce, ok := n.(*ast.CallExpr); ok {
					if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
						if id, ok := se.X.(*ast.Ident); ok && id.Name == "gtk" && id.Obj == nil {
							out[se.Sel.Name] = len(ce.Args)
						}
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
