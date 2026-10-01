package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The fit step turns go build errors into the parameter types the call sites pass (shapes_fit.txt):
// the call sites are the spec, so they beat the GIR-derived default.
var (
	argErr  = regexp.MustCompile(`^(\S+\.go):(\d+):(\d+): cannot use (.+?) \((?:variable|value|constant [^ ]+|untyped [a-z]+ constant)[^)]*? of (?:\w+ )?type ([^)]+)\) as ([^ ]+) value in argument to gtk\.(\w+)`)
	strErr  = regexp.MustCompile(`^(\S+\.go):(\d+):(\d+): cannot use (.+?) \(untyped (string|bool) constant[^)]*\) as ([^ ]+) value in argument to gtk\.(\w+)`)
	retErr  = regexp.MustCompile(`^(\S+\.go):(\d+):(\d+): cannot use (gtk\.(\w+)\(.*\)) \(value of type ([^)]+)\) as ([^ ]+) value in `)
	nilErr  = regexp.MustCompile(`^(\S+\.go):(\d+):(\d+): cannot use nil as ([^ ]+) value in argument to gtk\.(\w+)`)
	cntErr  = regexp.MustCompile(`^(\S+\.go):(\d+):(\d+): (?:too many|not enough) arguments in call to gtk\.(\w+)`)
	haveRe  = regexp.MustCompile(`^\s+have \((.*)\)$`)
	fitLine = regexp.MustCompile(`^arg (\S+) (\d+) (\S+)$`)
	retLine = regexp.MustCompile(`^ret (\S+) (\S+)$`)
)

type fitState struct {
	args  map[string]map[int]string
	rets  map[string]string
	funcs map[string]string // name -> "csym(types) ret" lines kept for explicit overrides
	notes []string
	asts  map[string]*ast.File
	fset  *token.FileSet
}

func loadFit(path string) *fitState {
	f := &fitState{args: map[string]map[int]string{}, rets: map[string]string{}, funcs: map[string]string{}, asts: map[string]*ast.File{}, fset: token.NewFileSet()}
	fh, err := os.Open(path)
	if err != nil {
		return f
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		if m := fitLine.FindStringSubmatch(sc.Text()); m != nil {
			i, _ := strconv.Atoi(m[2])
			f.set(m[1], i, m[3])
		}
		if m := retLine.FindStringSubmatch(sc.Text()); m != nil {
			f.rets[m[1]] = m[2]
		}
	}
	return f
}

func (f *fitState) set(name string, i int, t string) bool {
	if f.args[name] == nil {
		f.args[name] = map[int]string{}
	}
	if old := f.args[name][i]; old != "" && old != t && isStr(old) && isStr(t) {
		t = "any" // some call sites pass Go strings, some C-string byte arrays
	}
	if f.args[name][i] == t {
		return false
	}
	f.args[name][i] = t
	return true
}

func (f *fitState) save(path string) error {
	var names []string
	for n := range f.args {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("# Written by girgen -fit from go build errors; do not edit (put corrections in shapes.txt).\n")
	var rn []string
	for n := range f.rets {
		rn = append(rn, n)
	}
	sort.Strings(rn)
	for _, n := range rn {
		fmt.Fprintf(&b, "ret %s %s\n", n, f.rets[n])
	}
	for _, n := range names {
		var idx []int
		for i := range f.args[n] {
			idx = append(idx, i)
		}
		sort.Ints(idx)
		for _, i := range idx {
			fmt.Fprintf(&b, "arg %s %d %s\n", n, i, f.args[n][i])
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (f *fitState) argIndex(file string, line, col int, fn string) int {
	af, ok := f.asts[file]
	if !ok {
		af, _ = parser.ParseFile(f.fset, file, nil, 0)
		f.asts[file] = af
	}
	if af == nil {
		return -1
	}
	idx := -1
	ast.Inspect(af, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		se, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok || se.Sel.Name != fn {
			return true
		}
		for i, a := range ce.Args {
			p := f.fset.Position(a.Pos())
			if p.Line == line && p.Column == col {
				idx = i
			}
		}
		return true
	})
	return idx
}

func cleanType(t string) string {
	t = strings.ReplaceAll(t, "gtk.", "")
	if strings.HasPrefix(t, "untyped") {
		return ""
	}
	return t
}

// runFit applies one build log; it returns the number of changed entries.
func (f *fitState) runFit(log, srcRoot string) int {
	fh, err := os.Open(log)
	if err != nil {
		fatal(err)
	}
	defer fh.Close()
	changed := 0
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if m := argErr.FindStringSubmatch(line); m != nil {
			t := cleanType(m[5])
			ln, _ := strconv.Atoi(m[2])
			col, _ := strconv.Atoi(m[3])
			i := f.argIndex(srcRoot+"/"+m[1], ln, col, m[7])
			if t == "" || i < 0 {
				f.notes = append(f.notes, "unfit: "+line)
				continue
			}
			if old, ok := f.args[m[7]][i]; ok && old != t {
				f.notes = append(f.notes, fmt.Sprintf("conflict %s arg%d: %s vs %s", m[7], i, old, t))
			}
			if f.set(m[7], i, t) {
				changed++
			}
			continue
		}
		if m := strErr.FindStringSubmatch(line); m != nil {
			ln, _ := strconv.Atoi(m[2])
			col, _ := strconv.Atoi(m[3])
			if i := f.argIndex(srcRoot+"/"+m[1], ln, col, m[7]); i >= 0 && f.set(m[7], i, m[5]) {
				changed++
			}
			continue
		}
		if m := retErr.FindStringSubmatch(line); m != nil && balanced(m[4]) {
			if t := cleanType(m[7]); t != "" && f.rets[m[5]] != t {
				f.rets[m[5]] = t
				changed++
			}
			continue
		}
		if m := nilErr.FindStringSubmatch(line); m != nil {
			f.notes = append(f.notes, fmt.Sprintf("nil for %s: %s", m[5], line))
			continue
		}
		if m := cntErr.FindStringSubmatch(line); m != nil {
			f.notes = append(f.notes, "count: "+line)
		}
	}
	return changed
}

// balanced reports whether s is one call expression: its first "(" closes at the very end.
func balanced(s string) bool {
	d := 0
	for i, c := range s {
		switch c {
		case '(':
			d++
		case ')':
			d--
			if d == 0 && i != len(s)-1 {
				return false
			}
		}
	}
	return d == 0
}

func isStr(t string) bool { return t == "string" || t == "[]int8" || t == "any" }
