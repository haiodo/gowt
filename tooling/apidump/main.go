// Command apidump prints the exported API of package swt for one GOOS, one symbol per line, sorted:
// package-level funcs/vars/consts/types and the full method set (promoted included) of every
// exported type. Diff two dumps to see what a regeneration changed.
//
//	apidump [-goos os] [dir]    dump (GOOS from -goos, else $GOOS, else the host)
//	apidump -check [dir]        dump every platform in platforms.txt and compare them: each platform
//	                            lacking a symbol another one has is listed; the run fails unless
//	                            every difference is in platform-only.txt (see README "Round 19")
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const cfgDir = "tooling/apidump"

func main() {
	goos := flag.String("goos", "", "target GOOS (default $GOOS, else the host)")
	check := flag.Bool("check", false, "compare the platforms listed in "+cfgDir+"/platforms.txt")
	flag.Parse()
	dir := "swt"
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	if *check {
		os.Exit(compare(dir))
	}
	if *goos == "" {
		*goos = os.Getenv("GOOS")
	}
	if *goos == "" {
		*goos = runtime.GOOS
	}
	lines, err := dump(dir, *goos)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(strings.Join(lines, "\n"))
}

// dump type-checks dir as built for goos (file suffixes and tags apply) and lists its exported API.
func dump(dir, goos string) ([]string, error) {
	ctx := build.Default
	ctx.GOOS, ctx.CgoEnabled = goos, false
	build.Default = ctx // the source importer of the dependencies reads this one
	bp, err := ctx.ImportDir(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("GOOS=%s: %v", goos, err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range bp.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	var errs []error
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil), Error: func(e error) { errs = append(errs, e) }}
	pkg, _ := conf.Check("github.com/haiodo/gowt/swt", fset, files, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("GOOS=%s: %s does not type-check, %d errors, first: %v", goos, dir, len(errs), errs[0])
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
	return publicOnly(lines), nil
}

// publicOnly keeps the symbols of SWT's public Java API (public-api.txt, written by j2go with J2GO_DUMP_PUBLIC):
// a Java package-private member is exported in Go too, and differs between platform sources without being API.
func publicOnly(lines []string) []string {
	f, err := os.Open(filepath.Join(cfgDir, "public-api.txt"))
	if err != nil {
		return lines
	}
	defer f.Close()
	public := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := sc.Text(); !strings.HasPrefix(l, "#") {
			public[l] = true
		}
	}
	var out []string
	for _, l := range lines {
		// Functions named for a platform (ShellWin32_new, ColorCocoa_new) are glue, not API.
		if strings.Contains(strings.SplitN(l, "(", 2)[0], "Cocoa_") || strings.Contains(strings.SplitN(l, "(", 2)[0], "Win32_") || strings.Contains(strings.SplitN(l, "(", 2)[0], "Gtk_") {
			continue
		}
		// A signature naming a platform's PI type is that platform's own, never API.
		if strings.Contains(l, "internal/cocoa") || strings.Contains(l, "internal/win32") || strings.Contains(l, "internal/gtk") {
			continue
		}
		switch {
		case strings.HasPrefix(l, "type "):
			if public[l] {
				out = append(out, l)
			}
		case strings.HasPrefix(l, "func "):
			if public["func "+strings.TrimPrefix(strings.SplitN(l, "(", 2)[0], "func ")] {
				out = append(out, l)
			}
		case strings.Contains(strings.SplitN(l, "(", 2)[0], "."):
			head := strings.SplitN(l, "(", 2)[0]
			if public["method "+head] {
				out = append(out, l)
			}
		default: // const/var: the shared SWT constants and the statics j2go lists
			name := strings.Fields(l)
			if len(name) > 1 && public["func "+name[1]] {
				out = append(out, l)
			}
		}
	}
	return out
}

// compare returns the exit code: 0 when the platforms agree up to platform-only.txt.
var exemptions []*regexp.Regexp

func compare(dir string) int {
	platforms := readList(filepath.Join(cfgDir, "platforms.txt"))
	// platform-only.txt: "<goos>[,<goos>...] <dump line>" - API SWT itself has on those platforms only.
	allowed := map[string]bool{}
	var exempt []*regexp.Regexp // "* <regexp>" lines: symbols the ports legitimately declare differently (a class one port stubs by hand)
	for _, l := range readList(filepath.Join(cfgDir, "platform-only.txt")) {
		if strings.HasPrefix(l, "* ") {
			exempt = append(exempt, regexp.MustCompile(l[2:]))
			continue
		}
		allowed[l] = true
	}
	exemptions = exempt
	report, bad, err := diff(dir, platforms, allowed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, l := range report {
		fmt.Println(l)
	}
	fmt.Printf("apidump: platforms %s, %d unexplained difference(s)\n", strings.Join(platforms, " "), bad)
	if bad > 0 {
		return 1
	}
	return 0
}

// diff lists every symbol some platform lacks, except the allowed "<platforms having it> <line>" ones.
func diff(dir string, platforms []string, allowed map[string]bool) (report []string, bad int, err error) {
	has := map[string]map[string]bool{} // symbol -> platforms that have it
	for _, goos := range platforms {
		lines, err := dump(dir, goos)
		if err != nil {
			return nil, 0, err
		}
		for _, l := range lines {
			if has[l] == nil {
				has[l] = map[string]bool{}
			}
			has[l][goos] = true
		}
	}
	used := map[string]bool{}
	var syms []string
	for l := range has {
		syms = append(syms, l)
	}
	sort.Strings(syms)
	for _, l := range syms {
		var on, lacks []string
		for _, goos := range platforms {
			if has[l][goos] {
				on = append(on, goos)
			} else {
				lacks = append(lacks, goos)
			}
		}
		if len(lacks) == 0 {
			continue
		}
		key := strings.Join(on, ",") + " " + l
		exempted := false
		for _, re := range exemptions {
			if re.MatchString(l) {
				exempted = true
			}
		}
		if exempted {
			continue
		}
		used[key] = true
		if !allowed[key] {
			bad++
			report = append(report, strings.Join(lacks, ",")+" lacks "+l)
		}
	}
	for k := range allowed {
		if !used[k] {
			bad++
			report = append(report, "stale platform-only.txt entry: "+k)
		}
	}
	sort.Strings(report)
	return report, bad, nil
}

func readList(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}
