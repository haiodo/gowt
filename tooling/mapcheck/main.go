// Command mapcheck validates docs/code-map.md, the gowt feature-to-code map.
//
// Three checks, all against the repository the command runs in:
//
//  1. Repository paths named in map tables exist. Cells may use `*` globs,
//     {a,b} brace lists and <name> placeholders; a glob must match at least
//     one file, everything else must stat.
//  2. Markdown links in the map resolve to an existing file or directory
//     (anchors and link titles are ignored).
//  3. Every Go package - the union of `go list ./...` over darwin, windows
//     and linux - is listed in the curated `Packages:` metadata of at least
//     one map section, and every listed package exists.
//
// `go list` only feeds the set of packages to check; map rows and the
// Packages: lists are curated by hand in the map itself.
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var spanRE = regexp.MustCompile("`([^`]+)`")
var linkRE = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

type section struct {
	name     string
	packages []string
}

// parseMap splits the map into sections (## headings) with their curated
// Packages: lists, the backtick path spans of table rows, and the link targets.
func parseMap(text string) (sections []section, spans, links []string) {
	var sec *section
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			sections = append(sections, section{name: strings.TrimSpace(line[3:])})
			sec = &sections[len(sections)-1]
		}
		if sec != nil && strings.HasPrefix(line, "Packages:") {
			for _, m := range spanRE.FindAllStringSubmatch(line, -1) {
				sec.packages = append(sec.packages, m[1])
			}
		}
		if strings.HasPrefix(line, "|") {
			for _, m := range spanRE.FindAllStringSubmatch(line, -1) {
				spans = append(spans, m[1])
			}
		}
	}
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		links = append(links, m[1])
	}
	return sections, spans, links
}

// expandBraces expands the first {a,b,...} group (no nesting) of a path pattern.
func expandBraces(pattern string) []string {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		return []string{pattern}
	}
	close := strings.IndexByte(pattern[open:], '}')
	if close < 0 {
		return []string{pattern}
	}
	close += open
	var out []string
	for _, alt := range strings.Split(pattern[open+1:close], ",") {
		out = append(out, expandBraces(pattern[:open]+alt+pattern[close+1:])...)
	}
	return out
}

var placeholderRE = regexp.MustCompile(`<[^<>]+>`)

// checkPaths stats or globs every table span that looks like a repository
// path: no whitespace, and a directory component or a .go suffix. Spans such
// as `Run` or `make gen-check` are API or command names, not paths.
func checkPaths(root string, spans []string) []string {
	var bad []string
	for _, span := range spans {
		if span == "-" || strings.ContainsAny(span, " \t") {
			continue
		}
		if !strings.Contains(span, "/") && !strings.HasSuffix(span, ".go") {
			continue
		}
		pattern := placeholderRE.ReplaceAllString(span, "*")
		for _, p := range expandBraces(pattern) {
			if strings.ContainsAny(p, "*?[") {
				matches, _ := filepath.Glob(filepath.Join(root, p))
				if len(matches) == 0 {
					bad = append(bad, fmt.Sprintf("%s: matches no file", span))
				}
				continue
			}
			if _, err := os.Stat(filepath.Join(root, p)); err != nil {
				bad = append(bad, fmt.Sprintf("%s: %v", span, err))
			}
		}
	}
	return bad
}

// checkLinks resolves link targets relative to the map's directory, ignoring
// anchors and titles. Absolute URIs (https://, mailto:, ...) are not
// repository content; their availability is not checked.
func checkLinks(mapDir string, links []string) []string {
	var bad []string
	for _, link := range links {
		if u, err := url.Parse(link); err == nil && u.IsAbs() {
			continue
		}
		target := strings.SplitN(link, "#", 2)[0]
		if fields := strings.Fields(target); len(fields) > 0 {
			target = fields[0]
		}
		if target == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(mapDir, target)); err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", link, err))
		}
	}
	return bad
}

// listPackages returns the union of `go list ./...` package directories over
// the three ports, relative to root.
func listPackages(root string) (map[string]bool, error) {
	pkgs := map[string]bool{}
	for _, goos := range []string{"darwin", "windows", "linux"} {
		cmd := exec.Command("go", "list", "-f", "{{.Dir}}", "./...")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH=amd64")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("go list GOOS=%s: %w", goos, err)
		}
		for _, dir := range strings.Fields(string(out)) {
			rel, err := filepath.Rel(root, dir)
			if err != nil {
				return nil, err
			}
			pkgs[rel] = true
		}
	}
	return pkgs, nil
}

// checkCoverage reports packages missing from every section's Packages: list
// and listed packages that are not Go packages of the repository.
func checkCoverage(pkgs map[string]bool, sections []section) []string {
	var bad []string
	claimed := map[string]bool{}
	for _, sec := range sections {
		for _, p := range sec.packages {
			if !pkgs[p] {
				bad = append(bad, fmt.Sprintf("Packages: %s is not a package of this repository", p))
			}
			claimed[p] = true
		}
	}
	var missing []string
	for p := range pkgs {
		if !claimed[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	for _, p := range missing {
		bad = append(bad, fmt.Sprintf("package %s belongs to no map section (add it to the Packages: list of a section)", p))
	}
	return bad
}

func main() {
	mapFile := flag.String("map", "docs/code-map.md", "the code map to validate")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	b, err := os.ReadFile(*mapFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sections, spans, links := parseMap(string(b))

	var bad []string
	bad = append(bad, checkPaths(root, spans)...)
	bad = append(bad, checkLinks(filepath.Dir(*mapFile), links)...)
	pkgs, err := listPackages(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	bad = append(bad, checkCoverage(pkgs, sections)...)

	if len(bad) > 0 {
		fmt.Fprintf(os.Stderr, "%s is stale:\n", *mapFile)
		for _, s := range bad {
			fmt.Fprintln(os.Stderr, "  "+s)
		}
		os.Exit(1)
	}
	fmt.Printf("%s: %d sections, %d path spans, %d links, %d packages - all good\n",
		*mapFile, len(sections), len(spans), len(links), len(pkgs))
}
