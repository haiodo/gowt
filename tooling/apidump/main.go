// Command apidump prints the exported API of package swt, one symbol per line, sorted:
// package-level funcs/vars/consts/types and the full method set (promoted included) of
// every exported type. Diff two dumps to see what a regeneration changed.
package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	dir := "swt"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	fset := token.NewFileSet()
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	var files []*ast.File
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, m, nil, 0)
		if err != nil {
			fail(err)
		}
		files = append(files, f)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check("github.com/haiodo/gowt/swt", fset, files, nil)
	if err != nil {
		fail(err)
	}
	qual := types.RelativeTo(pkg)
	var lines []string
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		if !obj.Exported() {
			continue
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			lines = append(lines, types.ObjectString(obj, qual))
			continue
		}
		lines = append(lines, "type "+name)
		ms := types.NewMethodSet(types.NewPointer(tn.Type()))
		if types.IsInterface(tn.Type()) {
			ms = types.NewMethodSet(tn.Type())
		}
		for i := 0; i < ms.Len(); i++ {
			f := ms.At(i).Obj()
			if f.Exported() {
				lines = append(lines, name+"."+f.Name()+strings.TrimPrefix(types.TypeString(f.Type(), qual), "func"))
			}
		}
	}
	sort.Strings(lines)
	fmt.Println(strings.Join(lines, "\n"))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
