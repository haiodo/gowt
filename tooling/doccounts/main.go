// Command doccounts checks that the test-count tables in README.md and docs/platforms.md
// equal the counts in tests/expected*.txt and tests/expected_jface*.txt.
// It prints the expected text and exits 1 on a mismatch.
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type counts map[string]int

func read(path string) counts {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer f.Close()
	c := counts{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		c[strings.SplitN(line, "\t", 2)[0]]++
	}
	return c
}

func row(label string, c counts) string {
	fail := fmt.Sprint(c["fail"])
	if c["flaky"] > 0 {
		fail += fmt.Sprintf(" (+%d flaky)", c["flaky"])
	}
	return fmt.Sprintf("| %s | %d | %s | %d |", label, c["pass"], fail, c["skip"])
}

// The fail cell may carry a note between the number and "(+N flaky)".
var (
	failCell = regexp.MustCompile(`^(\d+)(?:.*\(\+(\d+) flaky\))?`)
	number   = regexp.MustCompile(`^\d`)
)

func rowMatches(cells []string, c counts) bool {
	if cells[1] != fmt.Sprint(c["pass"]) || cells[3] != fmt.Sprint(c["skip"]) {
		return false
	}
	m := failCell.FindStringSubmatch(cells[2])
	if m == nil || m[1] != fmt.Sprint(c["fail"]) {
		return false
	}
	flaky := m[2]
	if flaky == "" {
		flaky = "0"
	}
	return flaky == fmt.Sprint(c["flaky"])
}

func main() {
	perOS := map[string]counts{
		"macOS":   read("tests/expected.txt"),
		"Windows": read("tests/expected_windows.txt"),
		"Linux":   read("tests/expected_linux.txt"),
	}
	jm, jw, jl := read("tests/expected_jface.txt"), read("tests/expected_jface_windows.txt"), read("tests/expected_jface_linux.txt")
	mac := perOS["macOS"]
	total := mac["pass"] + mac["fail"] + mac["flaky"] + mac["skip"]

	var bad []string
	for _, doc := range []string{"README.md", "docs/platforms.md"} {
		b, err := os.ReadFile(doc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		text := string(b)
		seen := map[string]bool{}
		for _, line := range strings.Split(text, "\n") {
			cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
			if len(cells) < 4 {
				continue
			}
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			for name, c := range perOS {
				if strings.HasPrefix(cells[0], name) && number.MatchString(cells[1]) {
					seen[name] = true
					if !rowMatches(cells, c) {
						bad = append(bad, fmt.Sprintf("%s: %s\n  expected %s", doc, line, row(cells[0], c)))
					}
				}
			}
		}
		for name, c := range perOS {
			if !seen[name] {
				bad = append(bad, fmt.Sprintf("%s: no table row for %s\n  expected %s", doc, name, row(name, c)))
			}
		}
		if want := fmt.Sprintf("%d tests in all", total); !strings.Contains(text, want) {
			bad = append(bad, fmt.Sprintf("%s: missing %q (pass+fail+flaky+skip of expected.txt)", doc, want))
		}
		wantJ := fmt.Sprintf("%d pass on macOS, %d pass and %d fail on Linux and Windows", jm["pass"], jl["pass"], jl["fail"])
		if doc != "README.md" {
			wantJ = fmt.Sprintf("%d pass, %d skip on macOS; %d pass, %d fail, %d skip on Linux and Windows", jm["pass"], jm["skip"], jl["pass"], jl["fail"], jl["skip"])
		}
		if jl["pass"] != jw["pass"] || jl["skip"] != jw["skip"] || jl["fail"] != jw["fail"] {
			bad = append(bad, "tests/expected_jface_linux.txt and expected_jface_windows.txt differ; the docs sentence assumes they are equal")
		} else if !strings.Contains(text, wantJ) {
			bad = append(bad, fmt.Sprintf("%s: missing JFace text %q", doc, wantJ))
		}
	}
	if len(bad) > 0 {
		fmt.Fprintln(os.Stderr, "test counts in the docs disagree with tests/expected*.txt:")
		for _, s := range bad {
			fmt.Fprintln(os.Stderr, s)
		}
		os.Exit(1)
	}
}
