package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandBraces(t *testing.T) {
	cases := map[string][]string{
		"plain.go":                       {"plain.go"},
		"a_{x,y}.go":                     {"a_x.go", "a_y.go"},
		"widgets_{tree,table}*_test.go":  {"widgets_tree*_test.go", "widgets_table*_test.go"},
		"{a,b}/{c,d}.go":                 {"a/c.go", "a/d.go", "b/c.go", "b/d.go"},
		"unbalanced_{x.go":               {"unbalanced_{x.go"},
		"internal/cocoa/cocoa_<type>.go": {"internal/cocoa/cocoa_<type>.go"},
	}
	for in, want := range cases {
		got := expandBraces(in)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("expandBraces(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseMap(t *testing.T) {
	text := "# map\n\n" +
		"## Facade\n\n" +
		"Packages: `.`.\n\n" +
		"| Feature | Primary |\n" +
		"|---|---|\n" +
		"| App | `gowt.go`, `jobs.go` |\n\n" +
		"See [events.md](events.md#boundary) and [tooling](../tooling/j2go/).\n"
	sections, spans, links := parseMap(text)
	if len(sections) != 1 || sections[0].name != "Facade" {
		t.Fatalf("sections = %+v", sections)
	}
	if strings.Join(sections[0].packages, ",") != "." {
		t.Fatalf("packages = %v", sections[0].packages)
	}
	wantSpans := []string{"gowt.go", "jobs.go"}
	if strings.Join(spans, ",") != strings.Join(wantSpans, ",") {
		t.Fatalf("spans = %v", spans)
	}
	wantLinks := []string{"events.md#boundary", "../tooling/j2go/"}
	if strings.Join(links, ",") != strings.Join(wantLinks, ",") {
		t.Fatalf("links = %v", links)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range []string{
		"gowt.go", "quit_darwin.go", "quit_other.go",
		"swt/widgets_tree_darwin.go", "swt/widgets_table_darwin.go",
		"internal/cocoa/viewtable_manual.go",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCheckPaths(t *testing.T) {
	root := fixture(t)
	bad := checkPaths(root, []string{
		"gowt.go",                    // plain file
		"quit_{darwin,other}.go",     // braces
		"swt/widgets_<class>_*.go",   // placeholder glob
		"internal/cocoa/*_manual.go", // glob
		"swt/",                       // directory
		"Run",                        // API name: not a path
		"make gen-check",             // command: has a space
		"-",                          // none
		"missing.go",                 // bad
		"swt/missing_*.go",           // bad glob
		"no/dir/here.go",             // bad
	})
	want := []string{"missing.go", "swt/missing_*.go", "no/dir/here.go"}
	if len(bad) != len(want) {
		t.Fatalf("bad = %v, want %v", bad, want)
	}
	for i, w := range want {
		if !strings.HasPrefix(bad[i], w+":") {
			t.Fatalf("bad[%d] = %q, want prefix %q", i, bad[i], w)
		}
	}
}

func TestCheckLinks(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"events.md", "sub/page.md"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bad := checkLinks(dir, []string{
		"events.md#boundary",
		"sub/page.md \"a title\"",
		"#local-anchor-only",
		"https://example.com/page",
		"mailto:dev@example.com",
		"gone.md",
		"sub/gone/page.md",
	})
	want := []string{"gone.md", "sub/gone/page.md"}
	if len(bad) != len(want) {
		t.Fatalf("bad = %v, want %v", bad, want)
	}
	for i, w := range want {
		if !strings.HasPrefix(bad[i], w+":") {
			t.Fatalf("bad[%d] = %q, want prefix %q", i, bad[i], w)
		}
	}
}

func TestCheckCoverage(t *testing.T) {
	pkgs := map[string]bool{".": true, "swt": true, "rx": true, "look": true}
	sections := []section{
		{name: "Facade", packages: []string{"."}},
		{name: "swt", packages: []string{"swt", "gone"}},
	}
	bad := checkCoverage(pkgs, sections)
	want := []string{
		"Packages: gone is not a package of this repository",
		"package look belongs to no map section (add it to the Packages: list of a section)",
		"package rx belongs to no map section (add it to the Packages: list of a section)",
	}
	if strings.Join(bad, "\n") != strings.Join(want, "\n") {
		t.Fatalf("bad = %v, want %v", bad, want)
	}
	if b := checkCoverage(pkgs, []section{{name: "all", packages: []string{".", "swt", "rx", "look"}}}); len(b) != 0 {
		t.Fatalf("unexpected: %v", b)
	}
}
